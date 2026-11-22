# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre moderne, **entièrement gratuite** : likes illimités, possibilité de voir qui vous a liké, aucun abonnement, aucun boost payant, aucune publicité. Elle couvre le parcours complet : inscription, profil et photos, découverte géolocalisée, likes / passes, matchs, messagerie temps réel, notifications, préférences, confidentialité, blocage, signalement et suppression de compte.

> The codebase (identifiers, comments, API) is in English; the user interface is in French.

## Table of contents

1. [Features](#features)
2. [Stack](#stack)
3. [Architecture](#architecture)
4. [Getting started](#getting-started)
5. [Environment variables](#environment-variables)
6. [Database, migrations and seed](#database-migrations-and-seed)
7. [Tests and quality](#tests-and-quality)
8. [Build and deployment](#build-and-deployment)
9. [API overview](#api-overview)
10. [Key technical decisions](#key-technical-decisions)
11. [Known limitations](#known-limitations)

## Features

| Area | What members get |
|---|---|
| Account | Sign-up (18+ confirmation), sign-in, password recovery by 6-digit e-mail code, password change (signs out other devices), account deletion with full data erasure |
| Onboarding | First name → birthdate (18+, locked once set) → gender → who to meet → photos (min. 1) → approximate location → bio & interests; resumable at the first missing step |
| Profile | Up to 6 photos (add, replace, delete, reorder, main photo), bio, job, city label, up to 10 interests from a curated catalog |
| Discovery | Swipe cards (gesture **and** buttons), photo carousel, full profile, shared interests highlighted, approximate distance, demo-profile badge |
| Matching | Like / pass, mutual like = match, duplicate-proof and race-free, profiles never re-shown, optional "see passed profiles again" |
| Likes | Free "who liked you" list, like back from there |
| Chat | Conversations only after a match, real-time delivery, timestamps, unread counters, read receipts, history pagination, local "clear conversation", unmatch |
| Notifications | In-app list + real-time events (new match, new like), unread badge, mark (all) as read |
| Preferences & privacy | Genders sought, age range, max distance (all applied **mutually**), pause profile, hide distance |
| Safety | Block / unblock, report (auto-blocks; 3 distinct reporters ⇒ profile hidden pending review) |

## Stack

- **API**: Go 1.24, Gin, PostgreSQL 16, `pgx/v5` (pool, no ORM), JWT HS256 access tokens + rotating refresh tokens, bcrypt.
- **Mobile**: Expo SDK 54 / React Native 0.81 (iOS, Android and web), TypeScript (strict), React Navigation 6, TanStack Query 5, `expo-image`, `expo-image-picker`, `expo-image-manipulator`, `expo-location`, `expo-secure-store`.
- **Real time**: Server-Sent Events (`GET /events`) consumed through XHR streaming (works on iOS, Android and web with an `Authorization` header).
- **Infra**: Docker, Docker Compose, GitHub Actions (backend, mobile and end-to-end jobs).
- **Tests**: Go unit + integration tests (real PostgreSQL), Node unit tests, Playwright UI end-to-end on the web build, API smoke script.

## Architecture

```text
.
├─ apps/mobile/                 Expo app (iOS / Android / web)
│  ├─ App.tsx                   providers: safe area, React Query, auth, toasts
│  ├─ e2e/                      Playwright UI tests + API helpers
│  ├─ scripts/                  API smoke test, web server for E2E, LAN launcher
│  └─ src/
│     ├─ api/                   HTTP client (auto refresh), typed endpoints, SSE client, error copy
│     ├─ components/            domain components (SwipeCard, PhotoGrid, ReportSheet…)
│     ├─ hooks/                 auth session, React Query hooks, realtime cache sync
│     ├─ lib/                   validation, formatting, media, location, confirm (unit tested)
│     ├─ navigation/            auth stack → onboarding gate → tabs + stack
│     ├─ screens/               auth, onboarding, discover, likes, chat, profile
│     ├─ store/                 token storage (Keychain/Keystore; memory on web)
│     └─ ui/                    design system (tokens, Text, Button, TextField, states, toast…)
├─ services/api/
│  ├─ cmd/api                   composition root + HTTP integration tests
│  ├─ cmd/seed                  demo data generator (refuses production)
│  └─ internal/
│     ├─ platform/              config, db (+ migrations), middleware, httpx, media, mailer, realtime, logger
│     └─ features/              auth, profiles, photos, matching, chat, safety, notifications, users,
│                               + generic boilerplate modules kept intact: posts, comments, files
├─ docker-compose.yml           postgres + api (uploads volume)
└─ docs/ARCHITECTURE.md         deeper technical notes
```

Each backend feature follows `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`): handlers only bind/validate HTTP input and map errors, services own business rules, repositories own SQL. Visibility rules shared across features (profile completeness, blocks, matches) live once in `features/profiles/sql.go`.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for data model, security model and real-time design.

## Getting started

### Prerequisites

- Node.js 22+ and npm
- Go 1.24+
- Docker (or a local PostgreSQL 14+)

### 1. Configure

```bash
cp .env.example .env
# set a strong secret:
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env
```

### 2. Start PostgreSQL + API with Docker

```bash
docker compose up --build -d
curl http://localhost:18080/health   # {"status":"ok"}
```

Migrations run automatically at API start-up.

### 3. (Optional) Load demo profiles

```bash
docker compose exec api ./seed -count 24
# make half of them like your own account, for instant matches:
docker compose exec api ./seed -count 24 -like-email you@example.com
```

### 4. Start the app

```bash
cd apps/mobile
npm ci
npm run start        # Expo dev server (scan the QR code with Expo Go)
npm run web          # in a browser
```

- iOS simulator / web: the API defaults to `http://localhost:18080`.
- Android emulator: defaults to `http://10.0.2.2:18080`.
- **Physical phone: never use `localhost`.** Use `npm run start:phone` (detects your LAN IP) or set `EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080`.

If port 18080 is taken on your machine, change the host port in `docker-compose.yml` (e.g. `18081:8080`) and `EXPO_PUBLIC_API_URL` accordingly.

### Run the API without Docker

```bash
cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/api
```

In development, password-reset codes are printed in the API log (`"event":"dev_mail"`) unless SMTP is configured.

## Environment variables

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `JWT_SECRET` | yes | — | HS256 signing key (≥ 32 chars, placeholders rejected). Also derives the photo-URL signing key. |
| `DATABASE_URL` | yes | — | PostgreSQL connection string |
| `APP_ENV` | no | `development` | `development`, `test` (relaxed rate limits for automated suites) or `production` (requires SMTP, Gin release mode) |
| `PORT` | no | `8080` | API port (Docker publishes it on `18080`) |
| `ALLOWED_ORIGINS` | no | — | Comma-separated browser origins for CORS (web app / Expo web) |
| `TRUSTED_PROXIES` | no | — | Reverse-proxy IPs/CIDRs whose `X-Forwarded-For` is trusted (rate limiting uses client IP) |
| `UPLOAD_DIR` | no | `./data/uploads` | Photo storage directory (a volume in Docker) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | production | — | Relay for account-recovery e-mails |
| `POSTGRES_PASSWORD` | no | `postgres` | Used by `docker-compose.yml` |
| `EXPO_PUBLIC_API_URL` | mobile | dev fallbacks | API base URL baked into the app. Public by nature: never put secrets in `EXPO_PUBLIC_*`. |

## Database, migrations and seed

- Migrations are plain SQL files in `services/api/internal/platform/db/migrations`, embedded in the binary and applied at start-up **exactly once** (tracked in `schema_migrations`, serialised with a PostgreSQL advisory lock, one transaction per file).
- To add one, create `NNN_description.sql` with the next number.
- Main tables: `users`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (likes **and** passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`, `sessions`, `password_reset_tokens`.
- Seed: `go run ./cmd/seed [-count N] [-lat .. -lng .. -radius-km ..] [-like-email you@x.y]`, purge with `-purge`. Demo members use the reserved domain `demo.lueur.invalid`, have `users.is_demo = TRUE`, abstract gradient pictures (no real person) and a visible "Profil de démo" badge. The seed refuses to run with `APP_ENV=production`.

## Tests and quality

Backend (`services/api`):

```bash
gofmt -l .                     # must print nothing
go vet ./...
DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable' go test ./...
go build ./cmd/api ./cmd/seed
```

Integration tests run against a real PostgreSQL (they are skipped when no database URL is set). They cover auth (rotation, revocation, recovery, deletion), the critical dating path, permissions (outsiders can never read a conversation), privacy (no e-mail/coordinate leaks), concurrency (simultaneous mutual likes ⇒ exactly one match), photos (dangerous uploads, ordering), moderation, distance/age filters and the SSE stream.

Mobile (`apps/mobile`):

```bash
npm run typecheck
npm run lint
npm run format:check
npm test                       # unit tests (validation, formatting)
```

End-to-end (needs an API on `:18080` started with `APP_ENV=test` and `ALLOWED_ORIGINS=http://localhost:8081`):

```bash
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke   # API critical path
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
npx playwright test                                           # UI: sign-up → onboarding → discovery → like → match → real-time chat
```

Windows users can run everything with `scripts/qa-lite.ps1 -ApiBaseUrl http://localhost:18080`.

CI (`.github/workflows/ci.yml`) runs three jobs: **backend** (gofmt, vet, tests with PostgreSQL, build, Docker image health check), **mobile** (typecheck, lint, format, unit tests, web build) and **e2e** (API smoke + Playwright).

## Build and deployment

- **API**: the multi-stage `services/api/Dockerfile` produces a small Alpine image running as a non-root user with `api` and `seed` binaries. Deploy it behind an HTTPS reverse proxy (set `TRUSTED_PROXIES`), with a managed PostgreSQL, a persistent volume for `UPLOAD_DIR` (or an object-storage implementation of `media.Storage`), `APP_ENV=production` and SMTP configured. Disable proxy buffering on `/events` (the API already sends `X-Accel-Buffering: no`).
- **Mobile**: build store binaries with EAS (`npx eas build -p ios|android`) with `EXPO_PUBLIC_API_URL` pointing to your HTTPS API. Web: `npm run build:web` outputs a static site in `web-build/`.

## API overview

All endpoints except auth, `/health` and signed media require `Authorization: Bearer <access token>`. Errors are `{"error": "...", "code": "..."}`.

| Method & path | Purpose |
|---|---|
| `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Session lifecycle (refresh tokens rotate) |
| `POST /auth/password/forgot`, `/auth/password/reset` | Account recovery by e-mail code |
| `GET /me`, `POST /me/password`, `DELETE /me` | Account, password change, deletion |
| `GET/PATCH /me/profile`, `PUT /me/location`, `PUT /me/preferences`, `GET /interests` | Profile, approximate location, discovery preferences, catalog |
| `GET/POST /me/photos`, `PUT /me/photos/:id`, `DELETE /me/photos/:id`, `PUT /me/photos/order` | Photo management (multipart field `photo`) |
| `GET /media/photos/:id?exp&sig` | Signed, expiring photo URL |
| `GET /discovery`, `DELETE /discovery/passes`, `POST /swipes` | Ranked feed, reset passes, like / pass |
| `GET /likes/received`, `GET /matches`, `DELETE /matches/:userId` | Likes, matches, unmatch |
| `GET /profiles/:userId` | Public card (only if allowed) |
| `GET /chats`, `GET/POST /chats/:userId/messages`, `POST /chats/:userId/read`, `DELETE /chats/:userId` | Conversations |
| `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` | Safety |
| `GET /notifications`, `POST /notifications/:id/read`, `POST /notifications/read-all` | Notifications |
| `GET /events` | Server-Sent Events stream |
| `GET /health` | Liveness + database check |

## Key technical decisions

- **Built on the existing boilerplate** (Go/Gin/pgx + Expo) rather than rewriting it; generic modules (`posts`, `comments`, `files`) are kept untouched and tested but unused by the app.
- **Billing removed**: the product is free, so the Stripe module, its config and UI were deleted (migration `014` drops the table).
- **Privacy first**: coordinates are rounded to ~1 km before storage and never returned (not even to the owner); other members see a rounded distance only, which they can hide. E-mails are only returned to their owner. `/users` lists matched contacts only (no directory).
- **Photos**: content sniffing (never trusting file names), decompression-bomb limits, server-side re-encoding to JPEG that strips EXIF/GPS metadata, random storage keys, viewer-scoped signed URLs with bucketed expiry (cache friendly).
- **Matching**: one `swipes` row per (swiper, target) instead of separate likes/passes tables; a per-pair advisory lock guarantees a single match under concurrency; every discovery criterion is mutual; ranking is a pluggable `Ranker` (proximity, activity, shared interests, reciprocal interest) so a smarter engine can replace it.
- **Chat**: reuses the boilerplate's `conversations` tables; a conversation is created at match time and reachable only through an existing match, so unmatch/block revoke access instantly for both. Outsiders get 404 (no existence oracle).
- **Real time via SSE** instead of WebSockets: no new dependency, plain HTTP, simple auth; the in-process hub sits behind a `Publisher` interface (swap for PostgreSQL `LISTEN/NOTIFY` or a broker to scale horizontally).
- **Sessions**: the auth middleware checks the session row on each request, so logout, password changes and deletions take effect immediately.
- **Abuse protection**: token-bucket rate limits on auth, recovery, swipes, messages, reports and uploads — anti-abuse ceilings, not product quotas (likes stay unlimited).
- **Age**: 18+ enforced server-side, birthdate locked once set.
- **Dark-only UI** with an original warm palette ("lueur" = glow); every swipe gesture has a button alternative and controls carry accessibility labels.

## Known limitations

- Native push notifications are not wired (they need an EAS project and push credentials); notifications are in-app and real time.
- The real-time hub and rate limiter are per-process; run one API instance or add a shared backend before scaling out.
- There is no moderation back-office yet: reports are stored in the `reports` table and profiles past the threshold are hidden (`moderation_status = 'under_review'`) until handled.
- On web, sessions are kept in memory only (by design, no token in `localStorage`): reloading the page signs out.
