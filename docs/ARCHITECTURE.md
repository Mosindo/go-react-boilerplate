# Amora Architecture

Amora is a monorepo: a Go API, an Expo (React Native) client that also builds for the web, and Docker-based infrastructure. This document explains the boundaries so features can be added without breaking them.

## 1. Repository

- `apps/mobile` — Expo client (TypeScript)
- `services/api` — Go API (Gin, PostgreSQL, `pgxpool`, no ORM)
- `docker-compose.yml` — Postgres + API + uploads volume (`infra/` holds a variant for ad-hoc setups)
- `scripts/` — PowerShell QA helpers; `apps/mobile/scripts` — Node smoke test and Maestro fixtures
- `docs/` — this file

## 2. Backend

```text
services/api/
├─ cmd/api/main.go        composition root: config → pool → migrations → router → server
├─ cmd/seed/              development-only demo data
└─ internal/
   ├─ platform/           shared, domain-agnostic code
   │  ├─ config/ db/ logger/ errors/ validate/
   │  ├─ middleware/      auth (JWT), security headers, CORS, request id/log/metrics, rate limit, body limit
   │  ├─ realtime/        in-memory WebSocket hub + Publisher interface
   │  └─ mail/            Mailer interface: SMTP or log (dev)
   └─ features/
      ├─ auth/            register, login, refresh rotation, logout, recovery, delete account
      ├─ profiles/        profile, preferences, interests, privacy, public view, distance/age SQL helpers
      ├─ photos/          upload pipeline, storage abstraction, ordering, access-checked serving
      ├─ matching/        discovery query, swipes, match creation
      ├─ conversations/   match-bound chat, read state, hide, unmatch
      ├─ notifications/   in-app notifications (internal Notify only)
      └─ safety/          blocks and reports
```

Rules: `handler → service → repository`; no SQL in handlers; handlers stay thin and map domain errors to HTTP codes; shared concerns live in `platform`; features talk to each other only through small interfaces declared by the consumer (for example `matching.ProfileReader`, `conversations.Notifier`, `realtime.Publisher`) and are wired in `main.go`.

### Request pipeline

`Recovery → security headers → CORS → request id → metrics/logging → body limit (1 MiB, photos exempt) → per-IP limiter → route middlewares (JWT, per-user limiters) → handler`.

### Data model

`users` ⟶ `profiles` (1:1) ⟶ `preferences` (1:1), `user_interests` ⟷ `interests`, `photos` (≤ 6, `UNIQUE (user_id, position)` deferrable so reordering is one statement), `swipes (swiper, target)` primary key, `matches (user_a < user_b)` which is also the conversation, `chat_messages`, `blocks`, `reports`, `password_resets`, `sessions`, `notifications`.

Migrations are embedded SQL files applied once each in a transaction (`schema_migrations`), serialized by `pg_advisory_lock`.

### Matching and discovery

`matching/repository.go` holds one SQL fragment describing who may appear for a viewer: visible profile with a photo, mutual gender interest, both age windows, viewer's distance limit (haversine, with a latitude bounding box for index use), no block in either direction. Discovery adds "not already swiped" and the ranking `ORDER BY`; swipe validation reuses the same predicate. A swipe runs in one transaction behind a per-pair advisory lock so two simultaneous likes yield exactly one `matches` row.

### Realtime

Clients open `GET /ws`, send `{"type":"auth","token":…}` as the first frame, and receive JSON events. Services publish through `realtime.Publisher`; the hub is process-local. Replace it with a pub/sub-backed implementation to scale horizontally — nothing else changes.

### Security model

- JWT HS256 only, expiration required, strict claim checks; refresh tokens are random, stored hashed, rotated on use; password reset revokes all sessions
- bcrypt passwords (8–72 bytes), constant-time-ish login for unknown emails, no account enumeration on recovery
- every route authorizes on the server; foreign resources answer 404 instead of 403 where existence would leak
- uploads: content sniffing, size and pixel limits, re-encoding, random server-side file names, path-traversal-proof storage keys, authenticated reads
- location rounded before storage; birth date, email and coordinates never appear in another user's payload
- rate limits on credentials, swipes, messages, uploads and reports; global per-IP limit

## 3. Mobile

```text
apps/mobile/
├─ App.tsx                      providers: SafeArea → Auth(QueryClient) → Realtime → Navigator
└─ src/
   ├─ api/                      the only code that performs network I/O (client.ts adds auth + single-flight refresh)
   ├─ hooks/                    useAuth, useRealtime (WebSocket + cache updates), useDiscoveryQueue
   ├─ lib/                      pure logic with unit tests
   ├─ components/               feature components (forms, photo manager, swipe card, safety menu…)
   ├─ navigation/ screens/      root stack + tabs; onboarding gate lives in AppNavigator
   └─ shared/                   design system (tokens, ui, layout, feedback, dialog)
```

State: TanStack Query for server state (keys in `api/queryKeys.ts`); the WebSocket patches or invalidates those caches. Navigation gate: signed out → `AuthScreen`; signed in without a profile or photo → onboarding wizard; otherwise tabs (Discover, Matches, Activity, Profile) over a native stack (chat, profile detail, edit, photos, preferences, settings).

Platform notes: `shared/dialog.tsx` replaces `Alert` on web; photos are fetched with the bearer token (headers on native, blob URLs on web); uploads use a `{uri,name,type}` form part on native and a real `Blob` on web.

## 4. Testing strategy

- Go: pure unit tests (distance rounding, age, image pipeline, limiter, validators) and database-backed integration tests that drive the real router (`cmd/api/*_test.go`)
- Mobile: Vitest for pure logic and the HTTP client; `tsc` and ESLint
- E2E: Node HTTP/WebSocket smoke journey; Playwright browser journey against the web export; Maestro flow for devices
