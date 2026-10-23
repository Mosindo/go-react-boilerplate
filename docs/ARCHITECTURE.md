# Architecture

Cléo is a monorepo: a Go API (`services/api`), an Expo/React Native client (`apps/mobile`), Docker Compose for local
infrastructure (`docker-compose.yml`, `infra/`), and QA scripts (`scripts/`). The HTTP contract between client and
server is [`API.md`](API.md).

## Backend

```text
services/api/
├─ cmd/api/            main.go: config, pool, migrations, router wiring, graceful shutdown; integration tests
├─ cmd/seed/           demo accounts (development only)
└─ internal/
   ├─ platform/        shared concerns
   │  ├─ config/       environment variables
   │  ├─ db/           pgxpool connection + embedded SQL migrations (schema/*.sql, tracked in schema_migrations)
   │  ├─ middleware/   JWT auth (+ session check), rate limiting, CORS, security headers, request id/logging
   │  ├─ realtime/     in-process WebSocket hub (events to connected users)
   │  ├─ mailer/       SMTP or log mailer (password-reset codes)
   │  ├─ httpx/        bind/fail helpers, UUID + query parsing
   │  ├─ pagination/   opaque keyset cursors
   │  └─ errors/       AppError (status, code, message, field errors)
   └─ features/        one package per domain, each: handler.go -> service.go -> repository.go (+ model.go, routes.go)
      ├─ auth/          register/login/refresh/logout, password reset, GET/DELETE /me
      ├─ profiles/      profile, preferences, interests
      ├─ photos/        upload pipeline (validate, strip metadata, resize), ordering, private serving
      ├─ discovery/     candidate query + ranking (Recommender), profile cards
      ├─ matching/      swipes, matches, unmatch, hide
      ├─ chat/          messages, read receipts
      ├─ moderation/    blocks, reports, automatic suspension
      └─ notifications/ stored notifications + realtime push
```

Rules: handlers only bind/validate transport concerns and call services; services hold business rules and return
`AppError`s that are safe to show; repositories own all SQL; features talk to each other through small interfaces declared by
the consumer (`Notifier`, `PhotoLister`, `realtime.Publisher`), wired in `cmd/api/main.go`. One Gin server, no `net/http`
handlers mixed in.

### Request pipeline

`Recovery → SecurityHeaders → CORS → RequestID → metrics/logging → [RateLimit(IP)] → RequireUser (JWT + live session/user check) → RateLimit(user) → handler`.

### Data model

`users` ─ `profiles` / `preferences` / `user_interests` ─ `interests` · `photos` ─ `photo_blobs` · `swipes` · `matches` (one row per
ordered pair, doubles as the conversation) ─ `messages` · `blocks` · `reports` · `notifications` · `sessions` · `password_resets`.
Everything cascades from `users`, so account deletion removes all personal data in one statement.

### Consistency guarantees

- A swipe and the reciprocity check run in one transaction under a per-pair advisory lock: simultaneous mutual likes create exactly one match.
- Refresh-token rotation is a compare-and-swap on the stored hash: concurrent refreshes with one token yield one winner.
- Message insertion checks membership in the same statement as the insert; unmatch/block cannot race it.
- Photo mutations lock the owner row and use a deferred unique constraint to renumber positions safely.

## Mobile

```text
apps/mobile/src/
├─ api/        the only network layer: fetch client (timeout, error mapping, single-flight refresh), typed endpoints
├─ realtime/   WebSocket client (auth frame, ping, backoff reconnect) + React Query cache updates
├─ features/   swipe deck, photo manager, moderation actions, chat outbox, location
├─ screens/    auth, onboarding, main tabs, settings
├─ ui/         design-system components;  theme/ light + dark tokens;  i18n/ French copy;  lib/ pure logic (unit-tested)
```

## Extending

- New domain: create `internal/features/<name>/` with the five files, add a migration `schema/NNN_<name>.sql`, wire it in `main.go`,
  document endpoints in `API.md`, add integration tests in `cmd/api`.
- Better recommendations: implement `discovery.Recommender` (returns ranked `Candidate`s) and inject it in place of the SQL ranker.
- Several API instances: replace `realtime.Hub` and the in-memory `middleware.Limiter` with shared implementations (Postgres LISTEN/NOTIFY, Redis).
