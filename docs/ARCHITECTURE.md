# Ember — Architecture

## Layout

```text
.
├─ apps/mobile/            Expo (iOS, Android, web) client
├─ services/api/           Go API (Gin + pgx)
├─ docker-compose.yml      PostgreSQL + API
└─ docs/
```

## Backend

`handler → service → repository`, one feature per directory under `internal/features`, shared concerns in `internal/platform`.

| Feature | Responsibility |
| --- | --- |
| `auth` | register/login, refresh rotation (CAS on the token hash), logout, password reset/change, account deletion, session check used by the middleware |
| `profiles` | profile, preferences, privacy, coarse location, interests catalog, onboarding status (`missing`) |
| `photos` | upload validation + re-encoding, ordering, replace/delete, access-controlled content |
| `discovery` | feed, public profile view, swipes, match creation, unmatch |
| `chat` | conversations list, messages, read state, local hide |
| `safety` | blocks and reports |
| `notifications` | in-app notifications (match, message), unread counter |
| `realtime` | in-memory hub, WebSocket ticket + upgrade |

Cross-feature reactions go through small `Events` interfaces implemented in `cmd/api/events.go` (the only place that
knows both notifications and the hub). Features never import each other.

`platform`: `config` (env + validation), `db` (pool, tracked migrations), `middleware` (auth, security headers, CORS,
request id/logging), `ratelimit`, `mailer` (log / SMTP), `httpx` (error envelope `{"error": "..."}`, binding, pagination
limits), `logger` (redacting).

### Request pipeline

`Recovery → SecurityHeaders → CORS → RequestID → logging → [RequireUser: JWT HS256 + exp + uid/sid claims + DB session check → per-user rate limit] → handler`.
Credentials endpoints use a stricter per-IP limiter (30/min, burst 10). `X-Forwarded-For` is ignored unless `TRUSTED_PROXIES` is set.

### Data model (`001_init.sql`)

- `users(email UNIQUE, lower-case CHECK)`, `sessions(token_hash UNIQUE)`, `password_resets(token_hash UNIQUE, used_at)`
- `profiles(user_id PK)` birth date, gender CHECK, bio/city length CHECKs, `latitude/longitude` (rounded, paired CHECK), `is_discoverable`, `show_distance`
- `preferences(user_id PK)` `interested_in text[]` CHECK ⊆ {man, woman, nonbinary}, `min_age ≥ 18`, `max_age ≤ 99`, `min ≤ max`, `max_distance_km` NULL = anywhere
- `interests`, `user_interests(PK user_id, interest_id)`
- `photos(user_id, position 0..8 UNIQUE DEFERRABLE, bytea JPEG)` — position 0 is the main photo
- `swipes(PK swiper_id, target_id; action like|pass; CHECK swiper <> target)` — final, one row per ordered pair
- `matches(user_a < user_b CHECK, UNIQUE(user_a, user_b), ended_at, ended_by)` — one row per pair
- `conversations(match_id UNIQUE)`, `conversation_participants(last_read_message_id, hidden_before_message_id)`, `messages(BIGSERIAL, body 1..2000)`
- `blocks(PK blocker_id, blocked_id)`, `reports(reason CHECK, status open|reviewed|dismissed)`
- `notifications(type match|message, conversation_id, read_at)`

All user-owned tables reference `users(id) ON DELETE CASCADE`: deleting an account removes every trace in one statement
(including reports it sent or received).

Indexes: `users(email)`, `messages(conversation_id, created_at, id)`, `conversation_participants(user_id, conversation_id)`,
`notifications(user_id, created_at DESC, id)` and a partial unread index, `swipes(target_id, swiper_id) WHERE action='like'`,
`blocks(blocked_id, blocker_id)`, `profiles(latitude, longitude)`, `user_interests(interest_id, user_id)`, `reports(reported_id, created_at)`.

### Discovery query

One SQL statement joins the viewer, a candidate and the candidate's preferences. A candidate is eligible when:
has a photo ∧ both genders match both preferences ∧ both ages fit both ranges ∧ distance (haversine on rounded coordinates)
fits both limits ∧ not blocked in either direction. The feed additionally requires: discoverable ∧ not swiped by the viewer ∧ has not
passed the viewer. `swipe` re-checks eligibility server-side so ids cannot be forged. Interests and photo refs for the whole page
are loaded with two extra queries (no N+1). Cost is O(page size) after the candidate scan; the scan is bounded by
the filters and indexes (a geohash / PostGIS prefilter is the next step at large scale).

### Matching atomicity

`Swipe` runs in a transaction guarded by `pg_advisory_xact_lock(hash(pair))`. Both directions of a pair serialize, so two
simultaneous likes always produce exactly one match, one conversation and two participants (covered by a concurrency test).

### Chat security

Every chat query joins `conversation_participants` on the caller, requires `matches.ended_at IS NULL` and no block either way.
Anything else → `404`. Sending takes `FOR SHARE` on the match row so an unmatch/block cannot interleave with a send.

### Photos

Content sniffing (never the client MIME), `DecodeConfig` pixel cap, full decode, CatmullRom downscale to 1280 px, flatten on white,
JPEG q82. Max 8 MB upload, 6 photos, advisory lock per user for position changes. `GET /photos/:id` requires auth and applies the
same visibility rule as profiles (owner, or no block ∧ (discoverable ∨ matched)).

### Realtime

`POST /ws/ticket` (authenticated) → single-use 30 s ticket → `GET /ws?ticket=`. Events: `match.new`, `match.ended`,
`message.new`, `message.read`, `notification.new`. Slow consumers are dropped; the client resyncs on reconnect.

## API summary

All routes except `/health` and `/auth/*` require `Authorization: Bearer <access token>`. Errors: `{"error": "message"}`.

| Method & path | Purpose |
| --- | --- |
| `POST /auth/register · /auth/login · /auth/refresh · /auth/logout` | session lifecycle |
| `POST /auth/password/forgot · /auth/password/reset` | recovery (always `202` on forgot) |
| `GET /me` · `POST /me/password` · `POST /me/delete` | account, change password, delete account (password required) |
| `GET/PUT /me/profile` · `PUT /me/preferences · /me/privacy · /me/location` · `DELETE /me/location` | profile & settings; each returns the fresh overview |
| `GET /interests` | catalog |
| `GET/POST /me/photos` · `PUT /me/photos/order` · `PUT/DELETE /me/photos/:id` · `GET /photos/:id` | photos |
| `GET /discover?limit=` · `GET /profiles/:id` · `POST /swipes` · `DELETE /matches/:id` | discovery |
| `GET /conversations` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` · `DELETE /conversations/:id` | chat |
| `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` | safety |
| `GET /notifications` · `GET /notifications/unread-count` · `POST /notifications/:id/read · /notifications/read-all` | notifications |
| `POST /ws/ticket` · `GET /ws` | realtime |

Pagination: lists use a bounded `limit` (validated) and opaque cursors (`conversations`, `notifications`) or `before=<id>` (messages).

## Client

- `src/api`: the only code that touches the network (`client.ts` handles auth header, single-flight refresh on 401, timeout, typed `ApiError`; `endpoints.ts` is the typed surface).
- `src/auth`, `src/realtime`: providers. The realtime provider writes server events straight into the TanStack Query cache and invalidates everything on (re)connect.
- `src/navigation`: auth stack → onboarding (driven by the server `missing` list) → tabs (Discover, Messages, Activity, Profile) + stack screens.
- `src/features`: reusable forms/widgets (swipe deck, photos editor, profile & preferences forms, safety dialogs).
- `src/ui`, `src/theme`: design system ("Ember": warm terracotta accent, light/dark, 44px+ touch targets, accessibility labels/roles).
- Photos are private: native uses `Image` with an `Authorization` header; web fetches blobs with the token (`ui/PhotoImage.tsx`).
- Tests: Vitest for pure logic (`src/lib`), Playwright E2E on the web build (`e2e/web-smoke.js`).
