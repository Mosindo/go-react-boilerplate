# Lueur — Architecture

Lueur is a monorepo with a Go API (`services/api`) and an Expo / React Native client (`apps/mobile`)
that runs on iOS, Android and the web.

## 1. Backend

Feature-first Go with Gin. Dependencies point inward: `handler → service → repository`.

```text
services/api/
├─ cmd/api/        main.go (server, graceful shutdown) · router.go (wiring) · integration_test.go
├─ cmd/seed/       demo data (refuses APP_ENV=production, uses @demo.invalid)
└─ internal/
   ├─ platform/
   │  ├─ config/       env parsing and validation (JWT secret strength, SMTP consistency)
   │  ├─ db/           pgxpool connection + embedded migrations runner (schema_migrations)
   │  ├─ middleware/   security headers, CORS, request id/logging, auth (JWT + live session check), rate limiter
   │  ├─ httpx/        error envelope, pagination helpers, user id accessor
   │  ├─ logger/       structured, redacted logging
   │  ├─ realtime/     WebSocket hub: single-use tickets, per-user fan-out
   │  ├─ storage/      Storage interface + local-disk implementation
   │  └─ mailer/       SMTP / log / discard mailers (account recovery)
   └─ features/
      ├─ auth/          register, login, refresh rotation, logout, recovery, change password, delete account
      ├─ profiles/      profile, preferences, interests, privacy, public profile view
      ├─ photos/        upload/replace/delete/reorder, authenticated serving, image sanitising
      ├─ matching/      Recommender (discovery), swipes, match creation, unmatch
      ├─ chat/          conversations, messages, read state, local hide
      ├─ moderation/    blocks, reports (with evidence snapshot)
      └─ notifications/ stored notifications, realtime push
```

Cross-feature calls go through small interfaces declared by the consumer (`Notifier`, `CardLoader`,
`realtime.Publisher`, `middleware.SessionChecker`, `auth.PhotoEraser`), wired in `cmd/api/router.go`.

### Error envelope

Every error is `{"error": "<message>", "code": "<machine_code>"}`. The client maps codes to French
messages (`apps/mobile/src/api/errors.ts`). Internal errors are logged and never leaked.

### Endpoints

| Area          | Endpoints                                                                                                                                                         |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Health        | `GET /health`                                                                                                                                                     |
| Auth          | `POST /auth/register` `/auth/login` `/auth/refresh` `/auth/logout` `/auth/password/forgot` `/auth/password/reset` · `GET /me` · `PUT /me/password` · `DELETE /me` |
| Profile       | `GET /me/profile` · `PUT /me/profile` `/me/preferences` `/me/privacy` · `GET /interests` · `GET /profiles/:userId`                                                |
| Photos        | `GET /me/photos` · `POST /me/photos` (multipart `file`) · `PUT /me/photos/:id` (replace) · `PUT /me/photos/order` · `DELETE /me/photos/:id` · `GET /photos/:id`   |
| Matching      | `GET /discover?limit=` · `POST /swipes {targetId, action}` · `DELETE /discover/passes` · `DELETE /matches/:id`                                                    |
| Chat          | `GET /conversations?before=&limit=` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` · `DELETE /conversations/:id`                       |
| Safety        | `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports`                                                                                                   |
| Notifications | `GET /notifications?before=&limit=` · `POST /notifications/:id/read` · `POST /notifications/read-all`                                                             |
| Realtime      | `POST /ws/ticket` → `GET /ws?ticket=` — events: `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`                                   |

Lists are paginated: cursor (`before` = timestamp of the last item) for conversations, messages and
notifications; discovery returns the next batch (max 20) because processed profiles leave the feed.

### Data model

```text
users ─┬─ sessions, password_resets
       ├─ profiles (1:1) ── preferences (1:1) ── user_interests ── interests
       ├─ photos (≤ 6, position 0 = main)
       ├─ swipes (actor, target, like|pass)          PK (actor, target)
       ├─ matches (user_a < user_b)                  UNIQUE (user_a, user_b)
       │     └─ conversations (1:1) ── conversation_participants (last_read_at, hidden_at) ── messages
       ├─ blocks (blocker, blocked)
       └─ notifications
reports (reporter_id, reported_id nullable → survive account deletion, evidence JSONB)
```

All user-owned tables cascade on user deletion, so `DELETE FROM users` is a complete erasure (photo files
are removed by the service). `reports` keep only the reason and evidence with user ids set to NULL.

### Discovery query

One SQL statement: mutual preference filter (gender, age, distance on both sides), latitude bounding box
before the haversine distance, exclusion of processed/blocked profiles, ranking by shared interests,
proximity, recent activity and "already liked me". Photos and interests for the page are then loaded with
two batched queries (no N+1). Swap `matching.Recommender` to change the algorithm.

### Security model

- Passwords: bcrypt (72-byte cap), constant-time-ish login for unknown emails.
- JWT: HS256 only (`WithValidMethods`), mandatory `exp`, strict claims, 15-minute lifetime; the middleware also
  verifies the session row is active, so logout / password change / account deletion take effect immediately.
- Refresh tokens: random, stored hashed, single use (compare-and-swap rotation).
- Authorisation is always server-side; unknown or foreign resources answer 404.
- Rate limits: per IP (auth, recovery), per user (swipes, messages, uploads, reports, tickets), per email (login).
- Uploads: 8 MB cap, content sniffing, pixel cap, re-encode, random file names, private serving.
- Headers: nosniff, frame deny, no-referrer, HSTS behind TLS; CORS allow-list; WebSocket origin check.

## 2. Client

```text
apps/mobile/src/
├─ api/         client.ts (fetch wrapper, single-flight refresh) · services.ts · types.ts · tokenStore.ts · errors.ts
├─ auth/        AuthContext (booting → anonymous / authenticated)
├─ realtime/    RealtimeProvider (WebSocket → TanStack Query cache)
├─ navigation/  RootNavigator: auth stack | onboarding gate | tabs + stack
├─ screens/     Auth, Onboarding, Discover, Messages, Chat, Notifications, ProfileDetail, Profile*
├─ components/  ui kit, AuthImage, Avatar, SwipeDeck, PhotoGrid, ProfileForm, Sheet, SafetySheets, Toast
├─ theme/       light/dark palettes, spacing, radius
├─ i18n/        fr.ts (all copy)
└─ lib/         format, validation, cities, device (location/photo picker), deckBus
```

- No screen calls `fetch`: all network access goes through `src/api`.
- Server state lives in TanStack Query (`["me","profile"]`, `["conversations"]`, `["messages", id]`, …); realtime
  events patch or invalidate those caches.
- Private photos: native `Image` with a Bearer header (refresh + retry on 401); on the web a blob URL cache.
- Swipe deck: `PanResponder` + `Animated` (no gesture library); optimistic removal with rollback on failure.
- Tokens: SecureStore on native, `localStorage` on the web.

## 3. Quality gates

See README ("Tests et qualité") and `.github/workflows/ci.yml`: gofmt, vet, Go tests with PostgreSQL,
build, TypeScript, ESLint, Prettier, Vitest, web build, Playwright E2E, Docker smoke test.
