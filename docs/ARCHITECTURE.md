# Architecture

Ember is a monorepo: an Expo mobile app (`apps/mobile`) and a Go API (`services/api`) backed by PostgreSQL.

## Backend

```text
cmd/api            server entry point (config, migrations, graceful shutdown)
cmd/seed           demo data (refuses APP_ENV=production, only touches users.is_demo)
internal/server    router wiring + HTTP integration tests (package server_test)
internal/platform  config · db (+migrations) · middleware · logger · errors · pagination · realtime · storage
internal/features  auth · profiles · matching · chat · notifications · safety
```

Rules: one Gin HTTP server; handlers only bind/validate transport concerns and map errors; services own business rules; repositories own SQL; shared concerns live in `platform`; feature packages never import each other except through small exported helpers (`profiles.CardColumns/ScanCard` for profile cards, `notifications.InsertTx` to write notifications inside another feature's transaction).

### HTTP API

Errors use `{"error": "<message>", "code": "<stable_code>"}`. Lists use keyset pagination (`limit`, opaque `cursor` → `nextCursor`).

| Area | Endpoints |
| --- | --- |
| Health | `GET /health` |
| Auth | `POST /auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` · `/auth/password/forgot` · `/auth/password/reset` · `GET /me` · `POST /me/delete` |
| Profile | `GET/PUT /profile` · `PUT /preferences` · `GET /interests` · `GET /profiles/:id` |
| Photos | `POST /profile/photos` (multipart `photo`, optional `position` to replace) · `PUT /profile/photos/order` · `DELETE /profile/photos/:id` · `GET /photos/:id` |
| Matching | `GET /discover` · `POST /swipes` · `GET /matches` · `DELETE /matches/:id` · `GET /likes/received` |
| Chat | `GET /conversations` · `GET /conversations/:id` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` · `DELETE /conversations/:id` (hide for me) |
| Notifications | `GET /notifications` · `POST /notifications/:id/read` · `POST /notifications/read-all` |
| Safety | `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` |
| Realtime | `GET /ws` — first message `{"type":"auth","token":"<access token>"}`; server events: `ready`, `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`, `pong` |

### Authentication

Access token: JWT HS256, 15 min, claims `uid`/`sid`/`exp` (strict method check, mandatory expiry). Refresh token: 32 random bytes, only its SHA-256 is stored, rotated on every use (an old token is rejected). Passwords: bcrypt (max 72 bytes). Login spends the same bcrypt time for unknown e-mails.

### Data model

```text
users 1─1 profiles           profiles(first_name, birth_date, gender, bio, city, lat/lng ~1km, is_visible)
users 1─1 preferences        preferences(interested_in[], min_age, max_age, max_distance_km, show_distance)
users ─< user_interests >─ interests
users 1─< photos             photos(position 0..5, UNIQUE(user_id, position) deferrable) — position 0 = primary
users ─< swipes >─ users     swipes(from_user, to_user, action like|pass) PRIMARY KEY(from_user, to_user)
matches(user_a < user_b, UNIQUE)  1─1  conversations  1─< messages
conversations 1─< conversation_members(last_read_at, hidden_at)
blocks(blocker_id, blocked_id) · reports(reporter_id, reported_id, reason, details, context jsonb, status)
notifications(user_id, type new_match|new_message, actor_id, conversation_id, read_at)
sessions · password_resets · schema_migrations
```

Indexes cover every hot path: `users.email` (unique), `messages (conversation_id, created_at DESC, id DESC)`, `notifications (user_id, created_at DESC, id DESC)` + partial unread index, `conversation_members (user_id)`, `swipes` incoming-likes partial index, `profiles (is_visible, gender)` and `(lat, lng)`, `matches (user_b)`, `photos (user_id, position)`. Listing endpoints aggregate photos/interests in the same query (no N+1) and are always paginated or bounded.

### Discovery

`Discover` loads the viewer (profile + preferences + at least one photo, else `409 profile_incomplete`) and excludes in SQL: self, hidden profiles, anyone already swiped (hence matched), blocked pairs in either direction, profiles without photo, and anyone whose own preferences (gender, age range, distance) exclude the viewer. Ordering is the single extension point for recommendation work: people who already liked the viewer, then shared interests, then distance.

### Security checklist (enforced server-side)

18+ validation · strict input normalisation and length limits · private photos with sanitising re-encode · membership checks on every conversation call (non-members get `404`, indistinguishable from a missing id) · block checks on profile/photo/discover/swipe · per-IP and per-user rate limits · `TrustedProxies` allow-list (no spoofable client IP) · security headers · secrets only via environment · log redaction of tokens/passwords.

## Mobile

State: server state in TanStack Query (keys `profile`, `conversations`, `messages/:id`, `notifications`, `likes`, `blocks`), session in `AuthProvider` + `api/client.ts` (single-flight token refresh, secure storage), live updates in `RealtimeProvider` (WebSocket → cache updates, reconnect with backoff, full resync after reconnect/foreground). No screen performs network calls directly: everything goes through `src/api`.

Navigation: signed out → Welcome/Login/Register/Forgot password · signed in without a complete profile → Onboarding (profile → photos → preferences) · otherwise tabs Discover / Likes / Messages / Profile plus stack screens (chat, profile detail, edit profile, photos, preferences, notifications, settings, blocked users, report).

Design: "Ember" identity — raspberry accent, warm paper (light) / deep plum (dark) neutrals, tokens in `src/theme/tokens.ts`.
