# Alma — Architecture

## Layout

- `apps/mobile` — Expo / React Native client (TypeScript strict)
- `services/api` — Go API (Gin, pgx, PostgreSQL)
- `docker-compose.yml` — local stack (postgres + api)
- `docs/` — this document and contributor agent notes

## Backend

```text
services/api/
├─ cmd/api/main.go        server entry point (config → db → migrations → router)
├─ cmd/seed/              dev-only demo data (refuses APP_ENV=production)
└─ internal/
   ├─ app/                NewRouter(): wires platform + features; integration tests
   ├─ platform/           config, db (+ migrations), middleware (auth, security headers, CORS,
   │                      rate limit, logging), realtime hub, mail, validate, httpx, logger
   └─ features/<name>/    handler.go → service.go → repository.go, model.go, routes.go
```

Layering rules: handlers parse/answer HTTP only; services hold business rules; repositories are the only place with SQL; features never import each other's repositories (they share small interfaces such as `CardLoader`, `realtime.Publisher`, `notifications.Execer`).

### Features

| Feature | Responsibility |
| --- | --- |
| auth | register (18+), login, refresh rotation, logout, password reset by e-mail code, account deletion |
| profiles | own profile, preferences, interests, public `Card` assembly (batched, no N+1), visibility check |
| photos | upload pipeline (decode → resize → JPEG re-encode), ordering, authenticated delivery |
| matching | discovery feed query, like/pass, match creation, unmatch |
| chat | conversations, messages, read state, local hide, WebSocket gateway |
| safety | block (also ends the match) and report |
| notifications | in-app notifications (inserted inside the originating transaction) |

### Key invariants (all enforced server-side)

- Age ≥ 18 at registration (`platform/validate`), date of birth never serialized to other users.
- One swipe per ordered pair (`likes` PK); a per-pair advisory lock + unique `matches(user_low,user_high)` guarantee one match under concurrency.
- A conversation is accessible only if the caller is a participant, the match is active and no block exists in either direction (`chat.accessSQL`). Any failure answers 404.
- Photos and profiles use the same visibility rule (`profiles.CanView`).
- Coordinates are rounded to ~1 km before storage; distances are bucketed before display.
- JWT: HS256 only, mandatory `exp`, required `uid`/`sid`. Refresh tokens are hashed, rotated on use, revoked on reset/logout.

### Recommendation algorithm

Candidate selection lives in one SQL constant (`matching.feedSQL`): mutual gender/age/distance preferences, discoverable, has a photo, not swiped, not blocked. Ordering (`people who liked me first, then recent activity`) is the single place to evolve ranking later (e.g. interest overlap scores or a separate scoring service).

### Realtime

`/ws` accepts a socket, expects `{"type":"auth","token":…}` within 5 s, then registers the connection in an in-memory hub. Services publish `message`, `read`, `match`, `unmatch` events to user ids. The hub is per-instance; scale out by implementing `realtime.Publisher` over Postgres `LISTEN/NOTIFY` or Redis.

### Migrations

Embedded SQL files applied in order, once each, recorded in `schema_migrations`, serialized with an advisory lock. Never edit a released migration; add a new one.

## Mobile

```text
apps/mobile/src/
├─ api/          single network layer: auth refresh (single-flight), typed endpoints, photo URLs
├─ auth/         session provider (SecureStore tokens, expiry handling, keep-alive)
├─ realtime/     WebSocket provider (reconnect with backoff, resync, query-cache updates)
├─ screens/ components/ navigation/   UI
├─ lib/          pure logic (validation, formatting, message merging) — unit tested
└─ shared/       design system tokens and primitives
```

Server state lives in TanStack Query; screens never call `fetch` directly. Photos are private: `RemotePhoto` fetches them with the bearer token and caches them on disk per immutable URL.
