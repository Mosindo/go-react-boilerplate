# Lueur architecture

## 1. Overview

```text
┌────────────────────────┐   HTTPS (REST, JSON)    ┌──────────────────────────────┐     ┌────────────┐
│ apps/mobile (Expo)     │ ──────────────────────▶ │ services/api (Go, Gin)       │ ──▶ │ PostgreSQL │
│ iOS · Android · Web    │ ◀────────────────────── │  handler → service → repo    │     └────────────┘
│                        │   WSS /realtime (events)│  realtime hub · media signer │ ──▶ uploads volume
└────────────────────────┘                         └──────────────────────────────┘     (photos)
```

There is one HTTP server, one shared `pgxpool`, and no ORM: all SQL lives in repositories.

## 2. Backend (`services/api`)

```text
cmd/api/main.go            composition root (config, DB, migrations, wiring, graceful shutdown)
cmd/seed/main.go           demo data generator (refuses production)
internal/platform/
  config/                  env parsing + validation (JWT secret strength, SMTP in production)
  db/                      pool, embedded migrations, schema_migrations tracking, legacy-schema guard
  authtoken/               HS256 access tokens: strict alg/exp/iss/claims validation
  middleware/              auth, CORS, security headers, request id/logs/metrics, rate limiting
  httpx/                   uniform JSON errors, UUID/limit/cursor validation
  media/                   HMAC-signed, expiring photo URLs
  storage/                 Store interface + local disk implementation (path-traversal safe)
  mailer/                  SMTP mailer + dev log mailer
  realtime/                WebSocket hub (Publisher interface) and handler
  logger/                  structured logs with secret redaction
internal/features/
  auth/                    register, login, refresh rotation, logout, password reset/change, account deletion
  profiles/                profile, preferences, privacy settings, interests, public profile cards
  photos/                  upload/replace/delete/reorder, image processing, signed serving
  discovery/               eligibility query + pluggable Ranker
  matching/                swipes (like/pass), race-safe match creation, undo pass, unmatch
  chat/                    match-scoped conversations, messages, read receipts, local deletion
  safety/                  blocks and reports
  notifications/           in-app notifications (collapsed per conversation) + realtime push
```

Each feature follows `model.go`, `repository.go`, `service.go`, `handler.go`, `routes.go`. Handlers only parse
and validate transport concerns, services hold the business rules, repositories hold the SQL. Features depend
on each other through small interfaces declared on the consumer side (for example `discovery.ProfileReader`
and `matching.Notifier`).

### API surface

| Area | Endpoints |
| --- | --- |
| Health | `GET /health` |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/password/forgot`, `/auth/password/reset` |
| Account | `GET /me`, `POST /me/password`, `DELETE /me` |
| Profile | `GET /me/profile`, `PATCH /me/profile`, `PUT /me/location`, `PUT /me/preferences`, `PATCH /me/settings`, `GET /interests`, `GET /profiles/:userId` |
| Photos | `GET/POST /me/photos`, `PUT/DELETE /me/photos/:photoId`, `POST /me/photos/reorder`, `GET /media/photos/:photoId?exp&sig` |
| Discovery & matching | `GET /discover`, `POST /swipes`, `POST /swipes/undo`, `DELETE /matches/:matchId` |
| Chat | `GET /conversations`, `GET /conversations/unread-count`, `GET/DELETE /conversations/:id`, `GET/POST /conversations/:id/messages`, `POST /conversations/:id/read` |
| Safety | `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` |
| Notifications | `GET /notifications`, `GET /notifications/unread-count`, `POST /notifications/read-all`, `POST /notifications/:id/read` |
| Realtime | `GET /realtime` (WebSocket; first frame `{"type":"auth","token":"…"}`) |

Errors always use the shape `{"error": "<message in French>", "code": "<stable_code>"}`. Lists use keyset
pagination (`?before=<id>&limit=n`) and return `nextCursor`.

Realtime events: `ready`, `message.created`, `conversation.read`, `match.created`, `match.ended`,
`notification.created`.

### Data model

```text
users ─┬─ sessions, password_reset_codes
       ├─ profiles (1:1)        first_name, birthdate, gender, bio, job, city, lat/lng (rounded), discoverable, show_distance
       ├─ preferences (1:1)     interested_in[], age_min/max, max_distance_km
       ├─ user_interests ── interests (catalog)
       ├─ photos                position 0..5 (0 = primary), random storage_key
       ├─ swipes                PK (swiper_id, target_id), action like|pass
       ├─ matches               (user_a_id < user_b_id) UNIQUE, ended_at/ended_by
       │    └─ conversations (1:1) ── conversation_participants (last_read_at, hidden_at) ── messages
       ├─ blocks                PK (blocker_id, blocked_id)
       ├─ reports               reporter/reported ON DELETE SET NULL (kept for moderation)
       └─ notifications
```

All foreign keys cascade from `users`, so account deletion is one `DELETE`. The CHECK constraints mirror the
service validation (genders, lengths, age range ≥ 18, distance, coordinates, canonical match ordering).
The indexes match the hot paths: e-mail lookup, swipe and reverse-swipe lookups, the discovery partial index,
the lat/lng range, messages by `(conversation_id, created_at)`, unread notifications, open reports.

### Key flows

- **Discovery**: one SQL query applies every hard rule (mutual gender and age preferences, bounding box plus
  haversine distance against both members' limits, no previous swipe, no pass on me, no block, profile complete
  with a photo). It returns a pool of up to 100 candidates; `WeightedRanker` orders them and the top `limit` are
  hydrated into cards with three batched queries (no N+1). To improve recommendations, implement `discovery.Ranker`.
- **Swipe**: the transaction takes `pg_advisory_xact_lock(pair)`, inserts the swipe (`ON CONFLICT DO NOTHING`
  rejects repeats), detects the reciprocal like, then creates the match, conversation and participants. After the
  commit, both members get a `match.created` event and a notification.
- **Chat access**: every chat call goes through `Access()`. The caller must be a participant, the match active,
  and no block present. Otherwise the API answers 404, so a stranger cannot even learn that a conversation exists.
- **Account deletion**: the password is re-checked, the rows are deleted (cascade), the photo files erased and the
  realtime connections closed.

### Security summary

- bcrypt passwords (8–72 bytes, letter + digit). Login timing is equalised to avoid account probing. Password
  reset does not reveal whether an account exists, and its codes are hashed, single-use, limited to 5 attempts
  and expire in 30 minutes.
- Access tokens last 15 minutes (HS256, strict validation). Refresh tokens are 256-bit random values, stored
  hashed and rotated on every use. A reset or password change revokes the other sessions.
- Rate limits cover login, register, reset, refresh, swipes, messages, uploads and reports. `TRUSTED_PROXIES`
  stops `X-Forwarded-For` spoofing.
- Every input is validated server-side. Free text is normalised (control characters stripped, length limits).
  Strict UUID, cursor and limit parsing.
- Security headers include CSP `default-src 'none'`, `nosniff` and `frame-ancestors 'none'`. CORS uses an
  allow-list, and the WebSocket origin is checked.
- The logger redacts bearer tokens, JWTs and secrets.

## 3. App (`apps/mobile`)

```text
App.tsx                    providers: SafeArea, QueryClient, Auth, navigation, toast host
src/api/                   client.ts (single fetch gateway, single-flight refresh on 401), session.ts
                           (SecureStore / sessionStorage on web), endpoints.ts, types.ts, queryKeys.ts
src/auth/                  AuthProvider (boot, sign in/out, session expiry)
src/realtime/              RealtimeClient (backoff, token refresh on 4001) + provider syncing React Query caches
src/navigation/            guest stack → onboarding → tabs (Découvrir, Matchs, Activité, Profil) + modal screens
src/features/              auth, onboarding, discover (deck, card, match modal, detail), matches, chat,
                           notifications, profile (edit, photos, preferences, privacy, account), safety
src/ui/                    design-system primitives (Text, Button, TextField, Chip, IconButton, Section, States, Toast…)
src/theme.ts               colour, spacing, radius and typography tokens
src/utils/                 pure helpers (format.ts, unit tested), device helpers (picker, location), confirm dialog
e2e/run.mjs                Playwright scenario (driven by scripts/e2e.sh)
```

UI code never calls `fetch`: everything goes through `src/api`. Server state lives in React Query (feeds and
conversations use infinite queries; realtime events patch the caches). Local state stays in components.

Accessibility: every control has an accessible label and role, and selection state uses `aria-checked`.
Swipe gestures always have button equivalents, live regions announce errors, and text contrast is at least 4.5:1.

## 4. Decisions log

| Decision | Why |
| --- | --- |
| Removed billing, posts/comments/votes, organisations, `/users` list | The product is free, these modules did not fit a dating app, and listing every user was a privacy leak. |
| New migration set + `schema_migrations` | The old runner re-executed every file on each boot. The schema had to change radically anyway. |
| Conversations belong to a match | No conversation can exist without a mutual like, which closes off spam by design. |
| Report ⇒ block | Protects the reporter immediately, whatever the moderation delay. |
| Undo only for passes, within 10 min | Undoing a like could cancel a match the other person already saw. |
| Never re-show passed profiles; hide people who passed on you | Avoids repetitive feeds and pointless likes. |
| Signed media URLs instead of auth headers | Image components cannot send headers on every platform; the URLs expire and stay cache friendly. |
| In-memory hub and rate limiter behind interfaces | The simplest correct option for one instance, with a documented path to scale out. |
| Dark-only theme | A deliberate, consistent identity for evening use; one palette to keep accessible. |
| Web support + Playwright | Makes the critical journey testable automatically in CI without device farms. |
