# AGENTS.md - Aube

This repository is Aube, a free dating app built from a Go + Expo boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve across product types.

---

# PRODUCT GOAL

Aube is a free dating app (see `README.md`). It must stay 100% free: no subscription, paywall, paid boost,
artificially limited likes, locked essential feature or mandatory ads.

Core modules:
- auth
- profiles (profile, preferences, photos, interests)
- discovery (feed, likes/passes, matches)
- chat (REST + WebSocket)
- notifications
- safety (blocks, reports)

Keep `internal/platform` free of dating-specific naming.

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

Required indexes:
- `users.email`
- `likes (to_user_id, created_at)` for likes received
- `conversation_participants (user_id, conversation_id)`
- `messages (conversation_id, created_at, id)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` (unique)

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, typecheck and lint healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api`, need `DATABASE_URL_TEST`):
- Health, Auth lifecycle, Password reset
- Profiles and photos, discovery filters
- Match / chat / safety flow, simultaneous likes, account deletion
- WebSocket auth, rate limiting

Mobile (`apps/mobile`): `npm run typecheck`, `npm run lint`, `npm run format:check`, `npm test`, `npm run build:web`, and the browser journey `npm run e2e`.

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
  - mobile API fallback (`src/api/config.ts`)
  - README
- A frontend is only "ready" after a real smoke test:
  - register, onboarding (profile + photo)
  - discovery loads, like, match
  - chat send/read in real time
  - notifications read flow
  (`npm run e2e` in `apps/mobile` automates this in a browser)

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
- Reintroducing paid features, tracking of exact user positions, or product-specific naming into `internal/platform`

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
