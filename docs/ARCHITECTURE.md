# Lueur architecture

Technical reference for contributors and coding agents. The product overview, setup and decisions
are in the [README](../README.md).

## 1. Repository layout

| Path | Role |
| --- | --- |
| `services/api` | Go API (Gin, pgxpool, PostgreSQL). One HTTP server, one DB pool. |
| `apps/mobile` | Expo app for iOS, Android and web. |
| `docker-compose.yml` | Local PostgreSQL + API with persistent `postgres_data` and `uploads` volumes. |
| `scripts/qa.sh`, `scripts/qa.ps1` | Every quality gate in one command. |
| `.github/workflows/ci.yml` | Backend, mobile and end-to-end jobs. |

## 2. Backend

```text
services/api
├─ cmd/api            entry point: config, DB connect + migrations, graceful shutdown
├─ cmd/seed           demo data (refuses APP_ENV=production)
└─ internal
   ├─ app             composition root (wires platform + features) and HTTP integration tests
   ├─ platform
   │  ├─ config       environment parsing and validation
   │  ├─ db           pgxpool connection, embedded tracked migrations
   │  ├─ errors       typed application error (status + machine code + message)
   │  ├─ httpx        JSON binding, error responses, UUID/limit/cursor parsing
   │  ├─ middleware   auth (JWT + session check), rate limiting, body limits, CORS, security headers, logging
   │  ├─ media        storage interface (+ local disk), image sanitisation, signed media URLs
   │  ├─ realtime     WebSocket hub, single-use tickets, Publisher interface
   │  ├─ mailer       SMTP mailer and development log mailer
   │  └─ logger       structured, redacted handler logs
   └─ features        auth, profiles, photos, matching, chat, safety, notifications
```

Each feature has `model.go`, `repository.go` (all SQL), `service.go` (business rules, returns typed
errors), `handler.go` (binding + `httpx.Fail`), `routes.go`. Dependencies between features are
explicit and acyclic: `matching`, `chat` and `photos` use `profiles` (profile cards, photo view type);
`safety` uses `matching` (to break matches); `matching` and `chat` notify through `notifications`.
Realtime delivery goes through the `realtime.Publisher` interface.

### Request pipeline

`Recovery → SecurityHeaders → CORS → RequestID → metrics/logging → BodyLimit (1 MiB, 11 MiB for photo uploads)
→ route middleware (RequireUser, RateLimit) → handler`.

`RequireUser` validates the HS256 access token (method type and algorithm, mandatory `exp`, `iat`),
then checks in one query that the session is neither revoked nor expired and the account is not suspended.
The same query refreshes `users.last_active_at` at most every 5 minutes (used by ranking).

### Errors

Services return `*platformerrors.Error{Status, Code, Message}` for expected failures. `httpx.Fail` writes
`{"error": message, "code": code}`; any other error becomes an opaque 500 and is logged with the operation name.
Messages are user-facing (French); codes are stable for clients.

## 3. Data model

```text
users ─┬─ sessions, password_reset_tokens
       ├─ profiles (1:1)  ── user_interests ── interests
       ├─ preferences (1:1)
       ├─ photos (≤ 6, position 0 = main)
       ├─ swipes (swiper → target, like|pass, one per ordered pair)
       ├─ matches (user_a < user_b, one per pair) ── conversations (1:1) ─┬─ conversation_participants (2)
       │                                                                  └─ messages
       ├─ blocks (blocker → blocked)
       ├─ reports (reporter → reported, reporter kept as NULL after deletion)
       └─ notifications (group_key coalesces unread items)
```

Key constraints:

- `users.email` unique and normalised (`CHECK (email = lower(btrim(email)))`).
- `profiles`: gender enum, name/bio/job/city lengths, coordinates in range and both-or-none.
- `preferences`: `interested_in` non-empty subset of genders, `18 ≤ min_age ≤ max_age ≤ 99`, distance 1–500 km.
- `photos`: `UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED` so reorders apply atomically.
- `swipes`: primary key `(swiper_id, target_id)`, no self-swipe: a profile is never answered twice.
- `matches`: `CHECK (user_a_id < user_b_id)` + unique pair: no duplicate or mirrored matches.
- `conversations.match_id` unique, `ON DELETE CASCADE`: unmatching or blocking removes the conversation and messages.
- `reports`: one open report per reporter/reported pair.
- Every user-owned table cascades on `users` deletion.

Indexes cover the hot paths: e-mail lookup, session token hash, discovery (`profiles (gender, birthdate) WHERE discoverable`,
location box), likes received (`swipes (target_id, created_at) WHERE action = 'like'`), matches per user,
participants per user, messages `(conversation_id, created_at DESC, id DESC)`, notifications per user, blocks both ways.

Migrations are embedded SQL files applied once, in order, inside a transaction, tracked in `schema_migrations`
and serialised with an advisory lock. Never edit an applied migration; add a new numbered file.

## 4. Core flows

### Discovery

1. `Viewer` loads the caller's profile, preferences and coordinates; incomplete profiles get `409 profile_incomplete`.
2. `Candidates` applies every hard rule in SQL: active and discoverable, has a photo, mutual gender preference,
   mutual age range, mutual distance (bounding box on the location index, then haversine), not already swiped by
   the viewer, no block in either direction. It returns up to 200 rows with activity, distance, shared interests
   and whether the candidate already liked the viewer.
3. A `Ranker` (default `ScoreRanker`) orders them; the top N become profile cards, built in three batched queries
   (profiles, photos, interests) regardless of N.

### Swipe and match

`POST /swipes` runs in one transaction under `pg_advisory_xact_lock(hash(pair))`:
check target eligibility → insert swipe (`ON CONFLICT DO NOTHING`, conflict = `409 already_swiped`) → on a like,
look for the reverse like → insert the match and its conversation and participants. Because both directions take
the same lock, simultaneous mutual likes always produce exactly one match (covered by an integration test).
Both members receive `match.created` and a notification.

### Chat

Every chat query starts with `Membership`, which only returns a row when the caller participates in the
conversation and no block exists; otherwise the API answers 404 so outsiders cannot probe ids. Sending a message
updates `conversations.last_message_at` and the sender's `last_read_at` in the same transaction, publishes
`message.created` to both members and upserts a coalesced notification for the recipient. `POST …/read` moves
`last_read_at`, publishes `conversation.read` (read receipts) and clears the related notification.

### Realtime

Clients exchange their access token for a 30-second single-use ticket (`POST /realtime/ticket`) and open
`GET /realtime?ticket=…`. Tokens therefore never appear in URLs or proxy logs. The hub keeps one entry per
connection; `Publish` never blocks (a slow client misses events and resynchronises through REST). Browser
origins must be listed in `ALLOWED_ORIGINS`; native apps send no `Origin`.

### Photos

Upload → read at most 10 MiB → detect type from magic bytes (JPEG/PNG/WebP only) → check dimensions before
decoding (≤ 40 MP, ≥ 200×200) → decode → apply EXIF orientation → resize to ≤ 1440 px → re-encode JPEG (q82),
which drops all metadata → store under a random key → insert under a per-user advisory lock enforcing the
6-photo limit. Files are served by `GET /media/:key?exp&sig`, where `sig` is an HMAC (key derived from
`JWT_SECRET`) over key and expiry. URLs are only handed out by endpoints that already enforced visibility.

### Account deletion

`DELETE /me` re-checks the password, collects photo keys, deletes the user (all data cascades), purges files and
closes live sockets. Reports filed *by* the user remain with `reporter_id = NULL`.

## 5. Mobile app

```text
apps/mobile/src
├─ api/          typed endpoint functions; client.ts handles auth header, single-flight refresh on 401, timeouts, ApiError
├─ store/        token storage (SecureStore on native, memory only on web)
├─ hooks/        AuthProvider (session + profile status), React Query keys/hooks, useDeck, RealtimeProvider
├─ navigation/   RootNavigator: booting | signedOut (auth stack) | onboarding | app (tabs + stack)
├─ screens/      feature screens
├─ components/   cross-screen business components (match celebration, safety dialogs, profile facts)
├─ ui/           design system primitives
├─ theme/        light/dark palettes, typography and spacing tokens
└─ utils/        formatting, validation mirrors of server rules, location helper
```

- **Server state** lives in React Query. Realtime events update caches directly (`applyRealtimeEvent`) or
  invalidate the affected queries; when the socket is down, chat and badges fall back to light polling.
- **Discovery deck** is a local queue (`useDeck`): swipes update the UI immediately, requests run in the
  background, the queue refills before it empties and never repeats a profile.
- **Platform parity**: dialogs, action sheets and toasts are custom components (no `Alert`), so behaviour is
  identical on native and web; this is what makes the Playwright E2E meaningful.
- **Theming**: `ThemeProvider` follows the system scheme; components read colors through `useTheme()`.
- **Accessibility**: labelled buttons and inputs, roles and states, live regions for errors and toasts, swipe
  actions duplicated by buttons, 44 px+ touch targets.

## 6. Security checklist

- Passwords: bcrypt (cost 12); policy ≥ 8 chars with a letter and a digit; login timing does not reveal accounts.
- JWT: HS256 only, `exp` required, claims validated, session revocation enforced on each request.
- Refresh tokens: 256-bit random, stored as SHA-256, rotated on use, replay rejected.
- Password reset: 10-character code (~50 bits), hashed, 30 minutes, single use, 3 per hour, all sessions revoked.
- Input: strict server-side validation of every field; UUID path params; cursor and limit bounds; body size limits.
- Authorization: ownership/membership checks in SQL for photos, matches, conversations and notifications;
  visibility rules for profiles; blocks enforced in discovery, likes, profile view and chat.
- Abuse: rate limits on credentials (20/10 min per IP), uploads, swipes, messages and reports.
- Privacy: no e-mail, birthdate or coordinates in public payloads; coarse location; EXIF stripped.
- Headers: `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, HSTS behind HTTPS, restrictive CSP on media.
- Secrets: only from the environment; config rejects weak or placeholder `JWT_SECRET`.
