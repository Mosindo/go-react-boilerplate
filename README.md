# Lueur

**Lueur** is a free dating app, free all the way: no subscription, no paywall, no paid boosts,
no daily like limit and no ads. Two people can sign up, build a profile with photos and interests,
discover nearby members, like or pass, match when the interest is mutual, and chat in real time.
The app also covers notifications, privacy settings, blocking, reporting and permanent account deletion.

The repository is a monorepo:

| Path | Content |
| --- | --- |
| `services/api` | Go REST + WebSocket API (Gin, PostgreSQL via `pgxpool`, no ORM) |
| `apps/mobile` | Expo / React Native app (iOS, Android and web), TypeScript |
| `scripts/e2e.sh` | Browser end-to-end test against a throw-away stack |
| `docs/ARCHITECTURE.md` | Architecture, data model and technical decisions |

---

## Features

- **Accounts**: e-mail + password sign-up, login, rotating refresh tokens, password reset by e-mailed
  code, password change, logout, permanent account deletion (profile, photos, likes, matches, messages, notifications).
- **Onboarding**: first name, birthdate (18+ enforced server-side), gender, who you want to meet,
  1–6 photos, bio, job, up to 10 interests, approximate location (device position rounded to ~1 km, or a city picked from a list).
- **Discovery**: a swipeable card deck (with like, pass and undo-last-pass buttons). Eligibility is
  mutual: genders, age ranges and distances must suit both people. Profiles already swiped, people who passed on you,
  blocked members and paused profiles never show up. Ranking weighs shared interests, proximity and recent activity.
- **Matching**: a mutual like creates exactly one match and one conversation. This holds even under concurrent likes.
- **Chat**: realtime messages over WebSocket, timestamps, read receipts ("Lu"), unread counters,
  cursor pagination, icebreakers. You can delete a conversation on your side only, unmatch, block, or report.
- **Notifications**: in-app feed and badges for new matches and unread messages, pushed in real time.
- **Privacy & safety**: pause your profile, hide your distance, block and unblock, report. A report also blocks the member.
  Exact coordinates, e-mail and birthdate are never exposed. Photo metadata (EXIF/GPS) is stripped. Photos are served through short-lived signed URLs.

---

## Tech stack

**API**: Go 1.24 · Gin · PostgreSQL 16 · pgx/pgxpool · JWT HS256 (golang-jwt) · bcrypt · gorilla/websocket.
**App**: Expo SDK 54 · React Native 0.81 · React 19 · TypeScript (strict) · React Navigation 7 · TanStack Query 5 ·
expo-secure-store · expo-image(-picker/-manipulator) · expo-location.
**Tooling**: Docker / Docker Compose · GitHub Actions · ESLint · Prettier · Node test runner · Playwright.

---

## Quick start (Docker)

Prerequisites: Docker with Compose, Node.js 20+ for the app.

```bash
cp .env.example .env
# Edit .env: set JWT_SECRET (e.g. `openssl rand -base64 48`) and POSTGRES_PASSWORD.
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

Optional demo data (16 clearly flagged demo profiles around Paris):

```bash
docker compose exec api ./seed --yes-demo
# add  --like-email you@example.com  so every demo profile likes you (instant matches when testing)
```

Then start the app:

```bash
cd apps/mobile
cp .env.example .env      # set EXPO_PUBLIC_API_URL (LAN IP for a physical phone)
npm install
npm run start             # press i / a / w for iOS, Android or web
```

> On a physical phone, never use `localhost`: set `EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080`.
> For the web build, add its origin (e.g. `http://localhost:8081`) to `ALLOWED_ORIGINS`.

---

## Running the API without Docker

```bash
cd services/api
export DATABASE_URL='postgres://lueur:<password>@localhost:5432/lueur?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export ALLOWED_ORIGINS='http://localhost:8081'
go run ./cmd/api                       # migrations run automatically at startup
go run ./cmd/seed --yes-demo           # optional demo data
```

## Environment variables

API (`.env.example` at the repository root):

| Variable | Required | Description |
| --- | --- | --- |
| `JWT_SECRET` | yes | ≥ 32 random characters. Placeholder-like values are rejected at startup. |
| `DATABASE_URL` | yes | PostgreSQL connection string. Docker Compose builds it from the `POSTGRES_*` values. |
| `APP_ENV` | no | `development` (default), `test` or `production`. Production requires SMTP. |
| `PORT` | no | Listening port, default `8080` (published as `18080` by Compose). |
| `UPLOAD_DIR` | no | Photo storage directory, default `./data/uploads` (`/data/uploads` volume in Docker). |
| `ALLOWED_ORIGINS` | no | Comma-separated browser origins for CORS and WebSocket (web build). |
| `TRUSTED_PROXIES` | no | Reverse-proxy IPs/CIDRs whose `X-Forwarded-For` is trusted. Empty means none (protects rate limiting). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | prod | Delivery of password-reset codes. Without `SMTP_HOST` (dev/test only), codes are written to the API log. |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | Compose | Database container credentials. |
| `SEED_DEMO_PASSWORD` | no | Password of demo accounts; random (printed once) if empty. |

App (`apps/mobile/.env.example`): `EXPO_PUBLIC_API_URL`, the public API URL baked into the build.
Never put secrets in `EXPO_PUBLIC_*`.

## Database & migrations

- SQL migrations live in `services/api/internal/platform/db/migrations` and are embedded in the binary.
- They run at startup in order. Each one is applied once and recorded in `schema_migrations`. A Postgres
  advisory lock prevents concurrent API replicas from migrating at the same time.
- To change the schema, add a new numbered file (`003_...sql`) and never edit an applied migration.
- A database still holding the schema of the original generic boilerplate is detected and refused with a clear
  message. Use a fresh database (`docker compose down -v`).

## Demo data

`seed` (`go run ./cmd/seed --yes-demo` or `./seed` in the container) creates demo accounts that **cannot be confused with real members**:

- e-mails use the reserved, non-routable `.invalid` domain (`demo-01@demo.lueur.invalid` …);
- `users.is_demo = TRUE`, and the app shows a "Profil de démonstration" badge on every demo card and profile;
- demo photos are abstract gradients (no real faces);
- the command refuses to run when `APP_ENV=production`, and `--reset` deletes all demo accounts and their files.

## Tests & quality checks

```bash
# API: unit tests + HTTP integration tests (need a PostgreSQL database; skipped otherwise)
cd services/api
gofmt -l .                                      # must print nothing
go vet ./...
DATABASE_URL_TEST='postgres://postgres@localhost:5432/app_test?sslmode=disable' go test -race ./...
go build ./cmd/api && go build ./cmd/seed

# App
cd apps/mobile
npm ci
npm run typecheck && npm run lint && npm run format:check && npm test
npm run build:web                                # production web bundle in dist/

# End-to-end (two real users in Chromium against a fresh DB + API + web build)
./scripts/e2e.sh                                 # needs Go, Node, Python 3, PostgreSQL and the `playwright` package
```

What is covered:

- **Go unit tests** cover password, e-mail and profile validation, age rules, location rounding and distances,
  ranking, image validation (type sniffing, decompression bombs, EXIF stripping), signed media URLs, strict JWT
  validation (`alg`, `exp`, issuer, `alg=none`), the rate limiter, storage path traversal and config.
- **Go integration tests** (`cmd/api/main_integration_test.go`) exercise the real HTTP router with PostgreSQL:
  auth lifecycle, password reset, profile validation, photos, the full register → profile → discover → like →
  match → chat → read receipts → local deletion → block → report → account deletion flow. They also check
  permissions (strangers get 404), the realtime WebSocket, concurrent mutual likes creating a single match,
  undo, unmatch, pause and rate limiting.
- **App unit tests** (`npm test`) cover formatting/validation helpers and the realtime message cache merge.
- **E2E** (`apps/mobile/e2e/run.mjs`) runs the critical journey in the browser for two users: register → onboarding
  (photo upload, city) → discovery → like → match → realtime chat → read receipt → notifications → report →
  logout/login → account deletion.

CI (`.github/workflows/ci.yml`) runs all of the above on every push and pull request, and also builds the Docker image.

## Build & deployment

- **API**: `docker build -t lueur-api ./services/api`. This makes a small Alpine image with the `api` and `seed`
  binaries. It runs as a non-root user, has a `/health` healthcheck, and keeps photos in the `/data/uploads` volume.
  Put it behind an HTTPS reverse proxy (WebSockets on `/realtime` must be forwarded), set `APP_ENV=production`,
  `TRUSTED_PROXIES`, `ALLOWED_ORIGINS` and the SMTP variables, and back up both PostgreSQL and the uploads volume.
- **App**: build native binaries with [EAS Build](https://docs.expo.dev/build/introduction/) (`eas build`) with
  `EXPO_PUBLIC_API_URL` pointing to the HTTPS API. The web version is a static bundle (`npm run build:web` → `dist/`)
  you can host on any static host (add its origin to `ALLOWED_ORIGINS`).
- **Scaling notes**: rate limiting and the realtime hub are in-memory, which is fine for a single API instance.
  To run several replicas, swap them for a shared store (Redis or Postgres LISTEN/NOTIFY) behind the existing
  `realtime.Publisher` interface. Photo storage sits behind `storage.Store`, so an S3-compatible backend can replace the local disk.

## Main technical decisions

- **Kept the boilerplate's foundations** (Go/Gin/pgx feature slices, refresh-token sessions, Expo + React Query) and
  removed what conflicts with the product: Stripe billing (the app is free), posts/comments/votes, multi-tenant
  organisations, and the public `/users` listing (privacy).
- **Matches are race-safe**: a transaction-scoped advisory lock per pair plus unique constraints guarantee a single match.
- **Location privacy**: coordinates are rounded to ~1 km on the device and again on the server, used only for SQL
  distance filtering, and never returned. Members see "À 6 km" (5 km buckets beyond 20 km), or nothing if the owner hides it.
- **Photos**: the type is checked by magic bytes, the dimensions are checked before decoding, the image is downscaled
  and re-encoded to JPEG (metadata stripped), stored under random keys, and served via HMAC-signed URLs that expire within 6–12 h.
- **Realtime**: one WebSocket per client. The token goes in the first frame (never in URLs or logs), and the connection
  closes when the access token expires. Clients resync over REST on reconnect.
- **Moderation**: reports are stored and kept, anonymised, even after account deletion. A report always blocks the
  reported member for the reporter. Moderators review the `reports` table (status `open` → `reviewed`/`dismissed`).
- **No artificial limits**: likes are never capped. Per-user throttles only stop scripted abuse (e.g. 120 swipes/min).
- **Dark, warm identity** designed for evening use; the UI is in French.

More detail: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
