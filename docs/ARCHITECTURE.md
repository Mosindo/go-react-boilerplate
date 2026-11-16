# Lueur — Architecture

This document explains how the repository is organized so developers and AI agents can navigate it,
understand responsibilities and extend it without breaking its boundaries.

## 1. Monorepo

| Path | Content |
| --- | --- |
| `services/api` | Go API (Gin, pgxpool, explicit SQL) |
| `apps/mobile` | Expo app (iOS, Android, web) |
| `docker-compose.yml` | Postgres + API for local runs |
| `scripts/qa.sh` | Local quality gate |
| `.github/workflows/ci.yml` | CI (api, mobile, e2e, docker) |

## 2. Backend

```text
services/api/
├─ cmd/api/          main.go (config, pool, migrations, HTTP server), router.go (composition root)
├─ cmd/seed/         demo data (refuses production)
└─ internal/
   ├─ platform/      cross-cutting infrastructure, no domain logic
   │  ├─ config/     env loading and validation
   │  ├─ db/         pool + tracked, transactional, advisory-locked migrations
   │  ├─ middleware/ JWT auth (+ session check), security headers, CORS, body limit, logging
   │  ├─ httpx/      error envelope, pagination and UUID parsing
   │  ├─ ratelimit/  in-memory token buckets + Gin middleware
   │  ├─ storage/    Storage interface + local disk implementation
   │  ├─ imaging/    upload validation and JPEG re-encoding
   │  ├─ media/      HMAC-signed, expiring media URLs
   │  ├─ mailer/     SMTP / dev log / disabled mailers
   │  ├─ realtime/   WebSocket hub, single-use tickets
   │  └─ logger/     structured handler logs with secret redaction
   └─ features/
      ├─ auth/          accounts, sessions, password reset, account deletion
      ├─ profiles/      profile, preferences, interests, location, public cards
      ├─ photos/        upload, order, replace, delete, signed serving
      ├─ discovery/     candidate generation (SQL) + Ranker (Go)
      ├─ matching/      swipes, matches, unmatch
      ├─ chat/          messages, read receipts
      ├─ safety/        blocks, reports
      └─ notifications/ activity feed
```

Every feature has `model.go`, `repository.go` (all SQL), `service.go` (business rules),
`handler.go` (HTTP binding, status mapping) and `routes.go`. Handlers never touch SQL; services never
touch HTTP. Cross-feature dependencies go through small interfaces declared by the consumer
(e.g. `matching.CardLoader`, `chat.Notifier`, `realtime.Publisher`) and are wired in `cmd/api/router.go`.

### Key flows

- **Swipe → match** (`matching.PGRepository.RecordSwipe`): one transaction, advisory lock on the
  unordered pair, `INSERT … ON CONFLICT DO NOTHING` on `swipes`, reciprocal check, canonical
  `matches(user_a_id < user_b_id)` insert. Notifications and realtime events are emitted after commit.
- **Discovery** (`discovery`): SQL applies hard rules (mutual gender and age preferences, mutual
  distance with a bounding-box prefilter, visibility, photo required, not swiped, not blocked, demo
  accounts outside production only) and returns a pool of 200 recent candidates; `HeuristicRanker`
  orders them; `profiles.Service.Cards` hydrates the page in 3 queries.
- **Chat**: membership is checked on every call (active match, no block). Messages use keyset
  pagination (`created_at, id`) and an optional `client_message_id` unique per sender for
  idempotent retries.
- **Realtime**: `POST /realtime/ticket` (authenticated) → `GET /realtime?ticket=` upgrades to a
  WebSocket. The hub is in-process; scale-out requires a pub/sub fan-out.

### Data model

`users` 1—1 `profiles`, 1—1 `preferences`, 1—n `photos`, n—n `interests` (via `user_interests`),
1—n `sessions`, `password_resets`. `swipes(swiper, target)` and `blocks(blocker, blocked)` are pair
tables. `matches` (one row per pair) own `messages` and are the conversations. `notifications`
reference an actor and a match. `reports` survive account deletion with nulled references.

## 3. Frontend

```text
apps/mobile/src/
├─ api/          client.ts (only place calling fetch; token refresh), typed modules
├─ hooks/        useAuth (session), queries.ts (React Query keys/hooks/cache helpers),
│                useRealtime (WebSocket → cache), deckEvents
├─ navigation/   RootNavigator: booting → auth stack → onboarding gate → app stack + tabs
├─ screens/      route-level UI only
├─ components/   domain components (SwipeDeck, ProfileCardView, PhotoGrid, SafetySheet…)
├─ ui/           design-system primitives
├─ theme/        tokens (colors, spacing, radii, typography, shadows)
├─ store/        token persistence (SecureStore / sessionStorage on web)
└─ utils/        formatting, error translation, photo picking, location
```

Rules: screens never call `fetch`; server state lives in React Query; realtime events update the
cache (`upsertMessage`, `markOwnMessagesRead`, invalidations) rather than component state.

## 4. Adding a feature

1. Add a migration `services/api/internal/platform/db/migrations/00N_name.sql` (never edit an applied one).
2. Create `internal/features/<name>/` with the five files.
3. Wire it in `cmd/api/router.go`; add integration tests in `cmd/api`.
4. Add a typed client in `apps/mobile/src/api`, hooks in `hooks/queries.ts`, then screens.
5. Run `./scripts/qa.sh` and the E2E suite.
