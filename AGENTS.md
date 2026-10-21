# AGENTS.md - Alma

This repository is **Alma**, a 100 % free dating application (Go API + Expo client), built from the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Alma lets two people create an account and a profile, add photos, discover profiles, like or pass, match on mutual interest, chat in real time, receive notifications, manage preferences and privacy, block or report someone, and delete their account.

Non-negotiable product rules:
- 100 % free: no subscription, paywall, paid boost, artificial like limit, locked essential feature or mandatory ads.
- 18+ only, enforced server-side at registration.
- Exact location is never stored or exposed (coordinates are rounded to ~1 km; distances are coarse).
- Every important rule is enforced server-side; never trust the client.

Core modules (`services/api/internal/features`):
- auth
- profiles (profile, preferences, interests)
- photos
- discovery
- matches (swipes + matches)
- chat
- notifications
- safety (blocks + reports)

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

`services/api/cmd/api/main.go` (and `cmd/seed`)  
`services/api/internal/`
- `platform/`
  - `config/`
  - `db/`
  - `middleware/`
  - `logger/`
  - `errors/`
  - `httpx/`, `storage/`, `realtime/`, `mail/`, `geo/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `discovery/`
  - `matches/`
  - `chat/`
  - `notifications/`
  - `safety/`

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

Required indexes (all present in `internal/platform/db/sql`):
- `users.email` (unique)
- `swipes (target_id, swiper_id) WHERE action = 'like'`
- `matches (user_a, user_b)` unique, `(user_a|user_b, created_at)`
- `conversation_participants (conversation_id, user_id)` (primary key)
- `messages (conversation_id, created_at, id)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` unique, `(user_id, created_at)`
- `blocks (blocked_id, blocker_id)`

---

# TESTS

Before any backend delivery (from `services/api`, with `TEST_DATABASE_URL` pointing to a PostgreSQL test database):

- `gofmt -l .` (must print nothing)
- `go vet ./...`
- `go test -race -count=1 ./...`
- `go build ./cmd/api`

Frontend/API changes must also keep these healthy in `apps/mobile`:

- `npm ci`
- `npm run typecheck`
- `npm run lint`
- `npm run format:check`
- `npm test`
- `npm run build:web && npm run e2e:web` for user-flow changes (API running on `:18080`)

If tests fail: fix them before continuing. Never skip tests or disable lint to get green.

Integration tests to keep healthy (`services/api/cmd/api/main_integration_test.go`): health, auth (register, login, refresh rotation, recovery, revocation), profile/photos validation, discovery, likes, matches (including concurrent likes), chat and permissions, notifications, blocks, reports, account deletion, WebSocket delivery.

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
  - `docker-compose.yml`
  - mobile API fallback
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test:
  - register (with birth date) and onboarding (profile, preferences, photo)
  - login
  - discover loads, like, match
  - chat send/read (real time)
  - notifications read flow
  - block / delete account

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
- Introducing any paid feature, paywall or artificial limit on core features
- Storing or exposing exact user locations

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
