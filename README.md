# Ember — a free dating app

Ember is a 100 % free dating app: no subscription, no paywall, no paid boosts, no artificial
like limit, no mandatory ads. Two people create a profile with photos, discover each other,
like or pass, match when the interest is mutual, and chat in real time.

The repository started as the `go-react-saas` boilerplate and keeps its architecture
(Go + Gin + PostgreSQL + Expo), see [Decisions](#key-decisions).

## Features

- Accounts: register, login, refresh-token rotation, logout, password reset by e-mailed code, account deletion (removes all personal data and photo files)
- Profile: first name, age (18+ enforced server-side), gender, bio, city, interests, up to 6 photos (upload, replace, reorder, main photo, delete)
- Preferences & privacy: who I want to meet, age range, max distance, pause my profile, hide my distance
- Discovery: ranked deck with swipe gestures *and* buttons, shared interests, approximate distance, full profile view
- Matching: like/pass, mutual like = match (race-safe, no duplicates, processed profiles never come back), unmatch
- Chat: one conversation per match, real-time over WebSocket, sent time, read receipts and unread counts, per-user hide, history pagination
- Notifications: matches and new messages (in-app + live)
- Safety: block, report (with a snapshot of the conversation for moderators), blocked list, unblock

## Stack

| Layer | Tech |
|---|---|
| API | Go 1.24, Gin, pgx/pgxpool, JWT HS256, bcrypt, gorilla/websocket, x/image (no ORM) |
| DB | PostgreSQL 16, embedded SQL migrations tracked in `schema_migrations` |
| Mobile | Expo SDK 54 / React Native, TypeScript, React Navigation 7, TanStack Query |
| Infra | Docker, Docker Compose, GitHub Actions |

## Repository layout

```
apps/mobile/           Expo app (src/api, screens, navigation, realtime, theme, shared UI kit)
services/api/
  cmd/api/             HTTP server entry point (+ end-to-end integration tests)
  cmd/seed/            development-only demo data
  internal/platform/   config, db+migrations, middleware (auth, rate limit), realtime hub, media signing, mailer, errors
  internal/features/   auth, profiles, photos, discovery (swipes+matches), chat, notifications, safety
docs/API.md            the HTTP/WebSocket contract
```

Each feature follows `handler -> service -> repository`; handlers stay thin and no SQL lives outside repositories.

## Getting started

Prerequisites: Go 1.24+, Node 20+, Docker (or a local PostgreSQL 15+).

```bash
cp .env.example .env            # then set JWT_SECRET (openssl rand -hex 32)
export JWT_SECRET=$(grep ^JWT_SECRET= .env | cut -d= -f2)
docker compose up --build -d    # PostgreSQL + API on http://localhost:18080
curl http://localhost:18080/health
```

Migrations run automatically at API start-up (applied once each, serialised by an advisory lock).

### Without Docker

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET=$(openssl rand -hex 32)
cd services/api && go run ./cmd/api        # PORT defaults to 8080
```

### Demo data (development only)

```bash
export SEED_PASSWORD='choose-a-demo-password'
cd services/api && go run ./cmd/seed        # 20 accounts *@demo.invalid with generated placeholder pictures
# or inside Docker:  docker compose exec -e SEED_PASSWORD=... api ./seed
```

Demo accounts use the reserved `demo.invalid` domain and the command refuses to run with `APP_ENV=production`.

### Mobile app

```bash
cd apps/mobile
npm ci
# Never use localhost on a physical phone: use your machine's LAN IP.
EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080 npx expo start
```

Android emulator falls back to `http://10.0.2.2:18080`, iOS simulator to `http://localhost:18080`.
Before calling a build ready, check no other process owns the API port (`docker compose ps`, `curl http://<LAN_IP>:18080/health`).
In development the password-reset code is printed in the API log (`docker compose logs api`).

## Environment variables

See [`.env.example`](.env.example) for the full annotated list.

| Variable | Required | Purpose |
|---|---|---|
| `JWT_SECRET` | yes | ≥ 32 random chars, signs access tokens and photo URLs |
| `DATABASE_URL` | yes | PostgreSQL connection string |
| `PORT` | no (8080) | API port |
| `APP_ENV` | no | `production` requires SMTP and blocks seeding |
| `UPLOAD_DIR` | no | photo storage directory (persistent volume in production) |
| `ALLOWED_ORIGINS` | no | CORS / WebSocket origins for web clients |
| `TRUSTED_PROXIES` | no | proxies whose `X-Forwarded-For` is trusted |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | production | password-reset e-mails |
| `SEED_PASSWORD`, `SEED_LAT`, `SEED_LNG` | seed only | demo accounts |
| `EXPO_PUBLIC_API_URL` | mobile | API base URL (no secrets ever go in `EXPO_PUBLIC_*`) |

## Quality checks

```bash
# API (needs PostgreSQL: integration tests skip when DATABASE_URL is unset)
cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
gofmt -l .            # must print nothing
go vet ./... && go test ./... && go build ./cmd/api

# Mobile
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test

# Black-box journey against a running API (register → profile → photo → match → chat → delete)
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
```

`services/api/cmd/api/main_integration_test.go` covers auth, password reset, profile validation, photos, discovery filters, matching (including simultaneous likes), chat permissions, WebSocket delivery, blocking, reporting and account deletion against a real database.

`apps/mobile/e2e/maestro/critical-flow.yaml` describes the same journey for Maestro on a device; it is not executed in CI.

## Deployment

1. Build the image: `docker build -t ember-api services/api` (contains `api` and `seed`).
2. Provide `DATABASE_URL`, a strong `JWT_SECRET`, `APP_ENV=production`, SMTP settings and a persistent volume at `UPLOAD_DIR`.
3. Put it behind a TLS reverse proxy that forwards WebSocket upgrades on `/ws`, and set `TRUSTED_PROXIES` to the proxy address.
4. Build the mobile app with `EXPO_PUBLIC_API_URL=https://api.your-domain` (EAS Build / store release).

## Key decisions

- **Free by construction.** The Stripe billing module and the SaaS organisation/subscription tables were removed; nothing in the API or UI is gated.
- **Migrations are append-only and tracked.** Migration 014 retires the generic posts/comments/billing schema; dating tables follow in 015–020.
- **Matching is transactional.** A per-pair advisory lock makes two simultaneous likes yield exactly one match and one conversation; `(user_a < user_b)` plus a unique constraint rules out duplicates.
- **Ranking is isolated** in `discovery/ranking.go` (shared interests, "likes you", recent activity, proximity, small random term) so the algorithm can evolve without touching the query.
- **Location privacy.** Coordinates are rounded to ~1 km before storage, never returned by any endpoint; others only see a coarse distance (1 km steps under 10 km, then 5 km) and users can hide it.
- **Photos are private capability URLs.** Uploads are type-sniffed, size/pixel-bounded, re-encoded to JPEG (metadata stripped, max 1440 px). URLs are HMAC-signed and expire; they are only handed to the owner, matches and eligible candidates.
- **Real-time over WebSocket** with single-use tickets (no long-lived token in the URL). The hub is in-process; scale out by putting pub/sub behind `Hub.Publish`. Messages are sent through REST.
- **Sessions are checked on every request** (one indexed lookup) so logout, password reset and account deletion take effect immediately.
- **Blocking** removes the match and conversation and hides both people from each other everywhere. **Reports** keep a snapshot of the last 20 shared messages and survive account deletion with the user references nulled; there is no automatic sanction (avoids brigading) — review the `reports` table.
- **Rate limits** are in-memory per process (auth, uploads, reports, messages, swipes with a generous cap meant only to stop bots).
- **No push notifications yet** (needs APNs/FCM credentials); notifications are in-app and live over the WebSocket.
- **Text only chat**, no photo messages, to keep the moderation surface small.

## Known limitations

- The Maestro UI flow and the swipe gestures have not been run on a physical device or emulator in this repository's CI.
- Chat/notification fan-out and rate limiting are single-instance until a shared store is added.
- Image EXIF orientation is not applied server-side; the app captures with `expo-image-picker`, which normalises orientation.
- Legacy modules from the original boilerplate (`billing`, `posts`, `comments`, `files`, `users`) remain in `services/api/internal/features` but are not wired into the router; see CHANGELOG.
