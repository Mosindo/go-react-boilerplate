# Amora

Amora is a free, privacy-minded dating app. Two people create a profile, discover each other, like or pass, and — when the interest is mutual — match and chat in real time.

**It is free, completely.** There is no subscription, no paywall, no paid boost, no artificial like limit and no ad. Rate limits exist only to stop bots and abuse.

## What is in the box

| Area | What you get |
| --- | --- |
| Accounts | Email + password sign-up (18+), sign-in, refresh-token rotation, password recovery by emailed code, account deletion that erases everything |
| Profiles | First name, age (from birth date, never shown as a date), gender, bio, city, up to 10 interests, discovery preferences, privacy switches |
| Photos | Up to 6, reorder / main photo / replace / delete, JPEG + PNG validated by content, re-encoded and resized server-side, private (token-protected) |
| Discovery | Card stack with swipe *or* buttons, tap through photos, full profile view, "that's everyone for now" state |
| Matching | Mutual like ⇒ one match. Duplicates, repeated likes and re-appearing profiles are impossible; simultaneous likes are race-safe |
| Chat | Match-bound conversations, real-time delivery over WebSocket, read receipts, unread counters, local delete, unmatch |
| Location | Approximate by design: coordinates are rounded to ~1 km before storage, others only see a coarse distance, and you can hide it |
| Safety | Block (hides both sides everywhere), report with reasons, hidden-profile mode, server-side authorization on every route |
| Activity | In-app notifications for matches and missed messages, plus live toasts |

## Stack

- **API** — Go, Gin, PostgreSQL via `pgxpool`, explicit SQL (no ORM), JWT HS256 access tokens + hashed rotating refresh tokens, `gorilla/websocket`, `golang.org/x/image` for resizing
- **Mobile** — React Native (Expo SDK 54), TypeScript, React Navigation, TanStack Query, `expo-image`, `expo-image-picker`, `expo-location`. The same code also builds for the web, which is what the browser E2E test drives
- **Infra** — Docker, Docker Compose, GitHub Actions

## Repository layout

```text
.
├─ apps/mobile/                 Expo app
│  ├─ App.tsx
│  ├─ src/api/                  typed API client (the only place that calls fetch)
│  ├─ src/components/           profile form, photo manager, swipe card, safety menu…
│  ├─ src/hooks/                auth session, realtime (WebSocket), discovery queue
│  ├─ src/lib/                  pure, unit-tested logic (dates, swipe, messages…)
│  ├─ src/navigation/ src/screens/ src/shared/   navigation, screens, design system
│  ├─ scripts/                  HTTP end-to-end smoke test + Maestro fixtures
│  └─ e2e/                      browser (Playwright) and Maestro flows
├─ services/api/
│  ├─ cmd/api/                  HTTP server (composition root) + integration tests
│  ├─ cmd/seed/                 demo data (development only)
│  └─ internal/
│     ├─ platform/              config, db + migrations, middleware, logger, realtime hub, mail
│     └─ features/              auth, profiles, photos, matching, conversations, notifications, safety
├─ docker-compose.yml           Postgres + API (+ uploads volume)
└─ docs/ARCHITECTURE.md
```

Every backend feature keeps the `handler → service → repository` layering (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`). SQL lives only in repositories.

## Quick start (Docker)

```bash
cp .env.example .env
# edit .env: set JWT_SECRET (openssl rand -base64 48)
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# optional: 12 clearly fake demo members around Paris
docker compose exec api ./seed
```

Migrations run automatically when the API starts. Demo accounts are `<firstname>@demo.invalid` (for example `louis@demo.invalid`) with password `DemoPassword1!`; `docker compose exec api ./seed -purge` removes them.

### Run the mobile app

```bash
cd apps/mobile
npm ci
cp .env.example .env     # set EXPO_PUBLIC_API_URL to http://<your LAN IP>:18080
npm run start:phone      # detects your LAN IP and starts Expo
```

Never use `localhost` on a physical phone. Before calling a setup ready, check that nothing else is bound to the API port (`docker compose ps`, `curl http://<LAN_IP>:18080/health`). If it is, change the host port (for example `19080:8080`) in `docker-compose.yml`, `apps/mobile/src/api/client.ts`, `apps/mobile/scripts/start-phone.js` and this file.

Android emulators reach the host at `10.0.2.2`; iOS simulators can use `localhost`.

## Local development without Docker

```bash
# 1. PostgreSQL 15+ with an empty database, e.g.
createdb app
# 2. API
cd services/api
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/app?sslmode=disable"
export JWT_SECRET="$(openssl rand -base64 48)"
export UPLOADS_DIR="./data/uploads"
go run ./cmd/api            # applies migrations, listens on :8080
go run ./cmd/seed           # optional demo data
```

## Environment variables

Documented in [`.env.example`](.env.example) (API / compose) and [`apps/mobile/.env.example`](apps/mobile/.env.example).

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `JWT_SECRET` | yes | – | HS256 signing key, ≥ 32 characters, placeholders are rejected |
| `DATABASE_URL` | yes | – | PostgreSQL connection string |
| `PORT` | no | `8080` | API listen port |
| `UPLOADS_DIR` | no | `./data/uploads` (`/data/uploads` in Docker) | Where processed photos live — back it up |
| `ALLOWED_ORIGINS` | no | empty | CORS + WebSocket origins for browser clients |
| `TRUSTED_PROXIES` | no | empty | Reverse proxies whose `X-Forwarded-For` is trusted (needed for correct per-IP rate limiting behind a load balancer) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `MAIL_FROM` | no | – | Password-recovery email. With no `SMTP_HOST` the code is **written to the API log** (development only) |
| `APP_ENV` | no | – | `production` makes `seed` refuse to run |
| `EXPO_PUBLIC_API_URL` | mobile | `localhost:18080` in dev | API base URL as seen from the device |
| `DATABASE_URL_TEST` | tests | – | Enables the Go integration tests |

No secret ever ships in the mobile bundle: `EXPO_PUBLIC_*` values are public by design and only contain the API URL.

## Database and migrations

SQL migrations are embedded in the API binary (`services/api/internal/platform/db/migrations`) and applied in order at startup inside transactions, tracked in `schema_migrations`, guarded by an advisory lock so several instances can start together.

Dating tables: `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `chat_messages`, `blocks`, `reports`, `password_resets`, plus `users`, `sessions` and `notifications`. Highlights:

- foreign keys with `ON DELETE CASCADE` so deleting a user removes every trace; reports survive anonymised (`ON DELETE SET NULL`)
- `matches` stores each pair once (`CHECK (user_a < user_b)` + unique) and doubles as the conversation
- `swipes` has a primary key on `(swiper_id, target_id)`: a profile can only be decided once
- check constraints mirror the validation rules (age range, genders, text lengths, coordinates)
- indexes for the hot paths: discovery (`profiles` by gender/age and by location), `swipes (target_id, action)`, `chat_messages (match_id, created_at)`, unread partial index, `notifications (user_id, created_at)`, `photos (user_id, position)`

## API overview

All routes except `/health` and `/auth/*` need `Authorization: Bearer <access token>` (15 min). Refresh tokens rotate on use.

| Method & path | Purpose |
| --- | --- |
| `POST /auth/register` `login` `refresh` `logout` | Session lifecycle |
| `POST /auth/password-reset/request` `confirm` | Emailed one-time code, always answers 204 |
| `GET/DELETE /me` | Current user / delete account (password re-check) |
| `GET/PUT /me/profile`, `PUT /me/location` `preferences` `privacy`, `GET /interests` | Profile management |
| `GET/POST /me/photos`, `PUT /me/photos/order`, `PUT/DELETE /me/photos/:id`, `GET /photos/:id` | Photos (reads are access-checked) |
| `GET /profiles/:userId` | Public profile (no birth date, email or coordinates) |
| `GET /discover?limit=` | Next candidates, already filtered by both sides' preferences |
| `POST /swipes` | `{targetId, action: like\|pass}` → `{matched, matchId?, profile?}` |
| `GET /conversations`, `GET/POST /conversations/:id/messages`, `POST …/read`, `DELETE /conversations/:id`, `DELETE /matches/:id` | Chat, local delete, unmatch |
| `GET /notifications`, `POST /notifications/:id/read`, `POST /notifications/read-all` | Activity |
| `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` | Safety |
| `GET /ws` | WebSocket; first frame `{"type":"auth","token":…}` (tokens never appear in URLs). Events: `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`, `conversations.changed` |

Errors are `{ "error": "message" }` (plus `code` where the client branches, e.g. `profile_incomplete`). Lists are paginated (`limit` is validated and capped).

## Testing

```bash
# Backend: formatting, vet, unit + integration tests (needs PostgreSQL), build
cd services/api
gofmt -l .
go vet ./...
DATABASE_URL_TEST="postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable" go test -race ./...
go build ./cmd/api

# Mobile: types, lint, unit tests
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm test
```

Without `DATABASE_URL_TEST`, the database-backed Go tests are skipped (the CI always sets it).

End-to-end, against a running API:

```bash
cd apps/mobile
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke   # HTTP + WebSocket journey, cleans up after itself

# Browser journey through the real UI (Playwright + the web export)
EXPO_PUBLIC_API_URL=http://localhost:18080 npx expo export --platform web --output-dir /tmp/amora-web
# start the API with ALLOWED_ORIGINS=http://localhost:19006, then:
PLAYWRIGHT_CHROMIUM_PATH=/path/to/chromium WEB_DIST=/tmp/amora-web npm run e2e:web
```

`e2e:web` signs up, completes onboarding (including a real file upload and geolocation), discovers a member, likes, matches, chats with a live reply, then deletes the account. For on-device runs, `npm run e2e:ui:setup && npm run e2e:ui:run` drives `e2e/maestro/critical-flow.yaml` with Maestro (Windows helper; requires Maestro and a device).

`scripts/qa-lite.ps1` bundles the checks used by the QA Lite workflow.

## Build and deployment

```bash
docker build -t amora-api ./services/api     # static binary, non-root, ships ./api and ./seed
```

Production checklist:
- run behind TLS; set `TRUSTED_PROXIES` to your load balancer and `ALLOWED_ORIGINS` if you serve a web client
- generate a strong `JWT_SECRET`, change the Postgres password, never publish port 5432
- mount a persistent volume on `/data/uploads` and configure SMTP
- mobile: `EXPO_PUBLIC_API_URL=https://api.example.com npx eas build` (or `expo prebuild`), see Expo's docs

## Key decisions

- **A match is the conversation.** One table, one authorization rule ("you are one of the two users"); non-participants get 404 so ids cannot be probed.
- **One definition of "eligible".** Discovery and swipe validation share the same SQL predicate (mutual gender interest, both age ranges, distance, blocks, visibility, has a photo), so the client can never swipe on someone discovery would not show. Ranking is a single `ORDER BY` (people who already liked you first, then closest, then most recently active) — the place to plug in a smarter recommender.
- **First decision is final.** A pass cannot become a like later, which prevents flip-flopping and notification spam.
- **Location is approximate on purpose.** Rounding happens before storage; distances are bucketed in responses; "show my distance" can be switched off; without a location a user is simply not distance-filtered.
- **Photos are private files, not public URLs.** They are served through an authenticated endpoint that re-checks blocks/visibility, are re-encoded as JPEG (stripping EXIF/GPS and any appended payload), and are removed from disk on delete.
- **Real time with a safe fallback.** WebSocket pushes keep screens fresh, and every screen also works from plain REST (pull to refresh, refetch on reconnect).
- **Notifications are in-app.** No push provider is bundled; the notification store and `notification.new` event are the integration point for APNs/FCM.
- **No dark mode yet.** The design tokens are centralized (`apps/mobile/src/shared/ui/tokens.ts`) so adding a theme is a contained change.

## Known limits

- The rate limiter and the WebSocket hub are in-memory: they are correct for one API instance. For several instances, enforce limits at the proxy and back the hub with Redis/NATS pub-sub.
- Access tokens are not revoked server-side before they expire (15 min); refresh tokens and sessions are.
- On the web build the session lives in memory (no secure storage there); native apps use the OS keychain/keystore.
- Moderation is report collection only (`reports` table with `status`); there is no admin UI.
- Legacy generic modules from the original boilerplate (`billing`, `posts`, `comments`, `users`, `files`, `chat` packages, Tamagui files, `docs/agents/payments-agent.md`) are no longer wired into the router. They are kept only because deleting files needs the owner's validation; see the CHANGELOG for the removal list.
