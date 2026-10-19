# AGENTS.md - go-react-saas

This repository is `go-react-saas`, a reusable fullstack boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve across product types.

---

# PRODUCT GOAL

This repository is Ember, a 100% free dating application (founder decision, supersedes the former domain-agnostic boilerplate goal).

Non-negotiable product rules:
- no subscription, paywall, paid boost, artificial like limit, or mandatory ads
- 18+ only, enforced server-side
- never expose exact location, birth date or email of other users
- every important rule is enforced on the server

Core modules:
- auth
- profiles (profile, preferences, interests, photos)
- matching (discover, swipes, matches)
- chat
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

`cmd/api/main.go`, `cmd/seed/main.go`  
`internal/`
- `server/` (router wiring, integration tests)
- `platform/`
  - `config/`
  - `db/` (+ `migrations/`)
  - `middleware/`
  - `logger/`
  - `errors/`
  - `pagination/`
  - `realtime/`
  - `storage/`
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
- `profiles (is_visible, gender)`
- `conversation_members (user_id)`
- `messages (conversation_id, created_at, id)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`
- `swipes` incoming likes (partial)

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must print nothing), `go vet ./...`
- `go test ./...` with `DATABASE_URL_TEST` set so integration tests run
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npm run lint` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/internal/server`):
- Health (`/health`)
- Auth (`/auth/*`, `/me`), token validation, password recovery, account deletion
- Profiles, preferences, photos (`/profile*`, `/photos/:id`)
- Discovery and matching (`/discover`, `/swipes`, `/matches`, `/likes/received`)
- Chat (`/conversations*`) and realtime (`/ws`)
- Notifications (`/notifications*`)
- Safety (`/blocks`, `/reports`)

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
  - profile + photo + preferences
  - discover loads
  - like -> match
  - chat send/read (realtime)
  - notifications read flow
  (automated by `npm run e2e:smoke` in `apps/mobile`)

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
- Anything that paywalls or limits core features (see PRODUCT GOAL)

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
