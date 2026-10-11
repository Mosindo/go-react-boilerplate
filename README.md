# Ember — a free dating app

Ember is a complete, **100 % free** dating app: create a profile, add photos, discover people nearby, like or pass,
match when the interest is mutual, chat in real time, and stay in control with blocking, reporting, privacy settings
and one-tap account deletion.

There is **no subscription, paywall, paid boost, like limit or ad** anywhere in the code base. Who liked you is free
to see; every feature is available to every user.

Monorepo: a Go API (`services/api`) and an Expo / React Native client (`apps/mobile`, runs on iOS, Android and web).

## Features

| Area | What you get |
|---|---|
| Accounts | e-mail + password (bcrypt), 15 min JWT access token + rotating refresh token, password recovery by e-mail, change password, **delete account** (removes profile, photos, swipes, matches, messages, notifications) |
| Profile | first name, birth date (18+ enforced server-side, immutable), gender, bio, city, up to 10 interests, up to 6 photos |
| Photos | JPEG/PNG upload (≤ 8 MB), content sniffing, decode + re-encode to JPEG ≤ 1280 px (strips EXIF/payloads, fixes orientation), order, primary photo, replace, delete, **signed expiring URLs** |
| Discovery | mutual-preference deck (gender, age range, distance), never shows swiped, blocked, hidden or incomplete profiles; people who already liked you surface first; swipe or tap |
| Matching | like / pass, idempotent, pass→like upgrade, atomic match creation (advisory lock, no duplicates even under concurrency), unmatch |
| Chat | per-match conversation, real-time delivery over WebSocket, unread counters, read receipts, history pagination, local delete (hide) |
| Notifications | in-app `new_like` / `new_match`, unread count, real-time push to the open app |
| Safety | block (dissolves match, mutual invisibility), report (7 reasons, de-duplicated), privacy toggles (discoverable, show distance) |
| Location | opt-in; coordinates rounded to ~1 km before storage; distance shown only as a coarse figure (1, 2 … 10, 15 … 50, 60 …) |
| Abuse protection | per-IP / per-user rate limits on auth, swipes, messages, uploads, reports; strict input validation; security headers; CORS allow-list |

## Stack

* **API**: Go 1.24, Gin, PostgreSQL 16 via `pgxpool`, explicit SQL (no ORM), `golang-jwt` (HS256 only), `gorilla/websocket`, `x/image` (resizing)
* **Mobile/web**: Expo SDK 54, React Native 0.81, TypeScript (strict), React Navigation, TanStack Query, no UI framework (own design system with light/dark themes)
* **Infra**: Docker, Docker Compose, GitHub Actions

## Repository layout

```text
apps/mobile/            Expo app (src/api, src/features/*, src/ui, src/theme, src/realtime, src/i18n)
services/api/
  cmd/api/              HTTP server (composition root) + integration tests
  cmd/seed/             demo data (development only)
  internal/platform/    config, db (+ tracked migrations), middleware, httpx, realtime hub, mail, signed URLs, logger
  internal/features/    auth, users (profile/preferences/privacy), photos, discovery (deck/swipes/matches),
                        chat, notifications, safety (blocks/reports), realtime (WebSocket)
docs/API.md             full HTTP + WebSocket contract
docs/ARCHITECTURE.md    design notes and decisions
scripts/smoke-api.mjs   end-to-end smoke test against a running API
```

Each feature follows `routes.go → handler.go → service.go → repository.go (+ model.go)`: thin handlers, business
rules in services, all SQL in repositories.

## Quick start (Docker)

```bash
cp .env.example .env
# put a real secret in .env:   JWT_SECRET=$(openssl rand -hex 32)
docker compose up --build -d          # Postgres 16 + API on http://localhost:18080 (migrations run on boot)
curl http://localhost:18080/health

# optional: demo accounts (development only, password: demo-password-123)
docker compose run --rm -e SEED_CONFIRM=yes api ./seed
```

Then start the app:

```bash
cd apps/mobile
cp .env.example .env                  # set EXPO_PUBLIC_API_URL
npm ci
npm run web                           # or: npm start  → scan the QR code with Expo Go
```

**Physical phone?** Never use `localhost`: set `EXPO_PUBLIC_API_URL=http://<your-LAN-IP>:18080` and check
`curl http://<LAN-IP>:18080/health` from the phone's network first. Android emulator uses `http://10.0.2.2:18080`
automatically when the variable is unset. For the web build, add its origin to `ALLOWED_ORIGINS`
(e.g. `http://localhost:8081`).

## Local development without Docker

```bash
# 1. PostgreSQL 16 running locally, then:
createdb app && createdb app_test
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/app?sslmode=disable"
export JWT_SECRET="$(openssl rand -hex 32)"
cd services/api && go run ./cmd/api            # migrations are applied automatically at startup

# demo data / cleanup
SEED_CONFIRM=yes go run ./cmd/seed             # 10 labelled demo accounts (@demo.invalid, bio prefixed "[DÉMO]")
SEED_CONFIRM=yes go run ./cmd/seed -purge
```

## Environment variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | yes | – | PostgreSQL connection string |
| `JWT_SECRET` | yes | – | ≥ 32 chars, random; also derives the photo-URL signing key. Placeholder values are rejected |
| `PORT` | no | `8080` | API port |
| `APP_ENV` | no | `development` | `development`, `test` or `production` (production requires SMTP and refuses to seed) |
| `ALLOWED_ORIGINS` | no | – | CSV of browser origins for CORS and WebSocket (web build only) |
| `TRUSTED_PROXIES` | no | – | CSV of reverse-proxy IPs/CIDRs trusted for `X-Forwarded-For` (needed behind a load balancer so rate limits see real client IPs) |
| `APP_LINK_BASE` | no | `ember://` | deep-link prefix in recovery e-mails |
| `SMTP_HOST/PORT/USER/PASSWORD/FROM` | prod | – | password-recovery e-mail. Without `SMTP_HOST` (development) the e-mail, including the reset code, is printed in the API log |
| `DATABASE_URL_TEST` | tests | – | database used by integration tests (**its `public` schema is wiped**) |
| `EXPO_PUBLIC_API_URL` | mobile | `http://localhost:18080` (dev) | API base URL baked into the client. Public by design: no secret ever goes in the app |
| `SEED_CONFIRM=yes` | seed | – | explicit confirmation to run the seed |

## Database & migrations

Migrations are embedded SQL files in `services/api/internal/platform/db/migrations`, applied in filename order at
startup inside transactions and recorded in `schema_migrations` (advisory-locked, safe with several API instances).
`014_dating_reset.sql` drops the legacy SaaS tables of the original boilerplate; `015`–`018` create the dating schema:
`users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`,
`matches`, `blocks`, `reports`, `conversations`, `conversation_participants`, `messages`, `notifications`, with foreign
keys (`ON DELETE CASCADE` from `users`, reports keep an anonymised row), CHECK constraints, unique constraints and
indexes for every hot path (`swipes(to_user_id, action, created_at)`, `messages(conversation_id, created_at, id)`,
`conversation_participants(conversation_id, user_id)`, `notifications(user_id, created_at, id)` …).

## Tests & quality gates

```bash
cd services/api
gofmt -l .                      # must print nothing
go vet ./...
DATABASE_URL_TEST="postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable" go test -race ./...
go build ./cmd/api ./cmd/seed

cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test

# against a running API:
API_URL=http://localhost:18080 node scripts/smoke-api.mjs
```

Integration tests (`services/api/cmd/api`) cover registration → profile → photos → discovery → like → match →
conversation, plus permissions (non-participants get 404), blocking, reporting, notifications, concurrency
(simultaneous mutual likes create exactly one match), password recovery, rate limiting, account deletion and WebSocket
delivery. They skip automatically when no database URL is configured. `apps/mobile/e2e` contains a Playwright flow for the web build
(see `apps/mobile/package.json`) and a Maestro flow for devices.

## Build & deploy

* API image: `docker build -t ember-api services/api` (static binary, non-root user, healthcheck on `/health`).
* Production checklist: strong unique `JWT_SECRET`; `APP_ENV=production` with real SMTP; HTTPS termination in front
  of the API (HSTS is emitted when `X-Forwarded-Proto: https`); set `TRUSTED_PROXIES`; managed PostgreSQL with
  backups; `ALLOWED_ORIGINS` only if you ship the web build.
* Mobile: `eas build` / `npx expo export --platform web` with `EXPO_PUBLIC_API_URL` pointing at the HTTPS API.
  Android release builds need an HTTPS URL (cleartext HTTP is blocked).

## Key decisions

* **Kept**: Go + Gin + pgx, handler→service→repository layering, refresh-token sessions, Expo + React Navigation + TanStack Query.
* **Removed**: Stripe billing, multi-tenant organizations, posts/comments/files modules (not part of a free dating product), Tamagui (unjustified dependency), PowerShell-only scripts.
* **Photos live in PostgreSQL** (`bytea`, ≤ 6 × ~200 KB per user): one datastore to back up, transactional deletion, no object-storage setup. Delivery uses HMAC-signed URLs stable for a UTC day (cache-friendly, work in `<Image>` without headers). Swap `photos.Repository` for S3 if volume grows.
* **Realtime is a single-instance in-memory hub** behind a `Publisher` interface; chat *sending* is plain REST so nothing is lost if the socket drops, and clients refetch on reconnect. Scale out by implementing `Publisher` over Redis/NATS.
* **WebSocket auth** uses a 30 s single-use ticket (`POST /ws/ticket`) so tokens never appear in URLs.
* **Preferences are mutual and enforced server-side** both when building the deck and when accepting a swipe.
* **Recommendation** is one isolated SQL ordering (`discovery/repository.go`): already-liked-you first, shared interests, recent activity. Replace it to evolve the algorithm.
* **Blocking deletes the match and its messages** (privacy over evidence); reporting before blocking is how a user keeps a record with moderators. Moderation tooling (reviewing `reports`) is intentionally out of scope: reports are stored with status `open|reviewed|dismissed` for an admin tool to consume.
* **Push notifications** (APNs/FCM) are not included: notifications are in-app and real-time while the app is open.
* **Distance**: stored coordinates are rounded to 0.01° (~1 km) and shown coarsely, so exact positions are never revealed or stored.
* Account deletion is a hard delete (GDPR-friendly); the ≥ 18 rule is checked on every profile save and the birth date can't be edited afterwards.

Full API: [docs/API.md](docs/API.md) · Architecture notes: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
