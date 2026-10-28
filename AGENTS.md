# AGENTS.md - go-react-saas

This repository is `go-react-saas`, a reusable fullstack boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve across product types.

---

# PRODUCT GOAL

This repository is now **Lumen**, a 100 % free dating app (decision of the founder, replacing the generic boilerplate goal).

Non-negotiable product rules:
- no subscription, paywall, paid boost, artificial like limit, locked essential feature or mandatory ad
- privacy first: never expose exact coordinates, addresses or birth dates to other users
- every business rule is enforced server-side (never trust the client)

Core modules:
- auth, profiles, photos, discovery (swipes + matching), chat, notifications, safety (blocks + reports)

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

`cmd/api/main.go`  
`internal/`
- `platform/`
  - `config/`
  - `db/`
  - `middleware/`
  - `logger/`
  - `errors/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `discovery/`
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

Required indexes (all in `001_init.sql`):
- `users.email` (unique)
- `messages (match_id, created_at, id)`
- `match_participants (user_id, match_id)`
- `notifications (user_id, created_at, id)`
- `photos (user_id, created_at)`
- `swipes (target_id, swiper_id) WHERE action = 'like'`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (no output), `go vet ./...`
- `go test ./...` with `TEST_DATABASE_URL` set (otherwise the PostgreSQL integration tests are skipped)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api`):
- Journey: register → profile → photo → discover → like → match → live chat → read → notifications
- Auth/tokens, password recovery, account deletion
- Photos validation, permissions (outsiders), blocks, reports, discovery filters
- Concurrent mutual likes, rate limiting, WebSocket auth

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
  - register
  - onboarding (profile, preferences, photo)
  - discover loads
  - like / match
  - chat send/read
  - notifications read flow
  (`npm run e2e:smoke` automates this against a running API)

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
- Adding payments, paywalls or any feature that restricts core functionality behind money

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
