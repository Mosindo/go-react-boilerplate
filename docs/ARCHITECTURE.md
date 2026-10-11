# Ember architecture

## Repository

* `apps/mobile` — Expo client (iOS, Android, web)
* `services/api` — Go API (single Gin HTTP server, PostgreSQL, no ORM)
* `docs/API.md` — HTTP + WebSocket contract (source of truth shared by both sides)
* `scripts/smoke-api.mjs` — end-to-end smoke test for a running API

## Backend

```text
cmd/api            composition root: config → pool → migrations → services → routes → graceful shutdown
cmd/seed           development-only demo data (refuses APP_ENV=production, needs SEED_CONFIRM=yes)
internal/platform  config · db (+ migrations, Querier, Haversine) · middleware (auth, rate limit, security headers,
                   CORS, logging) · httpx (error/response helpers, cursors) · errors (AppError) · realtime (hub)
                   · mail · signedurl · logger
internal/features  auth · users · photos · discovery · chat · notifications · safety · realtime
```

Layering inside a feature: `routes → handler → service → repository`. Handlers bind/validate transport concerns and
translate `*platformerrors.AppError` through `httpx.Respond`; services own business rules and return `AppError`s with a
stable machine code; repositories hold all SQL. Features talk to each other through small interfaces
(`discovery.ProfileReader`, `discovery.ConversationReader`) or through transactional helpers that accept a
`db.Querier` (`chat.CreateConversation`, `notifications.Insert`, `discovery.DeleteMatchBetween`) so that multi-table
operations (a match = match row + conversation + participants + two notifications) commit atomically.

### Security model

* Passwords: bcrypt, 8–72 bytes. Login performs a dummy bcrypt compare for unknown e-mails (timing).
* Access token: HS256 only (`WithValidMethods`), `exp` required, `iss=ember-api`, `uid` + `sid` claims. Refresh token:
  256-bit random, stored as SHA-256, rotated on every use (an old token stops working immediately); password change/reset revokes all sessions.
* Authorization is always derived from the token's user id; every conversation/photo/notification query is scoped to it
  and non-members receive 404 (never 403) to avoid existence leaks.
* Photos: sniffed, decoded, resized, re-encoded; served through HMAC-signed URLs (24–48 h, day-aligned for caching).
* Rate limiting: in-memory fixed windows (per IP for unauthenticated routes, per user otherwise). Behind a proxy set `TRUSTED_PROXIES`.
* Location: opt-in, rounded to 0.01° before storage, distance coarsened on output.
* WebSocket: ticket flow, origin check, 1 KB frame limit, ≤ 5 connections per user, ping/pong deadlines.

### Matching

`discovery.Repository.Swipe` runs in one transaction guarded by `pg_advisory_xact_lock` on the user pair, so two
simultaneous opposite likes serialise and exactly one of them creates the match. `swipes` has a primary key
`(from_user_id, to_user_id)`; `matches` stores the pair ordered (`user_a < user_b`) with a unique constraint.

### Realtime

`platform/realtime.Hub` maps user id → live connections and implements `Publisher`. Events are best-effort; the
database is the source of truth and clients refetch on reconnect. Replace the hub with a pub/sub-backed `Publisher` to
run several API replicas.

## Frontend

```text
src/api         typed REST client (single place that uses fetch), token refresh single-flight, endpoint modules, types
src/realtime    WebSocket client (ticket, backoff, ping) + event → react-query cache reducers
src/features    auth · onboarding · discover · likes · chat · profile · notifications · safety
src/ui, theme   design system (light/dark tokens, primitives)
src/i18n        French strings
```

UI code never calls the network directly; screens use feature hooks built on `src/api`.
