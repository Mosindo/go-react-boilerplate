# Aube — Architecture

Aube is a free dating app: a Go API (`services/api`) and an Expo / React Native client (`apps/mobile`, also runnable on the web).

```
apps/mobile            Expo + React Native + TypeScript (iOS, Android, web)
  src/api              typed HTTP client (single entry point to the network), refresh-token handling
  src/auth             session context + secure token storage
  src/realtime         WebSocket provider, applies server events to the react-query cache
  src/screens          auth / onboarding / main screens
  src/components       design-system primitives + shared form sections
  e2e/run.js           Playwright browser journey (web build + live API)
services/api
  cmd/api              HTTP server (one Gin server, graceful shutdown)
  cmd/seed             demo data (@demo.invalid accounts), refuses APP_ENV=production
  internal/platform    config, db (+ tracked migrations), middleware, logger, httpx, realtime hub, mailer, storage
  internal/features    auth, profiles, discovery, chat, notifications, safety
```

Layering is `handler -> service -> repository`; SQL lives only in repositories. Small features (`notifications`, `safety`) keep handler, service and repository in one file.

## Data model (PostgreSQL, `internal/platform/db/migrations`)

| Table | Purpose |
|---|---|
| `users`, `sessions`, `password_resets` | credentials, rotating refresh sessions, recovery tokens (hashed) |
| `profiles`, `preferences` | identity (first name, birth date, gender, bio, city, **coarse** lat/lon), who/age/distance wanted |
| `photos` | ordered (`position`, 0 = main) references to files on disk |
| `interests`, `user_interests` | catalogue + user selection |
| `likes` | one row per ordered pair, `action in (like, pass)`; the primary key makes decisions final |
| `matches` | one canonical row per pair (`user_a_id < user_b_id`) |
| `conversations`, `conversation_participants`, `messages` | 1:1 chat per match, read marker and local-clear marker per participant |
| `blocks`, `reports`, `notifications` | safety and in-app feed |

Everything cascades from `users`, so deleting an account removes all personal data. Reports are kept for moderation with the account references set to `NULL`.
Indexes cover the hot paths (`users.email`, `messages(conversation_id, created_at, id)`, `notifications(user_id, created_at)`, likes received, participants by user, …).

## Key flows

* **Auth** – bcrypt passwords; 15 min HS256 access JWT (`uid`, `sid`, mandatory `exp`, algorithm pinned); 30 day opaque refresh token stored hashed and **rotated** on each refresh. The access token is also checked against the session row, so logout / password change / account deletion take effect immediately.
* **Discovery** – a single SQL query (`discovery/repository.go`) filters by mutual orientation, mutual age range, mutual distance, visibility, blocks, and excludes anything already liked/passed. Ordering lives in one constant (`discoveryRanking`): people who liked you first, then shared interests, then recent activity — the extension point for a smarter recommender. Cards are built in batches (no N+1).
* **Matching** – `POST /swipes` runs in one transaction behind a per-pair advisory lock: insert the decision (duplicate → `409`), look for the reciprocal like, create match + conversation + notifications. Simultaneous likes yield exactly one match (covered by a test).
* **Chat** – REST for history/sending (membership enforced inside the SQL), WebSocket `/ws` for push. The first frame authenticates (`{"type":"auth","token":…}`), so tokens never end up in URLs. Events: `message`, `read`, `match`, `unmatch`, `notification`. The hub is in-memory (single instance); swap `Publish` for Postgres `LISTEN/NOTIFY` or Redis to scale horizontally.
* **Photos** – multipart upload ≤ 5 MB; magic-byte/decoder validation (JPEG/PNG only), dimension limits against decompression bombs, re-encoded to a ≤ 1280 px JPEG (strips EXIF/GPS), random file names. Files are served only through short-lived HMAC-signed URLs generated when a viewer is allowed to see the profile.
* **Location** – optional. Coordinates are rounded to 2 decimals (~1 km) before storage, never returned by the API; others only see an approximate distance (1 km floor, whole km < 10 km, then steps of 5 km) and only if the owner leaves "show distance" on.
* **Safety** – blocking is symmetric in effect and removes any match + conversation; reports (reason + details) are stored for moderation; per-user/IP rate limits protect auth, swipes, messages, uploads and reports.

## Frontend decisions

React Navigation (native stack + bottom tabs), TanStack Query for server state, no global store. All network access goes through `src/api`. Swipe uses `PanResponder` + `Animated` (no extra native dependency); buttons offer the same actions for accessibility. Theme follows the system light/dark setting.
