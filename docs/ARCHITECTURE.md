# Lumen Architecture

Lumen is a free dating app: a Go/Gin API on PostgreSQL and an Expo (React Native + web) client in one monorepo.

## Layout

- `apps/mobile` – Expo app. `src/api` (HTTP + WebSocket clients, the only code that touches the network), `src/hooks` (React Query hooks), `src/features/<area>` (screens + components), `src/navigation`, `src/ui` (design system), `src/theme`, `src/store` (tokens and small client stores), `src/lib` (pure helpers).
- `services/api` – Go API (`cmd/api`, `cmd/seed`, `internal/platform`, `internal/features`).
- `infra` – Docker Compose (postgres + api). `docs/API.md` – the contract between both.

## Backend

`handler → service → repository`, per feature, wired only in `cmd/api/router.go`:

| Feature | Responsibility |
|---|---|
| `auth` | register/login/refresh/logout, password reset & change, account deletion, session validation |
| `profiles` | profile, preferences, interests, coarse location, visibility rules (`VisibilitySQL`) |
| `photos` | upload pipeline (sniff → decode → resize → re-encode), ordering, authorised file serving |
| `discovery` | candidate deck (hard filters + `rankScore`), swipes, atomic match creation |
| `chat` | matches list, conversations, messages, read state, unmatch |
| `safety` | blocks and reports |
| `notifications` | in-app notifications |

`internal/platform`: `config`, `db` (embedded, versioned migrations), `middleware` (auth, rate limit, CORS, security headers, logging), `realtime` (WebSocket hub), `storage` (photo store interface), `mail`, `errors` + `httpx` (typed errors → consistent JSON), `pagination` (keyset cursors), `logger`.

Rules: no SQL in handlers or services; services never import `net/http`/Gin; cross-feature calls go through small interfaces (e.g. discovery depends on a `MatchReader` implemented by chat). Match creation inserts into `conversations`/`notifications` inside the discovery transaction on purpose, so a match is all-or-nothing.

## Realtime

Writes go through REST; the in-memory hub only pushes (`message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`). Clients authenticate with the first WebSocket frame and must resync through REST after reconnecting. Scale-out: replace `realtime.Publisher` with a Postgres `LISTEN/NOTIFY` or Redis implementation.

## Frontend

React Query owns server state; `src/api/realtimeCache.ts` applies WebSocket events to the query cache. Photos are loaded with the bearer token (headers on native, fetch → object URL on web). Auth state: access token in memory + SecureStore (localStorage on web), automatic single-flight refresh on 401.

## Adding a feature

Add the next numbered file in `internal/platform/db/schema/`, create `internal/features/<name>/{model,repository,service,handler,routes}.go`, wire it in `router.go`, add integration tests in `cmd/api`, document endpoints in `docs/API.md`.
