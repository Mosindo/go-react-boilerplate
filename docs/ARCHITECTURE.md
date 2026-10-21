# Architecture

Alma is a monorepo with a Go REST/WebSocket API and an Expo (React Native + web) client. The README covers setup; this document explains the boundaries.

## Repository layout

- `apps/mobile` — the client. `src/api` (HTTP client + one module per domain), `src/hooks`, `src/screens`, `src/components`, `src/shared` (design system), `src/lib` (pure, unit-tested logic), `e2e` (Playwright).
- `services/api` — the server. `cmd/api` (entry point), `cmd/seed` (demo data), `internal/app` (router wiring), `internal/platform` (shared infrastructure), `internal/features` (domain modules).
- `docker-compose.yml` — PostgreSQL + API for local use.

## Backend layering

```
handler.go      parse + validate transport shape, call the service, write JSON (httpx.Fail maps errors)
service.go      business rules, normalisation, authorisation decisions, event publication
repository.go   SQL only (pgx, no ORM); transactions live here
model.go        request / response / domain types
routes.go       route registration, rate limits
```

Services return `platformerrors.Error` (`Kind`, stable `Code`, user message). Transport mapping happens once, in `internal/platform/httpx`. Any other error is logged and answered as a masked `500`.

Cross-feature rules: features never import each other's repositories. `discovery` depends on a small `ProfileLoader` interface (satisfied by `profiles.Service`); lists of people (matches, chats, notifications) join `profiles` and the main photo in SQL and sign thumbnails through `storage.Signer` — a constant number of queries per page, no N+1.

## Platform packages

| Package | Purpose |
| --- | --- |
| `config` | environment parsing and validation (JWT strength, production requirements) |
| `db` | pool, retry connect, embedded migrations tracked in `schema_migrations` |
| `middleware` | JWT + session validation, rate limiting, CORS, security headers, request logging |
| `storage` | `Storage` interface (local disk today), HMAC `Signer` for expiring photo URLs |
| `realtime` | in-process WebSocket hub implementing `Publisher` |
| `mail` | `Mailer` interface: SMTP, or a log-only mailer for development |
| `geo` | haversine distance, coordinate rounding, coarse display distance |
| `httpx`, `errors`, `logger`, `handlers`, `testutil` | helpers |

## Key flows

**Swipe → match.** `POST /swipes` runs one transaction: advisory lock on the *pair*, caller/target eligibility checks, `INSERT … ON CONFLICT DO NOTHING` on `swipes`, reciprocal-like check, then `matches` (canonical `user_a < user_b`), `conversations`, both `conversation_participants` and both notifications. The lock makes two simultaneous likes yield exactly one match; the unique constraints are the second line of defence. Events are published after commit.

**Chat authorisation.** Every chat query starts from `conversation_participants (conversation_id, user_id)`; no row ⇒ `404`. Messages, read state, local hiding and notifications derive from it. Blocking or unmatching deletes the match, which cascades to the conversation and messages.

**Photos.** Upload → type sniffing → dimension limits → decode → re-encode as JPEG (full 1080 px + 360 px thumbnail) → store → insert (per-user advisory lock enforces the 6 photo limit and contiguous positions). Clients only ever get signed, expiring `/media/photos/:id?size&exp&sig` URLs, handed out by the same code paths that decide profile visibility.

**Sessions.** Access JWT (15 min, HS256, `uid` + `sid`) + opaque refresh token (hashed, rotated on use). `RequireUser` re-checks the session row through a 10 s cache, so revocation is near-immediate.

**Realtime.** `GET /ws`, first frame `{"type":"auth","token":…}`. The client folds `message.new`, `conversation.read`, `conversation.removed`, `match.new`, `notification.new` into the TanStack Query cache and refetches on reconnect; REST remains the source of truth.

## Frontend

Screens orchestrate; components render; `src/api` is the only code that touches the network; `src/hooks` hold cross-screen behaviour (auth/session, realtime, optimistic swipes, photo operations). The root navigator selects Auth → Onboarding → Main from the auth state; the onboarding decision is taken once per sign-in so finishing the last step cannot unmount the wizard mid-flight. Visual identity lives in `src/theme` (light/dark palettes, spacing, radii) and `src/shared/ui`.

## Scaling notes

The hub, rate limiter and session cache are per-process. Run one API instance, or move those three pieces to Redis / PostgreSQL `LISTEN/NOTIFY` (each is behind a small interface or constructor). Photos live on local disk behind the `Storage` interface — swap in an S3-compatible implementation for multi-instance deployments.
