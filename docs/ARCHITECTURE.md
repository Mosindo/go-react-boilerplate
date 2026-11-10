# Lueur — Architecture

## Overview

```text
 Expo app (iOS / Android / web)                      Go API (Gin, single HTTP server)
 ┌──────────────────────────────┐   REST + JSON      ┌─────────────────────────────────────┐
 │ features/* screens           │ ─────────────────▶ │ middleware: security headers, CORS,  │
 │ hooks (React Query, session, │                    │ request id, logging, rate limiting,  │
 │ realtime, location)          │ ◀── WebSocket ──── │ body limits, JWT auth                │
 │ api/ (typed client, refresh) │   (ticket auth)    │ features: handler → service → repo   │
 └──────────────────────────────┘                    │ platform: db, storage, media, mailer │
                                                     └──────────────┬──────────────────────┘
                                                                    │ pgxpool (one pool)
                                                             PostgreSQL 16   +  photo volume
```

## Backend (`services/api`)

| Package | Responsibility |
| --- | --- |
| `cmd/api` | Loads config, connects (with retry), migrates, wires the router, graceful shutdown. |
| `cmd/seed` | Demo data generator (flagged `is_demo`, refuses production). |
| `internal/app` | Composition root shared by the binary and the HTTP integration tests. |
| `features/auth` | Register, login, refresh rotation, logout, password reset/change, account deletion. |
| `features/profiles` | Own profile, validation, preferences, privacy, coarse location, interests, public cards. |
| `features/photos` | Upload pipeline (sniff → bounds → decode → resize → JPEG), order/replace/delete, signed media serving. |
| `features/discovery` | Viewer context, SQL eligibility, ranking (`ranking.go`), race-free swipe → match. |
| `features/matches` | Match list (conversations), messages, read receipts, unmatch, realtime fan-out. |
| `features/safety` | Blocks, reports, auto-review threshold. |
| `features/notifications` | In-app notifications (collapsed per conversation), unread counts. |
| `features/realtime` | WebSocket endpoint and ticket issuance. |
| `platform/*` | `config`, `db` (tracked migrations), `middleware`, `httpx` (errors, ids, limits), `authtoken` (JWT), `ratelimit`, `storage` (local disk behind an interface), `media` (signed URLs), `mailer` (SMTP / dev log), `realtime` (hub), `geo`, `cursor`, `logger`. |

Feature modules depend on each other only through small interfaces declared by
the consumer (`matches.Notifier`, `discovery.MatchListener`,
`safety.MatchRemover`, `auth.AccountHooks`), wired in `internal/app`.

### Key flows

**Swipe → match.** `POST /discovery/swipes` runs one transaction: advisory lock
on the unordered pair → eligibility check (same SQL predicate as the deck) →
`INSERT … ON CONFLICT DO NOTHING` into `swipes` (duplicate → 409) → if like and
reciprocal like exists, insert the ordered pair into `matches`. After commit,
`matches.MatchCreated` publishes `match.new` and creates notifications for both.

**Discovery.** Bounding-box prefilter on the coarse coordinates (indexed),
mutual gender/age preferences, active & discoverable profile with ≥ 1 photo, no
block, not already swiped, not in the client's current deck. Up to 200
candidates are scored in Go (reciprocity, proximity, activity, shared
interests, jitter) and the top N are hydrated as cards in one query (photos and
interests aggregated with `json_agg`, no N+1).

**Chat.** Every query is filtered by the caller's id (`user_a_id = $me OR
user_b_id = $me`), so non-participants get 404. Messages are keyset-paginated
(`created_at, id`). Sending locks the match row (it must still exist: unmatch
and block delete it) and updates `last_activity_at` used to order the list.

**Realtime.** `POST /realtime/ticket` → single-use ticket (60 s) → WebSocket.
The process-local hub fans out events per user; the client re-syncs over REST
on every (re)connection, so missed events are never lost.

**Photos.** The raw upload is never stored or served. Only the re-encoded JPEG
is written under a random key; clients receive `/media/<key>?exp=&sig=` URLs
signed with a key derived from `JWT_SECRET`, aligned on hourly windows for
cacheability.

## Mobile (`apps/mobile`)

| Folder | Responsibility |
| --- | --- |
| `src/api` | `client.ts` (base URL, timeouts, typed errors, single-flight refresh, session events), `endpoints.ts` (every REST call), `tokenStore.ts` (SecureStore / sessionStorage on web), `types.ts`. |
| `src/hooks` | `useAuth` (session lifecycle), `queries.ts` (React Query hooks and cache helpers), `useRealtime` (WebSocket ↔ cache, celebrations), `useLocation`. |
| `src/navigation` | Root: Auth stack → Onboarding (until profile, preferences and a photo exist) → Main stack (tabs + chat, profile detail, settings). |
| `src/features` | Screens grouped by domain. |
| `src/theme`, `src/ui` | Design tokens (light/dark) and reusable primitives. |

Screens never call `fetch`: they use hooks or `src/api/endpoints.ts`.

## Testing strategy

- Go unit tests next to the code they cover (validation, ranking, geo, JWT, photo processing…).
- HTTP integration tests (`internal/app/app_integration_test.go`) on a real Postgres,
  each test isolated in its own random geographic area.
- Playwright E2E (`apps/mobile/e2e`) on the exported web build for the critical journey.
- `scripts/smoke.sh` for deployed environments.
