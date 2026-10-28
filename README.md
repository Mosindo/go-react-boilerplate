# Lumen — a free dating app

Lumen is a complete, 100 % free dating app: create an account, build a profile with photos, discover people who fit your preferences, like or pass, match when the interest is mutual, and chat in real time. There is **no subscription, no paywall, no paid boost, no artificial like limit and no ads**.

It is built from the `go-react-boilerplate` base (Go + PostgreSQL API, Expo / React Native client) and keeps its layered architecture.

## Features

| Area | What you get |
|---|---|
| Account | Register, login, refresh-token rotation, logout, **password recovery by emailed code**, change password, **delete account** (all data and photo files removed) |
| Profile | First name, age (18+ enforced server-side, birth date locked after creation), gender, bio, city, interests, discoverability and distance-visibility toggles |
| Photos | Up to 6, JPEG/PNG/WebP, ≤ 8 MB, validated by decoding, re-encoded to JPEG ≤ 1600 px with metadata stripped and EXIF rotation applied, reorder / main photo / delete, served only to authenticated viewers allowed to see the owner |
| Discovery | Ranked queue filtered by both users' gender, age and distance preferences; swipe deck or buttons; full-profile view; "show passed profiles again" |
| Matching | Like + like = match; atomic, duplicate-free, idempotent; a pass never matches; unmatch |
| Chat | Matched users only, text, timestamps, read receipts, unread counts, WebSocket live updates, history pagination, local conversation deletion |
| Notifications | Match and new-message notifications (deduplicated), unread badge, live push over the socket |
| Safety | Block, unblock, report (reason + details + optional conversation), hidden-from-each-other semantics, rate limiting |
| Location | Stored on a ~1 km grid only; others see a distance rounded up to 5 km steps (or nothing if hidden); never coordinates or addresses |

## Stack

- **API**: Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256 (15 min) + rotating refresh tokens, bcrypt, `gorilla/websocket`, `golang.org/x/image` (WebP decoding / resampling). No ORM.
- **Mobile**: Expo 54 / React Native 0.81, TypeScript (strict), React Navigation, TanStack Query, `PanResponder` + `Animated` (no native animation dependency).
- **Infra**: Docker, Docker Compose, GitHub Actions.

## Repository layout

```text
apps/mobile/            Expo client (src/api, auth, components, navigation, realtime, screens, theme)
services/api/
  cmd/api/              HTTP server (composition root) + end-to-end integration tests
  cmd/seed/             demo-data command
  internal/platform/    config, db (+migrations), middleware, realtime hub, storage, mail, logger, httpx, errors
  internal/features/    auth, profiles, photos, discovery, chat, notifications, safety
docs/                   API contract (API.md), architecture
```

Each backend feature follows `handler.go → service.go → repository.go` (+ `model.go`, `routes.go`): thin handlers, business rules in services, SQL only in repositories.

## Quick start (Docker)

```bash
cp .env.example .env
# put a random value in JWT_SECRET (>= 32 chars):  openssl rand -base64 48
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

The API listens on **18080** (host) → 8080 (container). Migrations run automatically at start. Uploaded photos live in the `uploads_data` volume.

Optional demo data (8 clearly fake accounts on the reserved `@demo.invalid` domain, generated gradient photos, password `demo-password-123`):

```bash
docker compose exec api ./seed          # create
docker compose exec api ./seed -purge   # remove
```

The seed refuses to run when `APP_ENV=production`.

## Running without Docker

```bash
# 1. PostgreSQL 14+ (any), then:
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export PORT=18080
cd services/api
go run ./cmd/api            # applies migrations, starts the API
go run ./cmd/seed           # optional demo data
```

## Mobile app

```bash
cd apps/mobile
npm ci
echo 'EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080' > .env     # never "localhost" on a physical phone
npx expo start
```

- Android emulator without `.env` falls back to `http://10.0.2.2:18080`; iOS simulator to `http://localhost:18080`.
- Before calling the app ready, check no other service hijacks the port: `docker compose ps`, `curl http://<LAN_IP>:18080/health`.
- Real-time uses `ws(s)://<same host>/ws`; the token is sent in the first frame, never in the URL.

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | yes | PostgreSQL connection string |
| `JWT_SECRET` | yes | ≥ 32 chars, random; placeholders are rejected at startup |
| `PORT` | no (8080) | API port |
| `APP_ENV` | no (`development`) | `production` hardens startup (requires SMTP, release mode) |
| `UPLOAD_DIR` | no (`./data/uploads`) | Photo storage directory (mount a volume) |
| `ALLOWED_ORIGINS` | no | CORS + WebSocket origins for web builds (native apps need none) |
| `TRUSTED_PROXIES` | no | Reverse-proxy IPs/CIDRs whose `X-Forwarded-For` is trusted; **set it behind a proxy**, otherwise every client shares the proxy IP for rate limiting |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | production | Mail transport for recovery codes. Without `SMTP_HOST` (development) codes are printed in the API log |
| `TEST_DATABASE_URL` | tests | Enables PostgreSQL integration tests |
| `EXPO_PUBLIC_API_URL` | mobile | Public API base URL (contains no secret) |
| `POSTGRES_PASSWORD` | compose | Database password for the compose stack |

See [`.env.example`](.env.example). Secrets are never committed (`.env*` is git-ignored).

## Database & migrations

SQL files in `services/api/internal/platform/db/migrations` are embedded in the binary and applied at startup in lexical order, **once each**, tracked in `schema_migrations` and guarded by an advisory lock (safe with several instances). Add a new file `002_*.sql` for changes; never edit an applied one.

Main tables: `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests` / `user_interests`, `photos`, `swipes`, `matches` / `match_participants`, `messages`, `notifications`, `blocks`, `reports`. Constraints (CHECKs on age/gender/lengths, unique pairs, `user_a < user_b` for matches), foreign keys with `ON DELETE CASCADE`, and the indexes required for the hot paths (`messages(match_id, created_at)`, `notifications(user_id, created_at)`, `photos(user_id, created_at)`, partial indexes for unread / likes / discoverable profiles) are in `001_init.sql`.

## Tests, lint, build

```bash
# Backend (integration tests need PostgreSQL; they use throw-away schemas)
cd services/api
export TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
gofmt -l .            # must print nothing
go vet ./...
go test ./... -count=1
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm test
npx expo export --platform android      # bundle check

# Full-stack smoke against a running API
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run e2e:smoke
```

Backend coverage includes the whole critical path (`cmd/api/journey_test.go`: register → profile → photo → discover → like → match → live chat → read receipts → notifications → unmatch), auth/token forgery, password recovery brute force, account deletion cascade, photo validation, permissions (outsiders get 404 everywhere), blocking, reports, preference filtering, concurrent mutual likes (exactly one match), rate limiting and WebSocket auth. Maestro flows for devices are in `apps/mobile/e2e/maestro`.

## Deployment

1. Build the image: `docker build -t lumen-api services/api` (also contains the `seed` binary).
2. Provide `DATABASE_URL`, `JWT_SECRET`, `APP_ENV=production`, SMTP settings, `TRUSTED_PROXIES`, and a persistent volume on `UPLOAD_DIR`.
3. Terminate TLS at a reverse proxy (HSTS is emitted when `X-Forwarded-Proto: https`) and forward WebSocket upgrades on `/ws`.
4. Point the mobile build at it with `EXPO_PUBLIC_API_URL=https://…` (EAS Build / `expo prebuild`).

## Key technical decisions

- **A match is a conversation.** One `matches` row per unordered pair (`user_a < user_b`, unique) is the chat room; `match_participants` holds per-user state (local deletion cursor). Unmatching deletes the row; swipes stay so the pair never reappears.
- **Atomic matching.** Swipes run in a transaction under a per-pair advisory lock, so two simultaneous likes see each other and exactly one match is created. Swipes are idempotent (the stored result is returned).
- **Server-side rules on every swipe.** The same SQL predicate that builds the discovery queue (mutual gender/age/distance, discoverable, has a photo, no block) is re-checked on swipe; clients cannot like someone they could not have been shown.
- **Ranking** (`discovery/repository.go`): people who already liked you (+10), shared interests (+3 each), proximity, then a stable per-viewer daily shuffle. It is one SQL `ORDER BY`, isolated so a better recommender can replace it.
- **Visibility has one definition** (`profiles.Repository.CanView`): owner, or target exists, not blocked either way, and discoverable or matched. Photos and public profiles both use it; failures answer 404, never 403, to avoid existence leaks.
- **Blocking hides, it doesn't delete** (messages stay as report evidence). The blocker gets `403 blocked` when writing; the blocked person sees the conversation vanish.
- **Distance privacy.** Coordinates are rounded to a 0.01° grid before storage, distances to 5 km steps, location updates are rate limited (20/h) and either party can hide distance.
- **Birth date is immutable** after creation; 18+ is checked server-side to the day.
- **Realtime** is an in-process hub (`platform/realtime`): authenticated by a first-frame JWT, bounded per-client buffers (slow consumers are dropped), sending stays on REST. For several API instances, back `Publish` with Postgres `LISTEN/NOTIFY` or Redis.
- **Rate limiting** is in-process fixed windows (auth per IP; swipes, messages, uploads, reports per user). Swipe and message limits are abuse guards (120 swipes/min), not product limits. Behind multiple instances use a shared limiter at the proxy.
- **Password recovery** uses an 8-character code (HMAC-hashed, 30 min, max 5 attempts, single use) and revokes all sessions; the endpoint never reveals whether an email exists.
- **Photos are never public files.** Random keys on disk, EXIF/GPS dropped by re-encoding, served through an authenticated, access-checked endpoint.
- **Moderation** is data-only for now: `reports` rows (`status=open`) are reviewed with SQL; there is no admin UI. Reports keep their evidence when the reported account is deleted (`reported_id` becomes NULL).
- **Residual risks (accepted, documented):** `POST /auth/register` reveals whether an email is registered (409) for usability; a determined attacker with many fake accounts could still narrow someone's area through repeated distance readings (mitigated by 1 km storage grid, 5 km steps, location rate limit, and the option to hide distance); refresh-token reuse is rejected but does not revoke the whole session family.
- **Not included (by design, documented):** push notifications to the OS (in-app + WebSocket only), email verification, social login, media in chat, admin console.

## API

The full contract is in [`docs/API.md`](docs/API.md); architecture notes in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## License

See the repository license (if any) before reuse.
