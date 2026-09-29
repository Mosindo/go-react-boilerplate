# Alba — a free dating app

Alba is a complete, **100% free** dating app: no subscription, no paywall, no paid boost, no artificial like limit, no ads.
Two people can create an account and a profile, add photos, discover nearby profiles, like or pass, match on mutual interest,
chat in real time, get notifications, set preferences and privacy, block or report someone, and delete their account.

Built on the `go-react-saas` boilerplate (Go API + Expo/React Native client).

## Stack

- **API**: Go, Gin, PostgreSQL (`pgxpool`, explicit SQL, no ORM), JWT HS256 access tokens + rotating refresh tokens, WebSocket (gorilla) for realtime
- **Mobile**: Expo SDK 54 / React Native, TypeScript (strict), React Query
- **Infra**: Docker + Docker Compose, GitHub Actions

## Repository layout

```text
apps/mobile/            Expo client (src/api, screens, components, realtime, shared/ui, lib)
services/api/
  cmd/api               API entry point (composition root)
  cmd/seed              demo-data seeder (never runs in production)
  internal/platform     config, db + migrations, middleware, ratelimit, urlsign, storage, realtime, mailer, notify
  internal/features     auth, account, profiles, photos, discovery, matching, chat, notifications, safety
docs/API.md             the API contract shared by server and client
docs/ARCHITECTURE.md    architecture notes
infra/, docker-compose.yml
```

Each backend feature follows `handler -> service -> repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`).

## Quick start

Prerequisites: Docker, Node 22+, Go 1.24+ (only for running the API without Docker).

```bash
cp .env.example .env            # then set JWT_SECRET (openssl rand -base64 48)
docker compose up --build -d    # PostgreSQL + API on http://localhost:18080
curl http://localhost:18080/health
```

Migrations run automatically at API start (tracked in `schema_migrations`, guarded by an advisory lock).
**Upgrading from the old boilerplate database?** Use a fresh database (`docker compose down -v`); the schema was rebuilt for Alba.

### Demo data (development only)

```bash
docker compose exec api ./seed -yes            # 24 demo users, password DemoPass!2026 (override with SEED_PASSWORD)
docker compose exec api ./seed -purge -yes     # remove exactly the demo users
```

Demo accounts use the reserved domain `demo+<n>@seed.alba.invalid` and are placed around Lyon. The seeder refuses to run with `APP_ENV=production`.

### Run the API without Docker

```bash
cd services/api
export DATABASE_URL='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
go run ./cmd/api
```

### Run the mobile app

```bash
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080 npm run start   # physical phone: never use localhost
```

Emulators: Android uses `http://10.0.2.2:18080`, iOS simulator `http://localhost:18080` (development fallbacks).
Before calling a build "ready", check that nothing else hijacks the API port (`docker compose ps`, `curl http://<LAN_IP>:18080/health`) and run the smoke journey:

```bash
cd apps/mobile && EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080 node scripts/e2e-smoke.js
```

## Environment variables

See [.env.example](.env.example).

| Variable | Required | Purpose |
|---|---|---|
| `JWT_SECRET` | yes | HS256 signing key (>= 32 chars, no placeholder values); also derives the photo-URL signing key |
| `DATABASE_URL` | yes (outside Compose) | PostgreSQL connection string |
| `APP_ENV` | no | `development` (default) or `production` (requires `SMTP_HOST`) |
| `PORT` | no | API port, default `8080` (Compose publishes `18080`) |
| `UPLOAD_DIR` | no | Private photo storage directory, default `./data/uploads` (a Docker volume in Compose) |
| `APP_BASE_URL` | no | Base of password-reset links |
| `ALLOWED_ORIGINS` | no | CORS origins (Expo web only) |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | prod | Password-reset email. Without `SMTP_HOST` (dev) the reset token is written to the API log |
| `EXPO_PUBLIC_API_URL` | mobile | API base URL used by the app |
| `DATABASE_URL_TEST` | tests | Database for integration tests (they skip if unset) |

No secret ever ships in the client.

## Tests and quality gates

```bash
# Backend (integration tests need a PostgreSQL)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'
gofmt -l .            # must print nothing
go vet ./...
go test -race ./...
go build ./cmd/api

# Mobile
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test
```

`cmd/api/main_integration_test.go` drives the whole journey through the real router:
register -> profile -> location -> photo -> discover -> like -> match -> chat -> read -> block -> report -> delete account.
Feature packages have their own integration suites (permissions, concurrency of mutual likes, chat isolation, uploads, ...).
`apps/mobile/e2e/maestro/critical-flow.yaml` describes the UI journey for Maestro (needs a device/emulator).

## API

Full contract in [docs/API.md](docs/API.md). Highlights: `/auth/*`, `/me`, `/me/profile|preferences|location|photos`, `/discover`, `/swipes`, `/matches/:id`,
`/conversations[/:id/messages|read]`, `/notifications`, `/blocks`, `/reports`, `/ws/ticket` + `/ws`.

## Key technical decisions

- **Matching**: `swipes(swiper, target, like|pass)` with a primary key that makes duplicates impossible (409). A match is stored as an ordered pair (`user_a < user_b`, unique) and created in one transaction under a per-pair advisory lock, so concurrent mutual likes yield exactly one match, one conversation and two notifications. Blocks use the same lock.
- **Discovery**: one set-based SQL query (mutual gender/age/distance compatibility, exclusions for swiped/blocked/matched), a bounding-box prefilter, then a pluggable `Ranker` (default: shared interests, proximity, recent activity). Swiped profiles never come back.
- **Location privacy**: coordinates are rounded to ~1 km before storage; other users only see a bucketed distance (ceil to 1 km under 10 km, 5 km above), never coordinates, and can be hidden per user.
- **Photos**: JPEG/PNG/WebP sniffed by content, dimension-checked before decoding, resized to 1080 px and re-encoded as JPEG (metadata stripped), max 6 per user, stored privately behind a `Storage` interface. Files are served through HMAC-signed, expiring URLs issued only for photos the caller may see.
- **Chat**: only participants of a live match can read/write (404 otherwise); blocks cut messaging immediately; local conversation deletion only hides it. Realtime is a WebSocket authenticated by a 60 s one-purpose ticket; sending goes through REST; clients resync over REST on reconnect and poll while offline.
- **Security**: bcrypt over a SHA-256 pre-hash (no 72-byte truncation), strict JWT validation (HS256, mandatory expiry), refresh rotation, generic login/forgot responses, per-IP and per-user rate limits, request body limits, security headers, parameterised SQL only. Age gate: 18+ enforced server-side at registration.
- **Account deletion** removes the user row (every dependent table cascades) and the photo files.
- **Realtime scaling**: the hub is in-memory per process; for several instances back `realtime.Publisher` with Redis pub/sub or Postgres `LISTEN/NOTIFY`. Rate limiters are per-process as well.
- **Not included (yet)**: push notifications (in-app + realtime only), a moderation back-office (reports are stored with status `open|reviewed|dismissed`), email verification. Access tokens live 15 min; a revoked session's already-issued access token stays valid until expiry.

## Deployment

Build the image from `services/api/Dockerfile` (non-root, ships `api` and `seed`). Provide `DATABASE_URL`, a strong `JWT_SECRET`, `APP_ENV=production`, SMTP settings, a persistent volume for `UPLOAD_DIR`, and terminate TLS in front (HSTS is emitted when `X-Forwarded-Proto: https`). Set `EXPO_PUBLIC_API_URL` to the public HTTPS URL when building the mobile app (EAS or `expo export`).

## Legacy code pending removal

The boilerplate's `billing`, `posts`, `comments`, `files`, `users` feature packages, `scripts/*.ps1`, `.codex/`, `docs/agents/` and a few empty stub modules in `apps/mobile/src` (`api/platform.ts`, `api/types.ts`, `screens/{HomeScreen,ChatScreen}.tsx`, `theme/`, `tamagui.config.ts`, `shared/layout/BottomNavigation.tsx`) are no longer wired into the product. They are awaiting an explicit go-ahead to be deleted.
