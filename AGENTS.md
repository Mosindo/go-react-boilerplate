# AGENTS.md - go-react-saas

This repository is **Alma**, a free dating app (Go API + Expo/React Native client), built from the `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Alma is a 100% free dating app: no subscription, no paywall, no paid boost, no artificial like limit, no mandatory ads.
Users can: sign up (18+), build a profile with photos, discover compatible profiles, like or pass, match on mutual likes, chat in real time, get notifications, set preferences and privacy, block/report, and delete their account.

Backend feature modules (`services/api/internal/features`):
- auth
- profiles (profile, preferences, interests, public cards)
- photos
- matching (discovery feed, likes, matches)
- chat (conversations, messages, WebSocket)
- safety (blocks, reports)
- notifications

Never reintroduce billing/paywall features without explicit founder approval.

---

# OFFICIAL STACK

## Frontend
- React / React Native (Expo)
- TypeScript
- Simple navigation / routing
- REST API

## API
- Go
- Gin
- PostgreSQL
- pgxpool
- JWT (HS256)
- No ORM
- No implicit magic

## Infra
- Docker
- Docker Compose

---

# BACKEND ARCHITECTURE

Target structure:

`cmd/api/main.go` (server), `cmd/seed` (dev-only demo data)
`internal/`
- `app/` (router composition, integration tests)
- `platform/`
  - `config/`, `db/` (+ tracked migrations), `middleware/`, `logger/`, `errors/`, `httpx/`, `realtime/`, `mail/`, `validate/`
- `features/`
  - `auth/`, `profiles/`, `photos/`, `matching/`, `chat/`, `safety/`, `notifications/`

Each feature should contain:
- `handler.go`
- `service.go`
- `repository.go`
- `model.go`
- `routes.go`

Rules:
- No mixing `net/http` and Gin
- Only one HTTP server
- Thin handlers
- Business logic outside handlers
- No SQL in handlers
- Shared concerns belong in `platform`
- Domain concerns belong in `features`
- Never break existing modules while extending the platform
- Respect the Go backend structure: `handler -> service -> repository`
- Respect the frontend API client abstraction; no direct network access from UI code

---

# SECURITY REQUIREMENTS

JWT:
- SigningMethodHMAC
- Alg() == HS256
- Expiration is mandatory
- Strict claim validation

Passwords:
- bcrypt
- Never store passwords in plain text

User input:
- Strict validation
- Trim and normalize email
- Validate pagination, filters, and sort inputs
- Validate file metadata and upload constraints

Secrets:
- Never hardcode secrets
- Expected environment variables:
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes (see migrations):
- `users.email` (unique)
- `photos (user_id, position)`
- `likes (target_id, kind, created_at)`
- `matches (user_low, user_high)` unique + `user_high`
- `conversation_participants (user_id, conversation_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (no output), `go vet ./...`
- `go test -race ./...` with `DATABASE_URL_TEST` set (integration tests are skipped without a database)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint`, `npm run format:check` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`internal/app/app_test.go`):
- Health (`/health`), Auth (`/auth/*`, `/me`), Profiles & preferences, Photos
- Discovery (`/discover`), likes/matches, Chat (`/conversations/*`, `/ws`), Notifications
- Safety (`/blocks`, `/reports`), account deletion, permissions

---

# NETWORK AND MOBILE

Mandatory rules to avoid recurring environment issues:

- Never use `localhost` for tests on a physical phone.
- Use `EXPO_PUBLIC_API_URL=http://<LAN_IP>:<PORT_HOST_API>`.
- Verify no local service is hijacking the API port before calling the app "ready":
  - `docker compose ps`
  - `netstat -ano | findstr :<PORT_HOST_API>`
  - `curl http://<LAN_IP>:<PORT_HOST_API>/health`
- If a conflict is detected, change the Docker host port (example: `18080:8080`) and align:
  - `docker-compose.yml` (`infra/docker-compose.yml` includes it)
  - mobile API fallback
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test:
  - register (18+) and onboarding (photo, profile, preferences)
  - discover loads, like, match
  - chat send/read (real time)
  - notifications read flow
  - block/report, logout, account deletion

---

# AGENT EXECUTION RULES

1. Propose a plan before substantial work.
2. Wait for validation when the task has non-obvious tradeoffs.
3. Apply changes through incremental patches.
4. Do not overwrite a full file without explicit validation when a targeted patch is possible.
5. Provide:
   - full diff
   - full modified files if requested
   - build/test outputs
   - new dependencies
6. No new feature should be added while higher-priority agreed tasks remain unvalidated.

---

# FORBIDDEN

- Massive refactor without request
- Unjustified new dependency
- File deletion without validation
- Implicit architecture change
- Mixing `net/http` and Gin
- Reintroducing billing, subscriptions or any paywall without approval

---

# DEFINITION OF DONE

A task is complete only if:
- code compiles
- tests pass
- critical scenario is verified
- runtime documentation reflects the actual repository state

---

# AGENT ROLE

The agent is an execution engineer.
Architecture and product decisions belong to the founder.
