# Lumo architecture

Lumo is a free dating app built on the `go-react-saas` boilerplate: a Go/Gin/PostgreSQL API and an Expo (React Native + web) client in one monorepo. This document explains how it is organised and why, so new contributors can extend it without breaking its boundaries. The HTTP contract lives in [`API.md`](API.md).

## 1. Monorepo structure

```text
apps/mobile/     Expo client (iOS, Android, web), TypeScript
services/api/    Go API (Gin, pgxpool, no ORM)
infra/           Docker Compose variant for running from infra/
scripts/         smoke-api.mjs: dependency-free end-to-end check of a running API
docs/            this document, API reference, agent briefs
```

## 2. Backend

```text
services/api/
├─ cmd/api/            composition root (config, DB, wiring, graceful shutdown) + integration tests
├─ cmd/seed/           demo data (refuses to run in production)
└─ internal/
   ├─ platform/        shared technical concerns
   │  ├─ config/ db/ logger/ errors/ handlers/(health)
   │  ├─ httpx/        uniform error body, strict JSON binding, pagination, UUID check
   │  ├─ middleware/   JWT auth, security headers, CORS, request id/log/metrics
   │  ├─ ratelimit/    fixed-window limiter + Gin middlewares (per IP / per user)
   │  ├─ storage/      Store interface + local-disk implementation (photo files)
   │  ├─ mail/         Mailer interface (SMTP, log, discard)
   │  └─ realtime/     WebSocket hub, single-use tickets, event publisher
   └─ features/        one folder per domain (handler → service → repository)
      ├─ auth/         register, login, refresh rotation, logout, password reset/change, account deletion
      ├─ profiles/     profile, preferences, interests catalogue, public profile, /me
      ├─ photos/       upload pipeline, ordering, serving with visibility rules
      ├─ matching/     discovery, swipes, matches, likes received
      ├─ chat/         conversations, messages, read state, local deletion
      ├─ safety/       blocks, reports
      └─ notifications/ in-app notifications (created by matching and chat)
```

Every feature has `handler.go` (HTTP only), `service.go` (rules, validation, event publishing), `repository.go` (all SQL) and `routes.go`/`model.go`. There is one HTTP server (Gin) and no `net/http` handlers besides what Gin wraps.

Cross-feature reuse is deliberately small and one-directional: `profiles` exports the shared `ProfileSummary`/`PublicProfile` shapes and SQL fragments; `notifications.Create` can join the caller's transaction (a match and its notifications commit atomically). Nothing imports `matching`, `chat` or `safety`.

### Request lifecycle

`SecurityHeaders → CORS → RequestID → logging/metrics → [rate limit] → RequireUser (JWT HS256, `exp`, `uid`, `sid` mandatory) → handler → service → repository`.

### Key design decisions

| Topic | Decision | Why |
| --- | --- | --- |
| Auth | 15 min JWT access token + 30 day opaque refresh token stored hashed (SHA-256), rotated on each use with compare-and-swap on the old hash | stolen refresh tokens work once at most; races cannot fork a session |
| Passwords | bcrypt, 8–72 bytes, constant-time login (dummy hash for unknown e-mails) | no account enumeration through timing |
| Recovery | 8 char one-time code (32-symbol alphabet) stored as HMAC, 30 min TTL, 5 guesses, 3 codes/hour; responses never reveal whether an account exists | brute force is infeasible |
| Age | `birth_date` required, 18–99 enforced server-side | the client is never trusted |
| Location | stored rounded to 0.01° (≈1 km); other users only ever see a bucketed distance (1, 2, 5, 10, 20, 50, 100…) and the owner can hide it | no precise position leaves the server, nothing exact is even stored |
| Photos | multipart, ≤ 8 MB, header-checked dimensions (anti decompression bomb), decoded then **re-encoded to JPEG** (EXIF orientation applied, metadata/trailing bytes dropped), random 128-bit keys, served only through an authorised endpoint, `private` cache | polyglot/malware uploads cannot survive; photos are not world-readable |
| Matching | `swipes(user, target, action)` PK = no duplicates; `matches(user_a < user_b)` unique pair; an advisory lock per pair serialises simultaneous mutual likes; a pair that was unmatched is never revived | exactly one match and one conversation per pair |
| Discovery | one SQL query with mutual gender/age/distance checks, a latitude window on `idx_profiles_discovery`, exclusion of swiped/blocked/hidden/photoless profiles, then interests/photos hydrated with two batched queries; ordering is a pluggable `Ranker` (people who already liked you first, then recent activity) | no N+1; the algorithm can evolve without touching the filters |
| Chat | one conversation per active match; every access goes through a participant+active-match check, so outsiders always get 404; read state is a per-participant `last_read_at` watermark (O(1) read receipts); local deletion = `cleared_at` + `hidden` | cheap, leak-free |
| Real time | WebSocket hub in memory; clients exchange a JWT for a 30 s single-use ticket (no token in URLs) | works for browsers; to scale out, swap `realtime.Publisher` for Postgres LISTEN/NOTIFY or Redis |
| Safety | blocking is symmetric in effect: it dissolves the match and hides both parties from discovery, profiles, photos, chats; reports are stored for moderators (kept after the reported account is deleted) and capped at 20/day/user | |
| Deletion | `DELETE /me` re-checks the password, purges stored files, closes live sockets and deletes the user row; foreign keys cascade to every personal record | right to erasure |
| Rate limiting | per-IP on credentials/signup/recovery, per-user on uploads, swipes, messages, safety actions; limits only stop scripted abuse (no product quota: likes are unlimited) | |
| Free forever | there is no billing code, plan, boost or quota anywhere | product rule |

### Database

Migrations are embedded SQL files applied in order, recorded in `schema_migrations`, each in its own transaction, guarded by a Postgres advisory lock so several instances can start together. `014` drops the SaaS tables inherited from the boilerplate; `015` creates the dating schema; `016` adds the haversine function.

Tables: `users` (+`is_demo`, `last_active_at`), `sessions`, `password_reset_codes`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. All have foreign keys with `ON DELETE CASCADE` (reports use `SET NULL`), CHECK constraints for enumerations and bounds, and the indexes required by `AGENTS.md` (`users.email`, `conversation_participants (conversation_id, user_id)`, `messages (conversation_id, created_at)`, `notifications (user_id, created_at)`, plus `idx_profiles_discovery`, `idx_swipes_target_action`, `idx_matches_user_a/b`, `idx_photos_user_created_at`…).

### Scaling notes

* Discovery sorts every compatible candidate inside the search radius; this measured ~100 ms with 100 000 profiles packed into one 2°×2° box and is far cheaper in realistic densities. When it matters, replace the `Ranker` with a precomputed per-user candidate queue.
* The hub, ticket store and rate limiters are per process. Run one API instance, or move them to Redis/Postgres first.
* Photos live on local disk behind `storage.Store`; implement the interface for S3-compatible storage before running several instances.

## 3. Frontend

See `apps/mobile/README.md` (screens, state management, realtime client, theming) for the details of the client. The rules that matter architecturally:

* UI code never calls `fetch`: everything goes through `src/api/*` and the shared client, which attaches the token and transparently refreshes it once on `401`.
* Photos need the `Authorization` header, so a small `AuthImage` component loads them (blob URL on web).
* Server state is cached with TanStack Query; WebSocket events patch/invalidate those caches.
* Strings are centralised (French) to make adding a locale straightforward.

## 4. Adding a feature

1. Add a migration (never edit an applied one).
2. Create `internal/features/<name>/{model,repository,service,handler,routes}.go`.
3. Wire it in `cmd/api/main.go` and add integration tests in `cmd/api`.
4. Document the endpoints in `docs/API.md`.
