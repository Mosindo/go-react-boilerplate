# Ember

Ember is a modern, **100% free** dating app: create a profile, discover people near you, like or pass, match when the interest is mutual, and chat in real time. There are no subscriptions, paywalls, paid boosts, like limits or ads — and the code is built so none of those can sneak in.

- **Mobile app** — React Native (Expo) + TypeScript, iOS / Android
- **API** — Go + Gin + PostgreSQL (`pgx`), JWT sessions, WebSocket for realtime
- **Infra** — Docker Compose for local and self-hosted deployments

## Features

| Area | What you get |
| --- | --- |
| Onboarding | Sign up, sign in, password recovery, 3-step profile creation (about you, photos, preferences) |
| Profile | First name, age (18+ enforced server-side), gender, bio, city, up to 10 interests, up to 6 photos (reorder, main photo, replace, delete) |
| Discovery | Swipe deck (drag or buttons), photo carousel, full profile view, mutual-preference filtering, approximate distance only |
| Matching | Like / pass, atomic mutual-like detection, no duplicates, handled profiles never reappear, unmatch |
| Chat | Match-gated conversations, realtime delivery (WebSocket), sent/read receipts, unread counters, local conversation deletion |
| Notifications | Persisted match / message notifications, unread badges, realtime toasts |
| Safety | Block, report (with reasons), unmatch, hide profile, hide distance, account deletion that wipes all data |

## Repository layout

```text
.
├─ apps/mobile/            Expo app (src/api, src/state, src/screens, src/components, src/ui, src/theme)
├─ services/api/
│  ├─ cmd/api/             API entry point (+ end-to-end integration tests)
│  ├─ cmd/seed/            Demo data command (never runs in production)
│  └─ internal/
│     ├─ features/         auth, profiles, matching, chat, notifications, safety
│     │                    (handler -> service -> repository, + model.go, routes.go)
│     └─ platform/         config, db (+ migrations), middleware, httpx, realtime, storage, imaging, mailer, logger
├─ docs/ARCHITECTURE.md    Architecture and data model
└─ docker-compose.yml      PostgreSQL + API
```

## Quick start (Docker)

```bash
cp .env.example .env
# put a random secret in .env:   JWT_SECRET=$(openssl rand -base64 48)
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# optional: create demo accounts (demo01@demo.invalid ... / DemoPass123)
docker compose exec api ./seed
```

The API listens on **http://localhost:18080** (host port 18080 → container 8080).

### Run the mobile app

```bash
cd apps/mobile
npm ci
# Physical phone: use your computer's LAN IP, never "localhost"
echo "EXPO_PUBLIC_API_URL=http://192.168.1.20:18080" > .env
npx expo start
```

Scan the QR code with Expo Go, or press `a` / `i` for an emulator. If `EXPO_PUBLIC_API_URL` is omitted in development, the app derives the API host from the Expo dev server (and uses `10.0.2.2` on the Android emulator).

Before declaring your setup healthy, check that nothing else is using the API port:

```bash
docker compose ps
curl http://<LAN_IP>:18080/health
```

If port 18080 is taken, change the host port in `docker-compose.yml` and `EXPO_PUBLIC_API_URL` together.

## Local development without Docker

```bash
# 1. PostgreSQL 14+ (any way you like), then:
createdb ember
cd services/api
export DATABASE_URL="postgres://localhost/ember?sslmode=disable"
export JWT_SECRET="$(openssl rand -base64 48)"
export UPLOAD_DIR=./data/uploads
export APP_ENV=development
go run ./cmd/api            # migrations are applied automatically on start
go run ./cmd/seed           # optional demo data
```

## Environment variables

API (`.env.example`):

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `JWT_SECRET` | yes | – | HS256 signing key, ≥ 32 chars, placeholders rejected |
| `DATABASE_URL` | yes | – | PostgreSQL connection string (Compose sets it for you) |
| `PORT` | no | `8080` | HTTP port |
| `APP_ENV` | no | `production` | `development` enables the dev mailer (reset codes in the log) and the seed command; the default is the safe one |
| `UPLOAD_DIR` | no | `./data/uploads` | Where processed photos are stored (a Docker volume in Compose) |
| `ALLOWED_ORIGINS` | no | – | Browser origins allowed for CORS / WebSocket (web builds only) |
| `TRUSTED_PROXIES` | no | – | Reverse-proxy IPs/CIDRs trusted for `X-Forwarded-For` (rate limiting) |
| `POSTGRES_PASSWORD` | no | `ember` | Compose database password |
| `DATABASE_URL_TEST` | tests | – | Database used by the Go integration tests |

Mobile (`apps/mobile/.env.example`): `EXPO_PUBLIC_API_URL`. Nothing secret ever ships in the app.

## Database and migrations

SQL migrations live in `services/api/internal/platform/db/migrations` and are embedded in the binary. On start the API takes an advisory lock, applies the files not yet recorded in `schema_migrations` (each in its own transaction) and continues — safe with several instances.

Tables: `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `blocks`, `reports`, `conversations`, `conversation_participants`, `messages`, `notifications`. Constraints, foreign keys with `ON DELETE CASCADE`, unique pairs and the indexes needed by each hot query are defined there; see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

### Seed data

`go run ./cmd/seed` (or `docker compose exec api ./seed`) creates 12 clearly labelled demo accounts: e-mails `demoNN@demo.invalid` (a reserved, undeliverable domain), `users.is_demo = TRUE`, procedurally generated abstract pictures (no real faces). It is idempotent, `go run ./cmd/seed -reset` removes them, and it refuses to run when `APP_ENV=production`. Sign in with one of them (password `DemoPass123`, override with `SEED_PASSWORD`), or register your own account and like the demo profiles — they do not auto-like back, so use two accounts to see a match.

## Tests and quality

```bash
# API (integration tests need PostgreSQL; they skip themselves without DATABASE_URL_TEST)
cd services/api
gofmt -l . && go vet ./...
DATABASE_URL_TEST="postgres://localhost/ember_test?sslmode=disable" go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npx tsc --noEmit && npm run lint && npm run format:check && npm test

# Critical path against a running API (register → profile → photo → discover → like → match → realtime chat → block → delete)
API_URL=http://localhost:18080 npm run e2e:api
```

What is covered: auth/sessions/recovery/deletion (service tests), JWT validation, rate limiter, image pipeline, profile validation, and a full HTTP flow with permissions, blocks, reports, notifications, WebSocket delivery and a simultaneous-likes race test; on mobile, the API client (token refresh), validation helpers, realtime cache merge, and the discovery / login / profile-form screens. CI (`.github/workflows/ci.yml`) runs all of it plus the Expo bundle check and the live API smoke test.

## Build and deploy

- **API image**: `docker build -t ember-api services/api` (static binary, non-root user, healthcheck on `/health`, uploads in `/data/uploads`).
- **Production checklist**: leave `APP_ENV` unset or `production`, set a strong `JWT_SECRET`, use a managed PostgreSQL with backups and a persistent volume for `UPLOAD_DIR` (or replace `platform/storage` with an object-storage implementation), TLS termination in front of the API (WebSocket upgrade must be allowed), and `TRUSTED_PROXIES` for your proxy. Wire a real `mailer.Mailer` in `cmd/api/main.go` — the default production mailer drops messages, so password recovery e-mails need it.
- **Mobile**: build with EAS (`eas build`) after setting `EXPO_PUBLIC_API_URL` to your public HTTPS API URL; the bundle itself is checked in CI with `expo export`.

## Key technical and product decisions

- **Kept**: Go/Gin/pgx (no ORM), JWT + rotating refresh tokens, Expo + React Navigation + React Query. **Removed**: Stripe billing, posts, comments, generic files, organizations/tenants, Tamagui (unused) — they contradicted the "free, dating" scope.
- **Matching is transactional**: a swipe takes a per-pair advisory lock, inserts the swipe, and creates the match + conversation in the same transaction when the other side already liked. Matches are stored once per pair (`user_a < user_b`). Swipes are permanent: liked or passed profiles never come back (there is no undo, by design).
- **Discovery rules** (server side): the candidate must be discoverable, have a photo, match the viewer's gender/age/distance preferences *and* the viewer must match theirs; blocked users (either direction) and handled profiles are excluded. Ordering: people who already liked you first, then most recently active, then nearest. The ordering lives in one SQL constant (`candidateOrder`) so a smarter recommender can replace it later. If either side has no location, the distance filter is skipped.
- **Location privacy**: coordinates are rounded to ~1 km before storage, never returned by any endpoint, and other users only see a distance rounded *up* to a 5 km step (and only if the owner allows it).
- **Photos**: JPEG/PNG only, ≤ 8 MB, ≤ 10 000 px, ≥ 200 px, fully decoded and re-encoded to a ≤ 1280 px JPEG (strips EXIF/GPS and polyglot payloads). Files get random names, are never served statically, and every read goes through an authenticated endpoint that refuses blocked users. Photo URLs are unguessable UUIDs.
- **Chat security**: every query joins `conversation_participants`; non-members get a 404 (no existence leak). Blocking or unmatching deletes the match and, by cascade, the conversation. "Delete chat" is local (`cleared_at`): the other person keeps their history. The WebSocket is server-push only and authenticates with a first-frame token (not in the URL); all writes go through REST.
- **Account recovery**: 10-character single-use codes valid for 1 hour, stored hashed, bound to the e-mail, rate limited; every session is revoked after a reset or password change. In development the code is written to the API log (no SMTP needed).
- **Deletion**: `DELETE /me` (password required) removes the user; foreign keys cascade to profile, photos (files removed afterwards), swipes, matches, messages, notifications and sessions. Reports filed *by or about* the user are removed too.
- **Abuse protection**: global and per-route in-memory rate limits (auth, uploads, swipes, messages, reports). Behind several instances, add a shared limiter at the gateway.
- **Age gate**: users must be 18+; enforced at registration confirmation (UI), profile creation (server) and in preference ranges.
- **Not included (yet)**: push notifications (in-app + realtime only), e-mail verification, photo moderation, an admin UI for reports (they are stored with a status for a future back office), horizontal scaling of the WebSocket hub.
