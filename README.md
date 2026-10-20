# Lumen

Lumen is a **100% free dating app**: no subscriptions, no paywall, no paid boosts, no artificial like
limits, no mandatory ads. Two people can create an account and a profile, add photos, discover
nearby profiles, like or pass, match when interest is mutual, chat in real time, get notified, manage
privacy, block or report someone, and delete their account.

It started as the `go-react-saas` boilerplate; the Go / PostgreSQL / Expo foundation was kept and the
SaaS demo features were archived (`archive/legacy-saas/`, not built).

## Stack

| Layer | Technology |
|---|---|
| API | Go 1.24, Gin, PostgreSQL 16, `pgxpool` (no ORM), JWT HS256 + rotating refresh tokens, bcrypt, gorilla/websocket |
| Mobile | Expo 54 / React Native 0.81, TypeScript (strict), React Navigation, TanStack Query, `expo-image-picker`, `expo-location` |
| Infra | Docker, Docker Compose, GitHub Actions |

Architecture, data model and security decisions: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).
REST + WebSocket contract: [`docs/API.md`](docs/API.md).

## Features

- **Accounts**: register, login, refresh, logout, password recovery by emailed code, account deletion (password required; erases every row and photo file).
- **Profile**: first name, age (18+ enforced server side), gender, bio, city, up to 10 interests, 1-6 photos (reorder, primary, delete, replace), approximate location.
- **Preferences & privacy**: genders you want to meet, age range, max distance, visible in discovery on/off, show distance on/off.
- **Discovery**: swipe deck (with Like / Pass buttons), full profile view, mutual preference filtering, ranking by shared interests then distance, never shows the same person twice.
- **Matches**: mutual like creates exactly one match + conversation (race-safe); unmatch.
- **Chat**: real-time (WebSocket), timestamps, read receipts, unread counts, per-user conversation deletion, 2000 char messages.
- **Notifications**: matches and messages, stored and pushed live; unread badge.
- **Safety**: block (dissolves match, hides both ways), report with reason, rate limiting, protected photos.

## Quick start (Docker)

```bash
cp .env.example .env
# put a random value in JWT_SECRET (>= 32 chars):
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(openssl rand -hex 32)/" .env
docker compose up --build -d
docker compose exec api ./seed          # optional demo users (demo-*@demo.invalid / demo-password-123)
curl http://localhost:18080/health      # {"status":"ok"}
```

Migrations run automatically when the API starts. Photos live in the `uploads_data` volume.

## Local development without Docker

Requirements: Go 1.24+, Node 20+, PostgreSQL 14+.

```bash
# database
createdb app    # or any database; then set DATABASE_URL

# API
cd services/api
export DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
export UPLOAD_DIR=./data/uploads
go run ./cmd/api                 # migrates, then listens on :8080
go run ./cmd/seed                # optional demo data (never runs with APP_ENV=production)
go run ./cmd/seed -reset         # remove demo data

# mobile
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080 npx expo start
```

On a physical phone **never use `localhost`**: set `EXPO_PUBLIC_API_URL` to your computer's LAN IP and
make sure nothing else uses the API host port (`docker compose ps`, `curl http://<LAN_IP>:18080/health`).
`npm run start:phone` detects the LAN IP for you. Android emulators reach the host at `10.0.2.2`
(the dev fallback in `src/api/client.ts` already uses it).

Demo logins after seeding: `demo-ines@demo.invalid` (woman) and `demo-hugo@demo.invalid` (man), password `demo-password-123`.
Demo accounts use the reserved `.invalid` TLD so they can never be real users or receive mail.

## Environment variables

See [`.env.example`](.env.example).

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | yes | - | PostgreSQL connection string |
| `JWT_SECRET` | yes | - | HS256 signing key, >= 32 chars, not a placeholder |
| `PORT` | no | `8080` | API port |
| `APP_ENV` | no | `development` | `development`, `production` or `test`. Production never logs recovery codes and the seed refuses to run |
| `UPLOAD_DIR` | no | `./data/uploads` | Photo storage directory |
| `ALLOWED_ORIGINS` | no | empty | CORS / WebSocket origins for web clients |
| `TRUSTED_PROXIES` | no | none | Reverse proxies whose `X-Forwarded-For` is trusted (rate limiting uses the client IP) |
| `APP_NAME` | no | `Lumen` | Name used in emails |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USERNAME` `SMTP_PASSWORD` `SMTP_FROM` | for recovery in prod | - | Email delivery. Without `SMTP_HOST`, development prints the recovery code in the API log |
| `TEST_DATABASE_URL` | for tests | - | Database used by integration tests (they use throwaway schemas) |
| `EXPO_PUBLIC_API_URL` | mobile | dev fallback | API base URL baked into the app (public by design, never put secrets in it) |

## Database and migrations

SQL files in `services/api/internal/platform/db/migrations` are embedded in the binary, applied in
order inside a transaction when the API starts, and recorded in `schema_migrations`. Add a new
`NNN_name.sql`; never edit an applied one. To start over locally: `docker compose down -v`.

## Tests and quality gates

```bash
# API: needs a PostgreSQL; integration tests create and drop their own schema
cd services/api
export TEST_DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable
gofmt -l .            # prints nothing
go vet ./...
go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npx tsc --noEmit && npm run lint && npm run format:check && npm test

# Black-box journey against a running stack (after `docker compose up` + seed)
API_URL=http://localhost:18080 node scripts/smoke-api.js
```

What is covered: `internal/app` runs the full journey over HTTP against PostgreSQL (register -> profile ->
discover -> like -> match -> chat -> notifications), conversation permissions, blocks, reports, account
deletion, preference filtering, concurrent likes, JWT forgery, refresh replay, password recovery, rate
limiting, photo upload validation and the WebSocket. Unit tests cover age rules, distance bucketing,
image processing, limiter, config and cursors. The mobile app has unit tests (validation, pagination, realtime cache
reducers, API client token refresh) and component tests (auth form, swipe deck). A Maestro flow for the
UI journey lives in `apps/mobile/e2e/maestro` (needs a simulator/device; not run in CI).

## Build and deploy

- **API**: `docker build -t lumen-api services/api` produces a small non-root image (`./api`, `./seed`) with a health check on `/health`. Run behind a TLS-terminating reverse proxy, set `APP_ENV=production`, a strong `JWT_SECRET`, `SMTP_*`, `TRUSTED_PROXIES`, and mount a persistent volume on `/data/uploads` (or implement `storage.Store` for object storage).
- **Mobile**: set `EXPO_PUBLIC_API_URL` to your public HTTPS API and build with EAS (`eas build`) or `npx expo export`.
- Scaling out: the rate limiter and the WebSocket hub are in-process. Behind several API replicas, move them to a shared store (Redis) behind the existing `middleware.Limiter` / `realtime.Publisher` seams.

## Key decisions

- **Free forever**: nothing in the schema or API models plans, credits or limits; swipes are only rate limited against abuse (240/min).
- **Reused foundation**: kept Go/Gin/pgx, sessions with rotating refresh tokens, Expo/React Query. Replaced Tamagui with a small in-repo UI kit (less native/build surface).
- **Privacy by construction**: coordinates stored at ~1 km, never returned; distance is bucketed; photos are re-encoded (metadata stripped) and only served to people allowed to see the profile.
- **One eligibility rule**: the same SQL decides what discovery shows and what a swipe accepts, so clients cannot bypass preferences or blocks.
- **Swipes are final**: a like or pass is recorded once (409 afterwards) and survives unmatching, so the pair never resurfaces. Undo/rewind was left out for simplicity.
- **Match = 1:1 conversation**; hiding a conversation is per user and clears that user's visible history; a new message brings it back.
- **No "who liked you" list and no like notifications**: only matches and messages notify, which keeps the feature set honest and paywall-free.
- **Recovery by emailed code** (no deep-link infrastructure needed): the app has an "enter code" screen.
- **Single-instance realtime** with REST as source of truth; clients refetch after reconnect.
- **Access tokens are checked against their session** (cached 10 s) so logout, password reset and deletion cut access almost immediately.
- **Legacy code archived, not deleted**: `archive/legacy-saas/`.

## Known limitations

- Moderation is report collection only (`reports` table, `status` column); there is no admin UI.
- Push notifications (APNs/FCM) are not implemented; notifications are in-app and live over WebSocket.
- The UI was verified through type-checking, lint, unit/component tests and an Expo bundle export, not on a physical device in this environment.
