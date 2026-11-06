# AGENTS.md - Lueur

This repository is `Lueur`, a free dating application (Go API + Expo/React Native client), built from the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Lueur is a 100 % free dating product. There must never be a subscription, paywall, paid boost,
artificial like limit, locked essential feature or mandatory advertising.

Core modules:

- auth (accounts, sessions, recovery, deletion)
- profiles (profile, preferences, interests, privacy)
- photos
- matching (discovery, swipes, matches)
- chat (conversations created from matches, messages, read state)
- moderation (blocks, reports)
- notifications
- realtime (WebSocket hub in `platform`)

Product rules that must stay true:

- Users must be 18+ (enforced by the database and the service).
- Never expose exact coordinates, birth date or email of another user.
- A user can never read a conversation they do not belong to.
- Blocking is mutual in effect and removes the match.

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
  - `db/` (embedded, tracked SQL migrations)
  - `middleware/`
  - `logger/`
  - `httpx/`
  - `realtime/`
  - `storage/`
  - `mailer/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `matching/`
  - `chat/`
  - `moderation/`
  - `notifications/`

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
- `swipes (actor_id, target_id)` (primary key) and `swipes (target_id, actor_id) WHERE action = 'like'`
- `matches (user_a, user_b)` (unique) and `matches (user_b, user_a)`
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` (unique)

---

# TESTS

Before any backend delivery (from `services/api`):

- `gofmt -l .` (must print nothing)
- `go vet ./...`
- `TEST_DATABASE_URL=postgres://... go test ./...` (integration tests need PostgreSQL)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep this healthy in `apps/mobile`: `npm ci`, `npx tsc --noEmit`, `npm run lint`, `npm run format:check`, `npm test`, `npm run build:web`
- Critical journeys: `npm run test:e2e` (Playwright, empty database, see README)

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/integration_test.go`):

- Health, auth lifecycle (register, login, refresh rotation, logout), password recovery
- Profile/preferences validation (age 18+, coordinates never returned)
- Photos (type sniffing, size limits, ordering, ownership, access control)
- Discovery, swipes, match creation (including concurrent mutual likes), chat permissions, read state
- Blocking, reporting, privacy, unmatch, account deletion cascade
- Realtime ticket + WebSocket delivery

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
- A frontend is only "ready" after a real smoke test (the Playwright E2E covers it):
  - register and onboarding (profile, preferences, photo)
  - discovery loads, like, match
  - chat send/read in real time
  - notifications read flow

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
- Reintroducing product-specific naming into shared modules without approval

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
