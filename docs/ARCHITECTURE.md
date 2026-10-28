# Architecture

## Layout

```text
apps/mobile            Expo / React Native client
services/api           Go API (single Gin HTTP server, single pgx pool)
docs                   API contract and this document
```

## Backend

```text
cmd/api/main.go        composition root (config → pool → migrations → wiring → http.Server + graceful shutdown)
cmd/seed               demo data (refuses to run in production)
internal/platform/
  config               environment parsing and validation
  db                   pool, embedded migrations (tracked, advisory-locked)
  middleware           JWT auth (HS256 only, mandatory exp), CORS, security headers, request id/logging, rate limiting
  realtime             WebSocket hub (first-frame auth, bounded buffers)
  storage              local blob storage (opaque keys, outside the web root)
  mail                 mailer interface (log in dev, SMTP in prod)
  httpx                handler helpers: error rendering, binding, cursors, pagination
  errors               typed client-visible errors (status + machine code)
  logger, handlers     structured logging with secret redaction, /health
internal/features/
  auth                 register/login/refresh/logout, recovery, password change, account deletion
  profiles             profile, preferences, interests, location, visibility rule, public views
  photos               validation/re-encoding pipeline, ordering, authenticated delivery
  discovery            candidate queue + swipes + atomic matching
  chat                 conversations (= matches), messages, read state, unmatch
  notifications        persistent notifications + live push
  safety               blocks and reports
```

Dependency direction (no cycles): `notifications ← chat ← discovery`, `profiles ← photos / discovery`, `auth` uses a `FileRemover`. Features depend on small interfaces (`Notifier`, `MatchHandler`, `Visibility`, `realtime.Publisher`) declared where they are used.

Rules kept from the boilerplate: Gin only (no `net/http` routing mix), thin handlers, business logic in services, SQL only in repositories, shared concerns in `platform`, domain concerns in `features`.

### Request flow

`route → middleware (request id, auth, rate limit) → handler (bind) → service (rules, orchestration) → repository (SQL) → handler (JSON)`. Expected failures are `platformerrors.Error` values rendered by `httpx.Fail`; anything else is logged and returned as an opaque 500.

### Data model

See `internal/platform/db/migrations/001_init.sql`. Highlights: `matches(user_a < user_b)` unique pair = conversation; `swipes(swiper_id, target_id)` primary key prevents duplicates; `photos(user_id, position)` deferrable unique for atomic reordering; `reports.reported_id ON DELETE SET NULL` to retain evidence; everything else cascades from `users` so account deletion is a single `DELETE`.

## Mobile

```text
src/api          typed client (timeout, single-flight token refresh), endpoint modules, types
src/auth         session context (secure storage)
src/realtime     reconnecting WebSocket client + pure cache reducers
src/navigation   auth / onboarding / main tabs / modals
src/screens      auth, onboarding wizard, discover, chats, notifications, profile
src/components   design-system components (Button, TextField, Chip, SwipeDeck, AuthImage …)
src/theme        tokens, light/dark palettes
src/lib          pure helpers (validation, formatting, swipe thresholds, location)
```

UI code never calls `fetch` directly; everything goes through `src/api`. Server state lives in TanStack Query; the WebSocket only updates those caches.

## Extending

- New feature: add `internal/features/<name>/{model,repository,service,handler,routes}.go`, register it in `cmd/api/main.go`, add a migration file, extend `docs/API.md`, add tests to `cmd/api`.
- Better recommendations: replace `Candidates` ordering in `discovery/repository.go` (the eligibility predicate must stay shared with `Swipe`).
- Horizontal scale: swap `realtime.Hub.Publish` for a shared bus and the in-memory limiter for a shared one.
