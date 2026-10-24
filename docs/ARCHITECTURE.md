# Architecture

Alba is a monorepo: a Go REST + WebSocket API, a PostgreSQL database and an Expo (React Native / web) client.

```text
.
├─ apps/mobile/            Expo client (TypeScript)
│  ├─ App.tsx              providers: theme, query cache, dialogs/toasts, auth, realtime
│  ├─ e2e/                 Playwright browser end-to-end tests
│  └─ src/
│     ├─ api/              HTTP client (token refresh, error mapping), typed endpoints
│     ├─ auth/             session state, secure token storage
│     ├─ realtime/         WebSocket provider, React Query cache updates
│     ├─ features/         auth, onboarding, discover, chat, profile, notifications, safety
│     ├─ navigation/       stacks and tabs
│     ├─ theme/, ui/       design system (tokens, Button, TextField, Photo, Dialog, Toast…)
│     └─ lib/              pure helpers (dates, validation, photo upload, location)
├─ services/api/
│  ├─ cmd/api/             HTTP server entrypoint + integration tests
│  ├─ cmd/seed/            demo data (refuses to run in production)
│  └─ internal/
│     ├─ platform/         config, db (migrations), middleware, ratelimit, storage, mailer,
│     │                    realtime (hub + WebSocket), geo, httpx, logger
│     └─ features/         auth, profiles, photos, matching, chat, safety, notifications
├─ scripts/smoke.mjs       API smoke test
└─ docker-compose.yml      PostgreSQL + API
```

## Backend layering

Every feature follows `routes.go -> handler.go -> service.go -> repository.go` (+ `model.go`).

- **handler**: binds/validates the HTTP shape, maps domain errors to status codes. No SQL, no business rules.
- **service**: business rules, validation, orchestration, side effects (notifications, realtime events).
- **repository**: all SQL (pgx, no ORM). Interfaces are defined next to their consumer.
- **platform**: cross-cutting code only. Features never import each other except through small interfaces
  (`CardLoader`, `notifications.Notifier`, `realtime.Publisher`) wired in `cmd/api/main.go`.

A single `pgxpool.Pool` and a single Gin server are shared by everything.

## Data model

| Table | Purpose |
| --- | --- |
| `users`, `sessions`, `password_resets` | credentials (bcrypt), rotating refresh sessions, hashed recovery codes |
| `profiles`, `preferences` | public profile data and discovery preferences (1-1 with a user) |
| `interests`, `user_interests` | interest catalog and user selection (max 10) |
| `photos` | up to 6 per user, ordered by `position` (0 = main), deferred unique `(user_id, position)` |
| `swipes` | one decision per `(actor, target)`; first decision is final |
| `matches` | one row per pair with `user_low < user_high` and a unique constraint |
| `conversations`, `conversation_participants`, `messages` | one conversation per match; `hidden_at` implements local deletion; `read_at` is the read receipt |
| `blocks`, `reports` | safety; reports are kept for moderation |
| `notifications` | persisted in-app notifications |

Everything references `users(id) ON DELETE CASCADE`, so deleting an account removes all of its data;
photo files are deleted from storage by the auth service.

## Key decisions

- **Eligibility is defined once** (`matching/repository.go`): the discovery feed and the swipe authorization use the same
  SQL, so a client cannot act on a profile the feed would not show. Rules: both profiles complete and active, mutual gender
  interest, mutual age range, mutual max distance (only when both positions are known), no block either way.
- **Matches are race-free**: a per-pair advisory lock serializes simultaneous likes; the canonical pair ordering and unique
  constraints make duplicates impossible.
- **Ranking is a single `ORDER BY`** (`discoverQuery`): people who already liked you, shared interests, recent activity.
  Replace it (or move it to a service) to evolve the algorithm.
- **Location privacy**: coordinates are rounded to ~1 km on write, never returned, and other users only see a distance in
  5 km buckets; users can hide the distance, clear their position, or leave discovery.
- **Photos**: content-sniffed (JPEG/PNG/WebP), size and pixel limits before decoding, EXIF orientation applied, everything
  re-encoded to JPEG (metadata stripped), full + thumbnail stored under random names, served only to authenticated viewers
  who may see the owner (not blocked, discoverable or matched). Ids are immutable so responses are cacheable.
- **Auth**: HS256 JWT access tokens (15 min, `uid` + `sid`), refresh tokens stored hashed and rotated on every use.
  Login does constant-work comparisons; password recovery never reveals whether an email exists and locks a code after
  5 wrong attempts.
- **Realtime**: `GET /ws` with an `auth` first frame. Events (`message.new`, `messages.read`, `match.new`, `match.removed`,
  `notification.new`) are pushed to the in-process hub; REST stays the source of truth and clients resync after reconnects.
  The hub is per process: use LISTEN/NOTIFY or a broker behind `realtime.Publisher` to scale horizontally.
- **Rate limiting** is in-memory per process (login, recovery, uploads, swipes, messages, reports).
- **Migrations** are embedded, applied once in filename order inside transactions, serialized by an advisory lock.

## Client

- React Query is the server-state cache; the WebSocket provider writes into it, so screens never poll.
- The HTTP client refreshes the access token once on `401` (single-flight) and ends the session only when the refresh token
  is explicitly rejected. Only the refresh token is persisted (SecureStore; localStorage on web).
- Photos are private, so images are loaded with an `Authorization` header (blob fetch on web).
- UI code never calls `fetch` directly; everything goes through `src/api`.
