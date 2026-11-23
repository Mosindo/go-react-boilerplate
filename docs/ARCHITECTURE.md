# Hearth — Architecture

Hearth is a free dating app built on the `go-react-saas` foundation: one Go
API (Gin, pgx, PostgreSQL, no ORM) and one Expo / React Native client (iOS,
Android, web). This document explains how the code is organized and why.

## 1. Repository layout

```text
.
├─ apps/mobile/                 Expo app (TypeScript)
│  ├─ App.tsx                   providers + navigator + global feedback
│  ├─ e2e/web-critical-flow.js  Playwright E2E (two users, full journey)
│  ├─ scripts/                  API smoke test, phone launcher, PNG helper
│  └─ src/
│     ├─ api/                   the only place that talks to the network
│     ├─ hooks/                 auth state, React Query hooks, realtime sync
│     ├─ navigation/            auth stack / onboarding / tabs + stacks
│     ├─ screens/<feature>/     route-level UI
│     ├─ components/            feature components (cards, photo grid, …)
│     ├─ shared/{ui,layout,feedback}  design system and app chrome
│     ├─ store/                 token persistence (SecureStore)
│     └─ utils/                 formatting, media/location helpers, errors
├─ services/api/
│  ├─ cmd/api                   composition root + integration tests
│  ├─ cmd/seed                  demo data (development only)
│  └─ internal/
│     ├─ platform/              cross-cutting infrastructure
│     └─ features/              business domains
├─ docker-compose.yml           postgres + api + mailpit (dev mail catcher)
├─ scripts/qa.js                local quality gate (mirrors CI)
└─ .github/workflows/ci.yml     api, mobile, e2e and docker jobs
```

## 2. Backend

### Platform (`internal/platform`)

| Package      | Role |
|--------------|------|
| `config`     | Environment loading and validation (refuses weak `JWT_SECRET`, production without SMTP). |
| `db`         | Shared `pgxpool`, embedded SQL migrations tracked in `schema_migrations` (advisory-locked). |
| `middleware` | Security headers, CORS, request id/log/metrics, body limit, `Authenticator` (HS256, `exp` required, session must be active). |
| `httpx`      | Uniform error payload `{"error","code"}`, UUID/limit parsing. |
| `ratelimit`  | In-memory token buckets for auth, swipes, messages, uploads, safety. |
| `storage`    | Object storage interface; local-disk implementation with strict keys. |
| `media`      | HMAC-signed, expiring photo URLs (key derived from `JWT_SECRET`). |
| `mailer`     | SMTP transport; log transport outside production. |
| `realtime`   | WebSocket hub (`GET /realtime`), per-user fan-out. |
| `logger`     | Structured logs with secret redaction. |

### Features (`internal/features`)

Every feature has `handler.go` (HTTP only) → `service.go` (business rules) →
`repository.go` (SQL) plus `model.go` and `routes.go`. Features talk to each
other through small interfaces declared by the consumer (for example
`matching.CardLoader`, `chat.Notifier`), wired in `cmd/api/main.go`.

| Feature         | Responsibility |
|-----------------|----------------|
| `auth`          | Register, login, refresh rotation, logout, password reset codes, password change, account deletion. |
| `profiles`      | Profile, preferences, privacy settings, approximate location, interests catalog, batched public cards. |
| `photos`        | Upload / replace / reorder / delete, image validation and re-encoding, signed media serving. |
| `matching`      | Discovery query, swipes, matches, unmatch. |
| `chat`          | Match-scoped conversations, messages, read receipts. |
| `safety`        | Blocks and reports (with auto-suspension threshold). |
| `notifications` | In-app notifications (collapsed per conversation), unread counts. |

### Data model

```text
users ─┬─ profiles (1:1)      first name, birthdate, gender, bio, city,
       │                      snapped lat/lon, discoverable, hide_distance
       ├─ preferences (1:1)   interested_in[], age range, max distance
       ├─ photos (≤6)         position 0..5 (deferred unique), storage_key
       ├─ user_interests ── interests (catalog)
       ├─ swipes              PK(swiper, target), like|pass
       ├─ matches             (user_a < user_b) unique, → conversation
       ├─ conversation_participants ── conversations ── messages
       ├─ blocks              PK(blocker, blocked)
       ├─ reports             reporter (SET NULL on delete), reason, status
       ├─ notifications
       ├─ sessions            hashed refresh tokens
       └─ password_resets     hashed one-time codes, attempts
```

All user-owned rows cascade on account deletion. Conversations are deleted
explicitly (they belong to two people). Reports keep a `NULL` reporter for
moderation history.

### Key flows

- **Discovery** — one SQL query applies *mutual* eligibility (each side
  matches the other's genders and age range, distance within both limits),
  excludes swiped, blocked (either way), hidden, suspended and incomplete
  profiles, then ranks by shared interests, proximity and recent activity.
  Cards are loaded in three batched queries (profiles, photos, interests).
- **Swipe → match** — in one transaction guarded by a per-pair advisory
  lock: re-check eligibility, insert the swipe (`PK` prevents duplicates),
  and on a reciprocal like create conversation + participants + match.
  Simultaneous likes yield exactly one match (covered by a test).
- **Chat** — every operation resolves membership first (participant + live
  match + no block); outsiders get `404`. Messages use keyset pagination.
  After commit, events are pushed to both participants over WebSocket.
- **Realtime** — the client authenticates with its first frame (no token in
  URLs). Events: `message.created`, `conversation.read`, `match.created`,
  `match.removed`, `notification.created`. The hub is in-process; for
  several replicas, back `Publish` with PostgreSQL `LISTEN/NOTIFY` or Redis.

### Privacy & security decisions

- Coordinates are snapped to 0.01° (~1 km) before storage, never returned
  (not even to the owner); others see a whole-km distance (min 1 km) unless
  hidden. No location history is kept.
- Photos: content sniffed (client MIME ignored), dimensions checked before
  decoding (decompression bombs), re-encoded to JPEG (EXIF/GPS stripped),
  ≤ 1600 px, served only through signed expiring URLs with `nosniff`.
- Auth: bcrypt (cost 12), password policy, constant-time login for unknown
  emails, refresh token rotation, sessions checked on every request so
  logout / reset / deletion are immediate.
- Abuse: rate limits, report → automatic block, 3 distinct open reports →
  account hidden from discovery and profile views pending review.
- Blocking never discloses itself: the other side sees the same event as an
  unmatch.

## 3. Frontend

- **Network** only in `src/api` (`apiRequest` adds the token, refreshes it
  once on `401`, normalizes errors). Screens use React Query hooks from
  `src/hooks/queries.ts`.
- **Realtime** `useRealtime` keeps caches in sync (inserts messages, read
  receipts, invalidates matches/notifications) and reconnects with backoff.
- **Navigation** signed-out stack → onboarding (until the profile has the
  required fields) → tabs (Découvrir, Matchs, Activité, Profil) with chat,
  profile detail, settings, report, … as stack screens.
- **Design system** `shared/ui/tokens.ts`: warm dark theme, AA contrast,
  pill buttons, glyph icons (no icon-font dependency). Cross-platform
  `ConfirmDialog` (React Native's `Alert` is a no-op on web).
- **Images** `expo-image` with memory+disk cache keyed by photo id;
  uploads are resized/compressed on device before sending.
- **Tokens** in SecureStore (Keychain/Keystore); memory-only on web.

## 4. Extending

- New domain → new folder under `internal/features` with the five files,
  routes registered in `cmd/api/main.go`, migration `NNN_*.sql`.
- Ranking → replace the `ORDER BY` of `candidateSQL` in
  `features/matching/repository.go`; eligibility rules stay untouched.
- Object storage → implement `storage.Storage` (S3, GCS) and wire it in
  `main.go`.
- Push notifications → add an Expo push transport next to
  `notifications.Service.Notify`.
