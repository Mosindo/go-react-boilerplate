# Tandem

Tandem is a **100 % free dating app** — no subscription, no paywall, no paid boost, no artificial like limit, no ads.
Two people create a profile, discover each other nearby, like or pass, match when the interest is mutual, and chat in real time.

This repository is a full-stack monorepo: a Go API, a PostgreSQL database and a React Native (Expo) app that also runs on the web.

## Features

| Area | What works |
| --- | --- |
| Accounts | Register, login (JWT + rotating refresh tokens), password recovery by e-mail code, logout, **account deletion** that erases every associated row |
| Onboarding | First name, birth date (18+ enforced server-side), gender, who you want to meet, location, bio, interests, photos |
| Photos | Up to 6, upload from the gallery, replace, delete, reorder, main photo; every file is validated, resized and **re-encoded** (EXIF/GPS stripped) |
| Discovery | Swipeable deck (drag or buttons), photo carousel, full profile view, shared interests, approximate distance |
| Matching | Like / pass, mutual like = match, no duplicates, processed profiles never reappear, race-free |
| Chat | Matches open a 1:1 conversation: text messages, timestamps, read receipts, unread counters, realtime via WebSocket, hide a conversation, unmatch |
| Notifications | In-app notifications for matches and messages (live + persisted), unread badges |
| Safety | Block (removes the match and hides both users from each other), report with reason, blocked list |
| Privacy | Pause your profile, hide your distance, coordinates rounded to ~1 km, distances shown in 5 km buckets |

## Stack

- **API**: Go 1.23+, [Gin](https://gin-gonic.com), PostgreSQL via `pgxpool`, no ORM, JWT HS256, bcrypt, `gorilla/websocket`
- **App**: Expo SDK 54 / React Native 0.81, TypeScript (strict), React Navigation, TanStack Query
- **Infra**: Docker + Docker Compose, GitHub Actions

## Repository layout

```text
.
├─ services/api/
│  ├─ cmd/api/               HTTP server + router wiring (and integration tests)
│  ├─ cmd/seed/              demo data (demo.tandem.local users only)
│  └─ internal/
│     ├─ platform/           config, db (+ migrations), middleware, errors, realtime hub, mail, logger
│     └─ features/           auth, profiles, photos, discovery (matching), chat, notifications, safety
│                            each: handler -> service -> repository (+ model, routes)
├─ apps/mobile/
│  ├─ src/api/               typed API client (the only place that calls the network)
│  ├─ src/auth, realtime/    session + WebSocket providers
│  ├─ src/screens, components, navigation, theme, lib
│  ├─ tests/                 unit tests (node --test)
│  └─ e2e/web-flow.mjs       browser end-to-end test (Playwright)
├─ scripts/e2e-web.sh        one-command E2E run
├─ docker-compose.yml        PostgreSQL + API
└─ docs/ARCHITECTURE.md      design decisions and data model
```

## Quick start (Docker)

```bash
cp .env.example .env
# edit .env: set POSTGRES_PASSWORD and JWT_SECRET (openssl rand -hex 32)
docker compose up -d --build
curl http://localhost:18080/health        # {"status":"ok"}

# optional: load demo users (development only)
docker compose exec api ./seed
```

Migrations run automatically when the API starts. `./seed` creates 12 clearly fake users `demoN@demo.tandem.local` (password `DemoPass123`) around Paris; demo 1 and 2 already "like" a few demo profiles so matches appear quickly. Remove them with `docker compose exec api ./seed -reset`. The seed refuses to run when `APP_ENV=production`.

### Run the app

```bash
cd apps/mobile
npm ci
# Physical phone: NEVER use localhost. Use the LAN IP of the machine running the API.
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npx expo start
# or: npm run start:phone   (auto-detects the LAN IP)
# web:  npx expo start --web
```

Before declaring the setup "ready", verify that nothing else hijacks the API port:

```bash
docker compose ps
curl http://<LAN_IP>:18080/health
```

If the port is taken, change `API_HOST_PORT` in `.env` and use the same value in `EXPO_PUBLIC_API_URL`.

Password recovery in development: without `SMTP_HOST`, the reset code is written to the API log (`docker compose logs api | grep password_reset_dev`).

## Local development without Docker

```bash
# PostgreSQL 14+ running locally, then:
export DATABASE_URL='postgres://app:secret@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
cd services/api
go run ./cmd/api            # migrations applied on start
go run ./cmd/seed           # optional demo data
```

## Environment variables

See [`.env.example`](.env.example).

| Variable | Required | Description |
| --- | --- | --- |
| `DATABASE_URL` | yes | PostgreSQL connection string (Compose builds it from the `POSTGRES_*` values) |
| `JWT_SECRET` | yes | ≥ 32 random characters; placeholders such as `change-me` are rejected |
| `PORT` | no | API port, default `8080` |
| `APP_ENV` | no | `development` (default), `production`, `test`. Production requires `SMTP_HOST` and `APP_BASE_URL` |
| `APP_BASE_URL` | prod | Public URL of the deployment |
| `ALLOWED_ORIGINS` | no | CORS allow-list for the **web** build (native apps are not subject to CORS) |
| `TRUSTED_PROXIES` | no | Reverse proxies whose `X-Forwarded-For` is trusted (needed for correct per-IP rate limiting behind a proxy) |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | prod | Password recovery e-mails |
| `EXPO_PUBLIC_API_URL` | app | API base URL as seen from the device (no secret ever goes in the app) |

## Tests and quality gates

```bash
# API (needs PostgreSQL; tests are skipped if no database is configured)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
gofmt -l .                  # must print nothing
go vet ./...
go test -race -count=1 ./...
go build ./cmd/api ./cmd/seed

# App
cd apps/mobile
npm ci
npm run typecheck && npm run lint && npm run format:check && npm test

# Browser end-to-end: register -> onboarding -> discover -> like -> match -> realtime chat -> delete account
npm install --no-save playwright && npx playwright install chromium     # once
DATABASE_URL=postgres://... ../../scripts/e2e-web.sh
# API-only smoke test against a running stack:
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
```

The Go integration suite covers auth (including `alg=none` tokens and session revocation), profile validation, photo rules and access control, discovery filters, like/pass/match (idempotency, contradicting decisions, a **concurrent mutual-like race**), chat permissions (outsiders always get 404), hide/unmatch/block, reports, WebSocket events, rate limiting and full account deletion.

## Build and deployment

- API image: `docker build -t tandem-api ./services/api` (static binary, non-root user, includes `seed`).
- Run it behind a TLS-terminating reverse proxy with `APP_ENV=production`, a managed PostgreSQL, strong secrets and SMTP credentials; set `TRUSTED_PROXIES` to the proxy address.
- Mobile builds: `npx expo export --platform web` (static web) or EAS Build for iOS/Android with `EXPO_PUBLIC_API_URL` pointing at the public API (HTTPS).
- The API keeps WebSocket connections in memory: run **one instance**, or put a Postgres `LISTEN/NOTIFY`/Redis bridge behind `realtime.Hub.Publish` before scaling out (see ARCHITECTURE.md).

## Main technical decisions

Details in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md). In short:

- **Kept** the existing Gin/pgx/JWT foundation, the `handler → service → repository` layering and the session/refresh-token design; **removed** the generic SaaS modules that did not serve a dating product (Stripe billing, organisations, posts, comments, files, generic chat).
- **Match = conversation**: one `matches` row per pair (`user_a_id < user_b_id`), which is also the chat room. Fewer tables, no inconsistent states.
- **Photos live in PostgreSQL** (`bytea`, ≤ ~300 KB each after re-encoding). Deleting an account therefore deletes its photos atomically, and there is no separate file store to secure. Photos are served only through an authenticated, access-checked endpoint.
- **One query defines eligibility and ranking** (`discovery/repository.go`), used both to build the deck and to authorise a like/pass. Improving recommendations means editing that single SQL.
- **Privacy by design**: coordinates are rounded to ~1 km at write time, never returned to other users, and distances are bucketed per 5 km.
- **WebSocket auth with single-use tickets** (30 s) so access tokens never appear in URLs; REST stays the source of truth and the client refetches on every (re)connect.
- **Rate limiting** is in-memory token buckets (per IP on auth routes, per user on likes, messages, uploads, deck).
- **Reports** are stored (`reports` table) for moderation; there is no admin UI in this version — review them with SQL (`SELECT * FROM reports WHERE status = 'open'`).
- **Interface language** is French; strings are inline in the screens for simplicity.

## License

See repository settings. Contributions: see [CONTRIBUTING.md](CONTRIBUTING.md).
