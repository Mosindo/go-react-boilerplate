# Architecture

Lumen is a free dating app: a Go API (Gin + PostgreSQL, no ORM) and an Expo / React Native client.

## Repository layout

```text
apps/mobile/            Expo (React Native, TypeScript) client
services/api/           Go API
  cmd/api/              composition root (config -> migrations -> router -> HTTP server)
  cmd/seed/             demo data (refuses to run in production)
  internal/app/         wires platform + features into one Gin router (+ end-to-end tests)
  internal/platform/    shared concerns: config, db (+migrations), middleware, httpx, realtime,
                        storage, mailer, logger, errors
  internal/features/    domain modules (see below)
docs/                   this file, API.md (contract), agents/
scripts/smoke-api.js    black-box journey against a running stack
archive/legacy-saas/    previous SaaS demo code, not built (see its README)
```

## Backend

`handler -> service -> repository`, one folder per feature, each with `handler.go`,
`service.go`, `repository.go`, `model.go`, `routes.go`.

| Feature | Responsibility |
|---|---|
| `auth` | register/login/refresh/logout, password recovery, `/me`, account deletion |
| `profiles` | profile, preferences/privacy, interests, location, profile hydration |
| `photos` | validated upload, re-encoding, ordering, protected delivery |
| `discovery` | candidate selection, swipes, matches, unmatch |
| `chat` | conversations, messages, read state, local deletion |
| `safety` | blocks and reports |
| `notifications` | match / message notifications (stored + pushed) |

Cross-feature calls go through small interfaces defined by the consumer
(`discovery.Notifier`, `chat.Notifier`, `photos.Visibility`, ...) and are wired in `internal/app`.

Errors: services return `platformerrors.AppError` (status + safe message); `httpx.Respond` maps
them to JSON and turns everything else into a logged generic 500.

### Data model

`users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`,
`photos`, `swipes`, `matches`, `blocks`, `reports`, `conversations`, `conversation_participants`,
`messages`, `notifications` (see `internal/platform/db/migrations`). Highlights:

- Everything references `users(id) ON DELETE CASCADE`: deleting an account erases all personal rows.
- `matches` stores the pair ordered (`user_a < user_b`) with a unique constraint: one match per pair, whoever liked first.
- `swipes` is the single source of truth for "already processed"; a pass or a like is final (409 on repeat) and survives unmatching, so a pair never reappears.
- `photos (user_id, position)` is unique and deferrable so reordering is one statement.
- Migrations are embedded, applied once in filename order inside a transaction, tracked in `schema_migrations` and guarded by an advisory lock.

### Matching

`discovery/repository.go` has one SQL predicate (`eligibleSQL`) that defines who may be shown *and*
who may be swiped: mutual gender / age / distance preferences, completed and discoverable profiles,
nothing already processed, no block in either direction. Swipe re-checks it server side, so a
client cannot react to an arbitrary user id. A per-pair advisory lock makes simultaneous likes
produce exactly one match. Ranking lives in `rankOrder` (shared interests, then distance) - the only
place to change to evolve recommendations.

### Location and privacy

Coordinates are rounded to 2 decimals (~1 km) before storage, never returned by any endpoint, and
only used in SQL (bounding box + haversine). Clients get a coarse bucket (`distanceKm`), hidden when
the user disables `showDistance`.

### Photos

Uploads are sniffed by content (JPEG/PNG only), size and pixel bounded, EXIF orientation applied,
flattened and re-encoded as JPEG (all metadata, GPS included, dropped), capped at 1280 px, stored under
server-generated keys through the `storage.Store` interface (local disk by default). Files are only served through
`GET /photos/:id/file` after the same visibility check as profiles (not blocked, discoverable or matched).

### Real time

`/ws` (gorilla/websocket) authenticates with a first `{"type":"auth"}` frame (tokens never go in URLs).
The in-process `realtime.Hub` fans events out per user. It is single-instance: to scale horizontally,
replace the hub with Redis/NATS pub-sub behind the `realtime.Publisher` interface. REST remains the
source of truth; clients refetch after reconnecting.

### Security summary

- JWT HS256, mandatory `exp`, strict method/claim validation, 15 min access tokens; opaque refresh tokens stored hashed, rotated on use (replay fails).
- bcrypt passwords (72 byte limit enforced), constant-time-ish login for unknown emails, no account enumeration on recovery.
- Password recovery uses a 10-char single-use code valid 15 minutes; confirming revokes every session.
- Per-IP limits on credential endpoints, per-user limits on swipes, messages, uploads and reports (in-memory token bucket).
- Authorization is always server side: conversation endpoints answer 404 to non-participants; blocks dissolve matches and hide profiles/photos both ways.
- Account deletion requires the password and cascades through the database, files and live sockets.

## Mobile

See `apps/mobile/README` sections in the root README. The client never talks to the network outside `src/api`.
