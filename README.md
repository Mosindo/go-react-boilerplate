# Lueur — a free dating app, end to end

Lueur is a complete, free dating app: create an account, build a profile with photos, discover people nearby,
like or pass, match when the interest is mutual, and chat in real time. It is a monorepo with a Go API, a
PostgreSQL database and a React Native (Expo) client that also runs in the browser.

**Free means free.** There is no subscription, no paywall, no paid boost, no artificial like limit and no
mandatory ads. Rate limits exist only to stop abuse (hundreds of likes per hour are fine).

> The repository started as a generic "go-react-saas" boilerplate. Its authentication, session, logging and
> Docker foundations were kept; the product modules were replaced by the dating domain (see
> [Legacy code](#legacy-code-pending-removal)).

## Features

| Area | What you get |
| --- | --- |
| Accounts | Email + password sign-up and login, refresh-token rotation, password recovery by email, change password, **account deletion** that erases the profile, photos, matches and messages |
| Onboarding | Name, birth date (18+ enforced by the server), gender, photos, bio, interests, who you want to meet, age range, distance, location; resumes where you stopped |
| Discovery | Swipe deck (drag or buttons), photo carousel, full profile view, mutual filtering on gender, age and distance, "who liked you" first, then shared interests and proximity |
| Matching | Like / pass, reciprocal like = match (atomic, race-free), no duplicates, processed profiles never come back, unmatch |
| Chat | Only between matches, real-time (WebSocket), sent/read receipts, unread counters, history pagination, local "clear conversation" |
| Notifications | Match and message notifications (in-app + live), mark read / read all |
| Photos | Up to 6, content-sniffed JPEG/PNG only, re-encoded and downsized, EXIF/GPS stripped, reorder, primary photo, replace, delete, private signed URLs |
| Location | Coarse (≈1 km) storage, distances shown in 5 km steps, never exact coordinates, optional hiding |
| Safety | Block, report (with optional block), pause profile, hide distance, blocked list |
| UX | Original "Lueur" identity, light/dark, loading / empty / error states, toasts, accessible controls, reduced-motion aware |

## Tech stack

- **API** — Go 1.24, Gin, pgx/pgxpool (no ORM, explicit SQL), JWT HS256 + rotating refresh tokens, bcrypt, gorilla/websocket
- **Database** — PostgreSQL 16, tracked SQL migrations embedded in the binary
- **Client** — Expo 54 / React Native 0.81 / React 19, TypeScript (strict), React Navigation, TanStack Query, expo-image, expo-image-picker, expo-location
- **Tests** — Go unit + database integration tests, Vitest, Playwright (browser E2E), Maestro flow for native devices
- **Infra** — Docker, Docker Compose, GitHub Actions

## Architecture

```text
.
├─ services/api/                 Go API
│  ├─ cmd/api                    HTTP server (main.go, app.go = dependency wiring)
│  ├─ cmd/seed                   demo data tool (never runs in production)
│  └─ internal/
│     ├─ platform/               config, db (+migrations), middleware, httpx, realtime hub,
│     │                          mailer, signing, logger, errors, health
│     └─ features/               auth · profiles · photos · matching · chat · notifications · safety
│                                (each: handler → service → repository, model, routes)
├─ apps/mobile/                  Expo app (iOS, Android, web)
│  ├─ src/api                    typed REST client + endpoint facade (UI never calls fetch directly)
│  ├─ src/realtime               WebSocket client, provider, query-cache updates
│  ├─ src/hooks, screens, components, shared/ui, theme, navigation, lib
│  ├─ e2e/web                    Playwright suite   e2e/maestro  native flow
│  └─ scripts                    API smoke, static server, fixtures for Maestro
├─ docker-compose.yml            PostgreSQL + API for local use
└─ docs/ARCHITECTURE.md          data model, API contract, decisions
```

Rules kept from the original boilerplate: one HTTP server (Gin), thin handlers, business logic in services,
SQL only in repositories, one shared connection pool, shared concerns in `platform`.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the data model, the matching and discovery design, the
realtime protocol and the security model.

## Quick start

Prerequisites: Docker, Node 20+ (22 recommended) and Go 1.24+ (only if you run the API without Docker).

```bash
# 1. Configure
cp .env.example .env
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(openssl rand -hex 32)/" .env

# 2. Start PostgreSQL + API (migrations run automatically at boot)
docker compose up --build
curl http://localhost:18080/health            # → {"status":"ok"}

# 3. (optional) demo profiles to swipe on — development only
docker compose run --rm api ./seed            # accounts: demo.camille@demo.invalid / demo-password-123

# 4. Start the app
cd apps/mobile && npm ci
npm run web                                   # browser: http://localhost:8081
# or on a phone (same Wi-Fi): npm run start:phone   → scan the QR code with Expo Go
```

The API is published on host port **18080** (the container listens on 8080).

### Running the API without Docker

```bash
createdb app                                   # any PostgreSQL 14+ database
cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export PORT=18080 ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api
go run ./cmd/seed                              # optional demo data;  go run ./cmd/seed --purge  removes it
```

### Phone checklist

- Never use `localhost` on a physical phone. `npm run start:phone` detects your LAN IP and exports
  `EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080`.
- Verify nothing else listens on the API port: `docker compose ps`, then `curl http://<LAN_IP>:18080/health`.
  If another service hijacks 18080, change the host port in `docker-compose.yml` and re-run `start:phone`.
- A change is "ready" only after the smoke test: `node apps/mobile/scripts/e2e-smoke.js` (API loop) and the
  manual run: register → onboarding → discover → like → match → chat → notifications.

## Environment variables

Documented in [`.env.example`](.env.example). Nothing secret is ever shipped to the client.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `JWT_SECRET` | yes | — | HMAC secret, ≥ 32 chars, placeholders rejected |
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `PORT` | no | `8080` | API listen port |
| `APP_ENV` | no | `development` | `production` enforces SMTP and never logs recovery links |
| `UPLOAD_DIR` | no | `./data/uploads` | where processed photos are stored (mount a volume) |
| `ALLOWED_ORIGINS` | web only | — | CORS allow-list (comma separated) |
| `TRUSTED_PROXIES` | behind a proxy | — | proxies whose `X-Forwarded-For` is trusted |
| `PASSWORD_RESET_URL` | no | localhost link | link template containing `{token}` |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | production | — | recovery emails |
| `EXPO_PUBLIC_API_URL` | release builds / phones | auto in dev | API URL as seen by the device |
| `SEED_PASSWORD`, `SEED_LAT`, `SEED_LNG` | no | demo defaults | seed tool |
| `DATABASE_URL_TEST` | tests | — | database used (and **reset**) by the Go integration tests |

## Database and migrations

Migrations are plain SQL files in `services/api/internal/platform/db/migrations/`, embedded in the binary and
applied at startup in filename order, each in its own transaction, once (tracked in `schema_migrations`,
protected by an advisory lock so several replicas can boot together). To change the schema add a new numbered
file; never edit an applied one.

Main tables: `users`, `sessions`, `password_reset_tokens`, `profiles`, `preferences`, `interests`,
`user_interests`, `photos`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`,
`blocks`, `reports`, `notifications`. Constraints, foreign keys with cascading deletes, unique pairs
(`matches.user_a < user_b`, one swipe per pair) and the indexes needed by the hot paths are in the migrations.

## Tests and quality gates

```bash
# Backend (needs a PostgreSQL database that the tests may drop and recreate!)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'
gofmt -l . && go vet ./... && go test -race ./... && go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci && npx tsc --noEmit && npm run lint && npm run format:check && npm test

# Browser E2E: signup → onboarding → discovery → like → match → chat → block (two real browsers)
#   terminal 1: API with ALLOWED_ORIGINS=http://localhost:8081 on an EMPTY database
#   terminal 2: EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web && npm run serve:web
npx playwright install chromium
npm run e2e:web

# API-level smoke of the whole loop (any running API)
node scripts/e2e-smoke.js
```

Native devices: `npm run e2e:ui:setup` prepares fixtures and `e2e/maestro/critical-flow.yaml` replays the flow
with [Maestro](https://maestro.mobile.dev) (`npm run e2e:ui:run` on Windows). CI (`.github/workflows/ci.yml`) runs
backend, mobile, browser E2E and Docker jobs.

## Build and deploy

- **API image**: `docker build -t lueur-api services/api` → static binary, non-root user, health check, photos in `/data/uploads`.
- Run it behind HTTPS (terminate TLS at a reverse proxy, set `TRUSTED_PROXIES`), with `APP_ENV=production`,
  SMTP configured, a strong `JWT_SECRET`, a managed PostgreSQL and a persistent volume for `/data/uploads`.
- **Mobile**: set `EXPO_PUBLIC_API_URL` to the public API URL, then `eas build` (or `expo prebuild`) for the stores,
  and `npm run build:web` for a static web build (`dist/`). Update `ios.bundleIdentifier` / `android.package`
  in `app.json`.
- Scaling out: the realtime hub and rate limiters are per-instance. Before running several API replicas, put
  a shared rate limiter in front and fan realtime events out with PostgreSQL `LISTEN/NOTIFY` or Redis behind
  `realtime.Publisher`; for photos switch `photos.Storage` to object storage.

## Key product and technical decisions

- **Chat only after a match**; a conversation is created in the same transaction as the match.
- **Server-side rules for everything**: age ≥ 18 (birth date immutable once set), mutual preferences are
  re-checked on every like (a crafted request cannot like someone outside your criteria), blocks are enforced
  in every query, conversation membership is checked on every chat operation (non-members get `404`).
- **Passes are final**: a profile you passed or liked never reappears; a repeated swipe returns `409`.
- **Ranking** is one `ORDER BY` in `matching/repository.go` (admirers first, shared interests, distance,
  recent activity) — the single place to evolve recommendations.
- **Photos are private**: the API hands out signed URLs valid 1–2 hours (HMAC, bound to the photo id), so image
  widgets work without headers. They are re-encoded JPEGs, so scripts, polyglots and metadata cannot survive.
- **Location**: coordinates rounded to ≈1 km at write time, only ever used server-side; others see 5 km steps.
- **Realtime**: WebSocket authenticated by a first frame (no token in URLs), closed when the access token
  expires, client reconnects with backoff, refreshes the token and refetches what it missed.
- **Reports outlive deleted accounts** (user ids set to `NULL`) so moderators keep their history. There is no
  admin UI: review `reports` with SQL (`status`: open / reviewed / dismissed / actioned).
- **No push notifications** yet (they need Apple/Google credentials); notifications are in-app and live.
- **French UI** with strings in the screens; the API returns stable `code`s so clients can localise.

## Security summary

JWT HS256 with strict algorithm, mandatory expiry and session check on every request (logout, password reset and
account deletion take effect immediately) · bcrypt with 72-byte limit and timing-equalised login · one-time
hashed recovery tokens, answers never reveal whether an email exists · single-use rotating refresh tokens ·
per-IP and per-user rate limits (client IP not spoofable unless `TRUSTED_PROXIES` is set) · strict input validation
and normalisation (email, names, text, pagination, ids) · upload hardening (content sniffing, pixel limits,
re-encoding, size caps, private storage) · security headers, CORS allow-list · secrets only from the environment ·
sensitive values redacted from logs.

## API overview

All routes except `/health`, `/auth/*` and signed `/photos/:id/image` require `Authorization: Bearer <token>`.
Errors are `{ "error": "...", "code": "snake_case" }`.

| | |
| --- | --- |
| Auth | `POST /auth/register · /auth/login · /auth/refresh · /auth/logout · /auth/password/forgot · /auth/password/reset · /auth/password/change`, `GET /me`, `DELETE /me` |
| Profile | `GET/PUT /profile/me`, `PUT /profile/preferences · /profile/location · /profile/privacy`, `GET /interests`, `GET /profiles/:userId` |
| Photos | `GET/POST /photos`, `PUT /photos/order`, `PUT/DELETE /photos/:id`, `GET /photos/:id/image?exp&sig` |
| Matching | `GET /discover?limit`, `POST /swipes`, `GET /matches`, `DELETE /matches/:id` |
| Chat | `GET /conversations`, `GET/POST /conversations/:id/messages`, `POST /conversations/:id/read`, `DELETE /conversations/:id` |
| Notifications | `GET /notifications`, `POST /notifications/:id/read`, `POST /notifications/read-all` |
| Safety | `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` |
| Realtime | `GET /ws` (WebSocket; first frame `{"type":"auth","token":"…"}`) |

## Legacy code pending removal

These modules come from the original SaaS boilerplate. They are **not wired** into the router, but their files
are still in the tree because the automated session that built Lueur was not allowed to delete files in bulk.
Remove them (and nothing else depends on them):

```bash
git rm -r services/api/internal/features/{billing,posts,comments,files,users} apps/mobile/tamagui.config.ts
```

Their old tables (`posts`, `comments`, `votes`, `files`, `subscriptions`, `organizations`) are created by the
early migrations and stay unused; a future migration can drop them. Billing code must never be reconnected:
the product is free by design.

## License and conduct

See `CODE_OF_CONDUCT.md` and `CONTRIBUTING.md`.
