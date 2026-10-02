# Alba — free dating app

Alba is a complete, **100 % free** dating app: no subscription, no paywall, no paid boost, no artificial like limit, no mandatory ads. Two people can sign up, build a profile with photos, discover each other, like or pass, match when the interest is mutual, chat in real time, get notified, tune their preferences and privacy, block or report someone, and delete their account.

## Features

| Area | What is implemented |
| --- | --- |
| Accounts | Register, login, refresh-token rotation, logout, password recovery by e-mailed code, password change, account deletion (erases all data) |
| Profile | First name, age (18+, birth date locked after sign-up), gender, bio, city, up to 10 interests, discoverable / show-distance switches |
| Photos | Up to 6, upload, replace, delete, reorder, primary photo; content sniffed, decoded, resized (max 1280 px) and re-encoded to JPEG (strips EXIF/GPS and payloads); served only via signed expiring URLs |
| Discovery | Swipe deck (drag, or buttons), tap photos, full profile; server-side filters: orientation both ways, age range both ways, distance both ways, hidden profiles, blocks, already-processed profiles |
| Matching | Like / pass, unique swipe per pair, match when both like, per-pair locking (no duplicate or missed match under concurrency) |
| Chat | Text messages, WebSocket push, timestamps, read receipts, unread counters, per-user history clearing, unmatch |
| Notifications | In-app list + tab badges (new match, new message — coalesced per conversation) |
| Safety | Block (removes the match, hides both sides), report with reason, rate limiting, strict validation, ownership checks on every resource |
| Location | Only coarse coordinates stored (rounded to ~1 km), never returned; other users see a distance rounded up to 5 km |

## Stack

- **API**: Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256 (15 min) + rotating refresh tokens, bcrypt, gorilla/websocket, `golang.org/x/image` (WebP decode + resize). No ORM, explicit SQL.
- **Mobile / web**: Expo SDK 54, React Native 0.81, TypeScript (strict), React Navigation, TanStack Query. Runs on iOS, Android and web (`react-native-web`).
- **Infra**: Docker, Docker Compose, GitHub Actions.

## Repository layout

```text
apps/mobile/            Expo app (src/api, auth, realtime, navigation, screens, ui, theme, lib)
  e2e/                  Playwright browser journey against the web build + real API
services/api/
  cmd/api               API entrypoint            cmd/seed   demo data (never in production)
  internal/app          wiring of platform + features into one Gin engine
  internal/platform/    config, db (+ migrations), middleware, httpx, ratelimit, realtime hub, storage, mailer, token, logger
  internal/features/    auth, profiles, photos, discovery, matches, chat, notifications, safety, realtime
infra/docker-compose.yml
docs/ARCHITECTURE.md
```

Each feature follows `handler → service → repository` (+ `model.go`, `routes.go`). Handlers never contain SQL.

## Quick start (Docker)

```bash
cp .env.example .env
# put a random value in JWT_SECRET:  openssl rand -hex 32
docker compose --env-file .env -f infra/docker-compose.yml up --build -d
curl http://localhost:18080/health            # {"status":"ok"}

# optional: 12 clearly marked demo accounts (password DemoPass123)
docker compose --env-file .env -f infra/docker-compose.yml run --rm api ./seed
```

The API listens on **18080** (host) → 8080 (container). Migrations run automatically at startup.

## Run without Docker

```bash
# 1. Postgres (any 14+)
createdb app
# 2. API
cd services/api
export DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export PORT=18080
go run ./cmd/api
go run ./cmd/seed          # optional demo data
```

### Mobile app

```bash
cd apps/mobile
npm ci
echo 'EXPO_PUBLIC_API_URL=http://<YOUR_LAN_IP>:18080' > .env   # never "localhost" on a physical phone
npx expo start             # scan the QR code with Expo Go, or press w for web
```

If `EXPO_PUBLIC_API_URL` is unset, native builds derive the host from the Expo dev server and use port 18080; web uses `localhost:18080` (set `ALLOWED_ORIGINS` on the API to the web origin, e.g. `http://localhost:8081`).

Try matching right away: seed the demo data, log in as `camille@demo.invalid` / `DemoPass123`, and like **Hugo** — he already liked Camille, so it is an instant match (`yanis@`, `lea@`, `sofia@`… are other demo accounts). `go run ./cmd/seed -reset` removes all demo accounts.

## Environment variables

See [`.env.example`](.env.example).

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `JWT_SECRET` | yes | – | ≥ 32 chars, placeholders rejected |
| `DATABASE_URL` | yes | – | Postgres connection string |
| `PORT` | no | `8080` | HTTP port |
| `APP_ENV` | no | `development` | `production` requires SMTP and disables seeding |
| `UPLOAD_DIR` | no | `./data/uploads` | Private photo storage (Docker: volume `/data/uploads`) |
| `TRUSTED_PROXIES` | no | – | Reverse-proxy IPs/CIDRs whose `X-Forwarded-For` is trusted |
| `ALLOWED_ORIGINS` | no | – | CORS / WebSocket origins for the web build |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | prod | – | Password-recovery e-mail; without host the code is logged (dev only) |
| `EXPO_PUBLIC_API_URL` | mobile | auto | API base URL (public by design: no secret ever goes in the app) |

## Database & migrations

Embedded SQL files in `services/api/internal/platform/db/migrations` are applied once each, in order, tracked in `schema_migrations`, guarded by an advisory lock (safe with several replicas). Add a new numbered file to change the schema; never edit an applied one.

Tables: `users, sessions, password_resets, profiles, preferences, interests, user_interests, photos, swipes, matches, messages, blocks, reports, notifications`. All foreign keys cascade from `users`, so deleting an account removes everything attached to it. Required indexes include `users.email`, `messages(match_id, created_at)`, `notifications(user_id, created_at)`, partial indexes for likes and unread messages.

## Tests and quality gates

```bash
# API (needs Postgres; integration tests skip when no DB URL is set)
cd services/api
export DATABASE_URL_TEST='postgres://app:app@localhost:5432/app_test?sslmode=disable'
gofmt -l . && go vet ./... && go test -race ./... && go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npx tsc --noEmit && npm run lint && npm run format:check && npm test

# Browser journey: signup → profile → discover → like → match → live chat (+ account deletion)
npm run export:web                 # with EXPO_PUBLIC_API_URL pointing at a running API
PLAYWRIGHT_CHROMIUM_PATH=/path/to/chrome npm run e2e   # omit the variable to use Playwright's own browser
```

API integration tests cover auth lifecycle, password recovery brute-force limits, profile/age validation, photo security, discovery filters, the full match → chat → read → notification journey, WebSocket delivery, block/report/unmatch/delete, authorization (outsiders get 404), and concurrent opposite likes.

## Deployment

1. Build the API image (`services/api/Dockerfile`, non-root, health check) and run it behind HTTPS (set `X-Forwarded-Proto`; HSTS is emitted automatically).
2. Provide a managed Postgres, a persistent volume (or object-storage replacement of `storage.Store`) for `UPLOAD_DIR`, a strong `JWT_SECRET`, `APP_ENV=production` and SMTP credentials.
3. Build the app with EAS (`EXPO_PUBLIC_API_URL=https://api.your-domain`) or export the web build (`npm run export:web`, static `dist/`).

## Key decisions

- **Matches are conversations.** A `matches` row is the conversation anchor; messages cascade with it. Unmatching or blocking deletes both.
- **Photos are private.** Files live outside any static path; the API hands out HMAC-signed URLs (valid 6–12 h, stable inside a window so image caches work). Blocked users never receive new URLs.
- **Real time without fragility.** Commands use REST; WebSocket (single-use 60 s ticket, no long-lived token in URLs) only pushes events. On reconnect the client refetches, so no state depends on the socket. The hub is in-process; for several replicas replace `realtime.Hub` by a shared bus.
- **Abuse limits are not product limits.** Rate limits (login, swipes 120 burst/60 per minute, messages, uploads, reports) stop scripts; ordinary use never meets them. There is no daily like cap.
- **First swipe wins.** Repeating a swipe is idempotent and returns the original decision, so a pass can never be turned into a match later by replaying requests.
- **Age.** 18+ enforced server-side from the birth date, which cannot be edited after the first save.
- **Moderation.** Reports are stored (`reports.status`) for review; there is no admin UI yet — review through SQL or add an admin feature.
- **Ranking.** Candidate ordering lives in one SQL expression (`discovery/ranking.go`): shared interests, people who already liked you, recent activity, then a per-pair hash. Replace it to evolve recommendations.
- **No push notifications** (APNs/FCM would add third-party credentials); notifications are in-app and real time while the app is open.
- **Deleted from the former boilerplate:** Stripe billing, posts, comments, organizations/multi-tenancy, Tamagui.

## License / contribution

See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
