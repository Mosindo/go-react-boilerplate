# Architecture

## Monorepo

- `apps/mobile` — Expo app (iOS, Android, web), TypeScript strict.
- `services/api` — Go API (Gin, pgx), one HTTP server.
- `docs` — this file and the API reference.
- `docker-compose.yml` — PostgreSQL + API for local runs.

## Backend

```text
services/api/
├─ cmd/api/main.go            composition root (wiring, server, graceful shutdown)
├─ cmd/seed/main.go           demo data (refuses APP_ENV=production)
└─ internal/
   ├─ platform/
   │  ├─ config/              env parsing and validation
   │  ├─ db/                  pool, tracked embedded migrations
   │  ├─ middleware/          auth (JWT HS256), CORS, security headers, body limit, rate limit, logging
   │  ├─ httpx/               shared handler helpers (errors, params, pagination)
   │  ├─ errors/              ValidationError
   │  ├─ realtime/            in-process hub used by the long-poll endpoint
   │  ├─ storage/             Store interface + local disk implementation
   │  ├─ imaging/             upload validation, EXIF orientation, resize, JPEG re-encode
   │  ├─ signedurl/           HMAC signed photo URLs
   │  ├─ mail/                SMTP / dev log mailer
   │  ├─ logger/              structured logs with secret redaction
   │  └─ handlers/            /health
   └─ features/
      ├─ auth/  account/  profiles/  photos/  matching/  chat/  safety/  notifications/
```

Rules: `handler -> service -> repository`; SQL only in repositories; handlers stay thin; one shared pgx pool; no ORM.
Features may depend on `platform` and, one-way, on `profiles` (types and lookups): `matching -> profiles`, `account -> auth`.

### Request lifecycle

`SecurityHeaders -> LimitBody -> CORS -> RequestID -> metrics/logging -> route middleware (RequireUser, RateLimit) -> handler -> service -> repository`.

### Security model

- JWT HS256, mandatory expiry (15 min); refresh tokens are random, SHA-256 hashed in DB, rotated on use.
- Passwords: bcrypt; constant-time-ish login for unknown emails; 10–72 characters.
- Authorization is always derived from the token's user id; objects are looked up *with* the owner (`WHERE id = $1 AND user_id = $2`) so foreign ids behave as "not found".
- Photos: content-sniffed, decoded and re-encoded, random keys, signed URLs, no public directory.
- Rate limits and body limits protect credential, upload, swipe, message and report endpoints.
- Location: rounded coordinates only; never returned by any endpoint.

### Data model highlights

- `matches(user_a < user_b)` unique pair; `conversations.match_id` unique; messages cascade from conversations.
- `swipes(user_id, target_id)` primary key prevents duplicate decisions.
- `blocks`, `reports` (references `ON DELETE SET NULL` so moderation evidence survives account deletion).
- Indexes cover the hot paths: `messages(conversation_id, created_at DESC, id DESC)`, `notifications(user_id, created_at DESC, id DESC)`, `conversation_participants(user_id, conversation_id)`, `swipes(target_id, user_id) WHERE like`, partial index for discoverable profiles.

### Evolving recommendations

`matching/repository.go: candidateQuery` is the single place that defines eligibility and ordering. To plug in a better ranker, keep the eligibility `WHERE` clause (it is the security boundary) and change `ORDER BY`, or fetch a larger candidate pool and re-rank it in `matching.Service.Discover`.

## Frontend

```text
apps/mobile/src/
├─ api/        client.ts (single fetch wrapper: auth header, refresh-on-401, errors), endpoints.ts, types.ts
├─ state/      session (auth lifecycle), discoverStore (swipe queue), queryClient
├─ hooks/      queries.ts, useRealtime.ts (long poll -> query invalidation), useLocation.ts
├─ screens/    Auth, Forgot, Onboarding, Discover, ProfileDetail, Messages, Chat, Notifications, Me, EditProfile, Photos, Preferences, Privacy, Settings
├─ components/ PhotoGrid, PhotoCarousel, PreferencesForm, InterestPicker, ReportSheet
├─ ui/         theme (light/dark), Text, Button, Input, Chip, Avatar, Screen, StateViews, Toast, Sheet
├─ navigation/ RootNavigator (auth / onboarding / main tabs + stack)
└─ lib/        pure, unit-tested helpers (dates, validation, formatting, queue merge)
```

UI code never calls `fetch`; everything goes through `src/api`. Server state lives in TanStack Query; the swipe queue lives in `discoverStore` so the detail screen and the discover screen share it.
