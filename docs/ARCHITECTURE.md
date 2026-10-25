# Architecture

Lueur is a monorepo: a Go API (`services/api`), an Expo mobile client (`apps/mobile`) and Docker infrastructure (`infra`).
The REST/WebSocket contract lives in [API.md](API.md).

## Backend

```text
cmd/api         entry point: config -> pool -> migrations -> http.Server (graceful shutdown)
cmd/seed        demo accounts (is_demo = true, refuses APP_ENV=production)
internal/app    wiring: builds repositories, services, handlers and routes; integration tests live here
internal/features/<name>/{model,repository,service,handler,routes}.go
internal/platform/{config,db,httpx,logger,mailer,middleware,realtime,testutil}
```

Layering is strict: `handler -> service -> repository`. Handlers bind/validate transport concerns and map errors to
`{error, code}` responses; services hold business rules; repositories own all SQL (explicit queries, `pgxpool`, no ORM).
Features depend on each other only through small interfaces declared by the consumer (for example `ProfileReader`,
`NotificationPusher`), wired in `internal/app`.

| Feature | Responsibility |
|---|---|
| `auth` | register/login, refresh rotation, logout, password reset by e-mailed code, change password, account deletion |
| `profiles` | profile, preferences, interests catalog, coarse location, public profile views and summaries |
| `photos` | upload pipeline (decode, orient, resize, re-encode), ordering, replacement, authorized image serving |
| `discovery` | candidate feed (mutual preferences, ranking) and swipes (atomic like/pass/match) |
| `matches` | match list and unmatch |
| `chat` | conversations, messages, read state, local deletion, WebSocket endpoint |
| `safety` | blocks and reports |
| `notifications` | match / message notifications, read state, realtime push |

### Platform notes

* `middleware.RequireUser` validates the HS256 token strictly (algorithm, mandatory `exp`, claims) and checks the session in the
  database, so revoking a session or deleting an account is immediate.
* `middleware.RateLimiter` is a fixed-window in-memory limiter keyed by IP (before auth) or user (after auth).
* `realtime.Hub` is an in-process fan-out keyed by user id. Features publish through the `realtime.Publisher` interface.
  Multi-instance deployments need a shared bus behind that interface.
* `db.RunMigrations` applies embedded SQL files once each (table `schema_migrations`, advisory lock, one transaction per file).
* `testutil.NewPool` gives each test package its own temporary PostgreSQL schema.

## Data model

```text
users ──< sessions, password_resets
users ──1 profiles ──< user_interests >── interests
users ──1 preferences
users ──< photos (position 0..5, unique per user, deferred constraint for reordering)
users ──< swipes (from,to unique, like|pass)        blocks (blocker,blocked)        reports (reporter,reported,reason,status)
matches (user_a < user_b, unique pair) ──1 conversations ──< conversation_participants (hidden_at = history cleared marker)
conversations ──< messages (body 1..2000, read_at)
users ──< notifications (match|message, partial unique index coalesces unread message notifications per conversation)
```

All user-owned tables reference `users(id) ON DELETE CASCADE`: deleting the user removes every trace.

Indexes (besides primary keys and uniques): `users.email` (unique), `messages (conversation_id, created_at, id)`,
partial `messages (conversation_id, sender_id) WHERE read_at IS NULL`, `conversation_participants (conversation_id, user_id)` (PK) and
`(user_id, conversation_id)`, `notifications (user_id, created_at, id)` + unread partial index, `swipes (to_user_id, action)`,
`matches (user_b, created_at)` / `(user_a, created_at)`, `profiles (gender, birth_date) WHERE discoverable`, `blocks (blocked_id)`,
`reports (reported_id, created_at)` / `(status, created_at)`.

## Discovery query

`discovery.candidateFrom` is the single definition of eligibility, used by both the feed and swipe validation:
discoverable profile with at least one photo, mutual gender interest, mutual age range, mutual distance limits (haversine in SQL on
the coarse coordinates), nobody blocked anybody. The feed adds "not already swiped" and orders by `feedScore`, then a per-viewer
stable pseudo-random tie-break. Public profile details (interests, photos, distance) are loaded for the whole page in a fixed number of
queries.

## Frontend

`apps/mobile/src`: `api/` (typed client, token refresh, one module per domain), `features/<domain>` (screens, hooks),
`navigation/`, `realtime/` (WebSocket service feeding the query cache), `shared/` (UI kit, feedback, formatting), `theme/` (tokens,
light and dark). UI code never calls `fetch` directly; images are loaded through `AuthImage`, which sends the bearer header.
