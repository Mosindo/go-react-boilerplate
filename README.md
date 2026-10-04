# Lumen — a free dating app

Lumen is a complete, **100 % free** dating app: create an account, build a profile with photos, discover people near you, like or pass, get a **match** when the interest is mutual, and chat in **real time**.

There is no subscription, no paywall, no paid boost, no artificial like limit and no ads. Every feature is available to everyone.

> The repository started life as the generic `go-react-saas` boilerplate. The Go/Gin/PostgreSQL/Expo foundations were kept; the SaaS-specific parts (organizations, Stripe billing, posts/comments) are no longer wired in.

## Features

| Area | What you get |
|---|---|
| Account | Register, login, refresh-token rotation, logout, password recovery by email, change password, **delete account** (removes all personal data) |
| Profile | First name, birth date (18+ enforced server side, immutable), gender, bio, city, up to 10 interests, up to 6 photos |
| Preferences | Who you want to meet, age range, max distance, hide from discovery, hide distance |
| Photos | Upload with server-side validation, re-encoding (EXIF stripped) and thumbnails, reorder, main photo, delete — served only to people allowed to see them |
| Discovery | Compatible, nearby, never-seen profiles; photo carousel; like / pass (button or swipe); full profile view |
| Matching | Mutual like ⇒ match, atomically and exactly once; repeated likes are idempotent; passed profiles never come back |
| Chat | Match-only conversations, realtime delivery over WebSocket, timestamps, read receipts, unread counters, hide a conversation for yourself, unmatch |
| Safety | Block (removes the match, hides both sides from each other), report with reasons, rate limiting, strict authorization on every route |
| Notifications | In-app notifications (matches) with unread badge, pushed live over the WebSocket |
| Privacy | Coordinates stored at ~1 km precision and **never** returned; distances shown as approximate 5 km steps; birth date and email never exposed |

## Tech stack

- **API**: Go 1.24, Gin, PostgreSQL 16, `pgx/v5` (`pgxpool`), JWT HS256 + rotating refresh tokens, bcrypt, `gorilla/websocket`, `golang.org/x/image`. No ORM.
- **Mobile / web**: Expo SDK 54, React Native 0.81, TypeScript (strict), React Navigation, TanStack Query.
- **Infra**: Docker, Docker Compose, GitHub Actions.

## Repository layout

```text
.
├─ apps/mobile/                 Expo app (iOS / Android / web)
├─ services/api/
│  ├─ cmd/api/                  HTTP server + router wiring (+ integration tests)
│  ├─ cmd/seed/                 Demo-data command (development only)
│  └─ internal/
│     ├─ platform/              config, db + migrations, middleware, realtime hub, storage, mail, errors, httpx
│     └─ features/              auth, profiles, photos, discovery, chat, safety, notifications
│        └─ <feature>/          handler.go → service.go → repository.go (+ model.go, routes.go)
├─ infra/docker-compose.yml     postgres + api
├─ docs/                        API.md (contract), ARCHITECTURE.md
└─ .env.example                 every environment variable, documented
```

Dependency rule: `handler → service → repository`. Handlers only bind/validate transport concerns and call the service; services hold the business rules; repositories hold the SQL. Features talk to each other through small interfaces wired in `cmd/api/router.go`.

## Quick start (Docker)

```bash
cp .env.example .env
# put a random secret in .env:   JWT_SECRET=$(openssl rand -base64 48)
docker compose up --build            # postgres + api on http://localhost:18080
curl http://localhost:18080/health   # {"status":"ok"}

# optional demo accounts (development only, see "Demo data")
docker compose exec api ./seed
```

Then start the app:

```bash
cd apps/mobile
cp .env.example .env                 # set EXPO_PUBLIC_API_URL
npm ci
npm run start                        # press w for web, or scan the QR code with Expo Go
```

> **Physical phone**: never use `localhost`. Set `EXPO_PUBLIC_API_URL=http://<your-LAN-IP>:18080`, and check `curl http://<LAN-IP>:18080/health` from another device. If something already uses port 18080, change the host port in `infra/docker-compose.yml` (e.g. `28080:8080`) **and** `EXPO_PUBLIC_API_URL`.

## Local development without Docker

Requirements: Go 1.24+, Node 20+, PostgreSQL 16.

```bash
createdb app && createdb app_test
export DATABASE_URL='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export UPLOAD_DIR=./data/uploads PORT=18080

cd services/api
go run ./cmd/api          # applies migrations on boot, listens on :18080
go run ./cmd/seed         # optional demo accounts
```

## Environment variables

Full, commented list in [`.env.example`](.env.example).

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `JWT_SECRET` | yes | – | ≥ 32 random chars; placeholder-looking values are rejected |
| `DATABASE_URL` | yes | – | PostgreSQL connection string |
| `APP_ENV` | no | `development` | `development`, `test` or `production` (production requires SMTP + `APP_BASE_URL`) |
| `PORT` | no | `8080` | HTTP port |
| `UPLOAD_DIR` | no | `./data/uploads` | Photo storage directory (use a persistent volume) |
| `APP_BASE_URL` | prod | – | Public URL used in password-reset emails |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USERNAME` `SMTP_PASSWORD` `SMTP_FROM` | prod | – | Outgoing email. In development, emails are printed in the API log |
| `ALLOWED_ORIGINS` | no | – | Browser origins allowed by CORS/WebSocket (web build only) |
| `TRUSTED_PROXIES` | no | – | Reverse proxies whose `X-Forwarded-For` is trusted (rate limiting is per client IP) |
| `MAX_PHOTO_BYTES` | no | `8388608` | Upload size cap |
| `DATABASE_URL_TEST` | tests | – | **Disposable** database for integration tests (its `public` schema is wiped) |
| `EXPO_PUBLIC_API_URL` | app | see `apps/mobile/.env.example` | API base URL baked into the app. It is public: never put a secret in `EXPO_PUBLIC_*` |

## Database & migrations

Migrations are plain SQL files in `services/api/internal/platform/db/schema/`, embedded in the binary and applied **in order, once**, at boot (tracked in `schema_migrations`, guarded by an advisory lock so several instances can start together). To change the schema, add the next numbered file; never edit one that has shipped.

Core tables: `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `photos`, `interests`, `user_interests`, `swipes` (likes/passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Constraints, foreign keys (with cascading deletes), uniqueness and the indexes required for list endpoints are in the SQL files.

## Demo data

`go run ./cmd/seed` (or `docker compose exec api ./seed`) creates 12 clearly fake accounts on the reserved, undeliverable domain `demo.lumen.invalid`, with generated gradient photos (no real people). Log in with `camille.01@demo.lumen.invalid` / `lumen-demo-pass` (override with `DEMO_PASSWORD`). `go run ./cmd/seed -purge` removes them. The command refuses to run when `APP_ENV=production`.

## Tests & quality gates

```bash
# Backend (needs a disposable Postgres for the integration tests)
cd services/api
export DATABASE_URL_TEST='postgresql://postgres:postgres@localhost:5432/app_test?sslmode=disable'
gofmt -l .            # must print nothing
go vet ./...
go test ./... -count=1
go build -o /dev/null ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci
npx tsc --noEmit
npm run lint
npm test
npm run build         # production web export
npm run e2e:smoke     # API smoke test against a running API (API_BASE_URL)
```

Backend integration tests run the real router against Postgres and cover: auth lifecycle (rotation, logout revocation, password reset), JWT hardening, profile/age validation, photo upload attacks, discovery filters and privacy of the payload, like/match rules (idempotency, race on simultaneous likes), chat permissions/read state/pagination, block/report, account deletion, rate limiting and the WebSocket.

## API

The full contract (endpoints, payloads, error format, WebSocket events) is in [`docs/API.md`](docs/API.md).

## Key technical decisions

- **Matching is transactional.** A swipe, the reciprocal-like check, the match, its conversation, both participants and both notifications happen in one transaction guarded by a pair-scoped advisory lock: two people liking each other at the same instant produce exactly one match. A repeated swipe returns the stored decision and never writes. `matches(user_a < user_b)` plus a unique constraint make duplicates impossible at the database level.
- **A "pass" is permanent**; an unmatch or block never resurrects a match. Simpler and safer than a cooldown, and it satisfies "already seen profiles never reappear".
- **Recommendations are isolated** in one SQL ranking expression (`discovery/repository.go`): hard filters (mutual gender interest, mutual age range, mutual max distance, not blocked, not swiped) then ranking (people who already liked you → shared interests → proximity → recent activity). Replace `rankScore` or the candidate query to evolve the algorithm without touching handlers.
- **Location minimisation.** Only a ~1 km rounded point is stored (optional; nothing is requested until the user taps "use my location"). Other users only ever receive `distanceKm` rounded **up** to 5 km steps, and nothing if either side hides distance.
- **Realtime = REST for writes, WebSocket for pushes.** Messages are persisted through `POST /conversations/:id/messages` (authorization and validation in one place); the in-memory hub pushes `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`. The socket authenticates with its first frame (tokens stay out of URLs and logs). The hub is single-instance; swap `realtime.Publisher` for Postgres `LISTEN/NOTIFY` or Redis to scale out.
- **Sessions are checked against the database** on every request, so logout, password reset, suspension and account deletion take effect immediately, not when the 15-minute access token expires. Refresh tokens are random, stored hashed, and rotated atomically (a replayed token fails).
- **Photos** are never stored as uploaded: content is sniffed (not trusting filename/MIME), dimensions checked before decoding (decompression-bomb guard), then re-encoded to JPEG (drops EXIF/GPS and trailing payloads) as a 1440 px image plus a 360 px thumbnail. Files live behind a `storage.Store` interface and are served only through the authenticated, visibility-checked endpoint.
- **Account deletion is real deletion**: foreign keys cascade through profile, photos (files removed), swipes, matches, conversations, messages, blocks and notifications. Reports are kept without the reporter/reported link (moderation evidence), containing no personal data. Messages you sent disappear for the other person too.
- **Rate limiting** is in-memory per instance (auth endpoints per IP, plus per email on login; uploads, messages and reports per user; generous limits on swipes as bot protection — not a quota). Put a shared limiter in front when running several instances.
- **No push notifications / email verification yet.** Push needs per-platform credentials; email verification needs a mail provider. Both are straightforward to add: `notifications` already stores events and `platform/mail` already sends mail.
- **Moderation** stores reports and lets users block; a moderation UI is out of scope. Accounts can be suspended by setting `users.status = 'suspended'` (sessions stop working immediately).

## Deployment

1. Provision PostgreSQL 16 and a persistent volume for `UPLOAD_DIR`.
2. Run the API image (`services/api/Dockerfile`) with `APP_ENV=production`, `DATABASE_URL`, `JWT_SECRET`, `APP_BASE_URL`, SMTP settings, `TRUSTED_PROXIES` (your load balancer) and `ALLOWED_ORIGINS` (if you serve the web build).
3. Terminate TLS in front of it (HSTS is emitted when `X-Forwarded-Proto: https`) and allow WebSocket upgrades on `/ws`.
4. Build the app with `EXPO_PUBLIC_API_URL=https://api.example.com` (`npm run build` for web, EAS Build for stores).
5. Back up PostgreSQL **and** the uploads volume together.

## License

See repository licensing terms. Contributions: see [`CONTRIBUTING.md`](CONTRIBUTING.md).
