# AGENTS.md - Ember

This repository is **Ember**, a 100% free dating app (Go API + Expo/React Native client) built from the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Ember is a free dating app. There is no subscription, paywall, paid boost, artificial like limit or locked feature, and none may be added.

Core modules (`services/api/internal/features`):
- auth (accounts, sessions, password recovery, account deletion)
- profiles (profile, preferences, interests, photos)
- matching (discovery, swipes, matches)
- chat (conversations, messages, WebSocket)
- notifications
- safety (blocks, reports)

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
  - `matching/`
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
- Privacy: never expose exact coordinates, birth dates or e-mail addresses of other users; photos are only served through authorized endpoints
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
- `swipes (swiper_id, target_id)` and incoming likes `(target_id, swiper_id)`
- `matches (user_a, user_b)`
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must print nothing), `go vet ./...`
- `DATABASE_URL_TEST=... go test ./...` (integration tests need PostgreSQL)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npm run lint`, `npm run format:check` and `npm test` healthy in `apps/mobile`
- `npm run e2e:api` (apps/mobile) exercises the critical path against a running API

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/main_integration_test.go`):
- Health, auth guards
- Registration -> profile -> photos -> discovery -> like -> match -> chat -> notifications
- Permissions (conversation access, photo access), blocks, reports, account deletion
- Simultaneous mutual likes produce exactly one match

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
  - register and create a profile with a photo
  - discovery loads and like/pass work
  - a mutual like produces a match
  - chat send/read works in real time
  - notifications and blocking work

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
- Any monetization mechanism (payments, subscriptions, gated features)

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
