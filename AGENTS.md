# AGENTS.md - Lien

Fullstack dating app (Go API + Expo client). The architecture must remain scalable, secure and maintainable.

---

# PRODUCT GOAL

This repository is **Lien**, a 100 % free dating application (no subscription, paywall, paid boost, artificial like limits or mandatory ads).

Modules: auth, profiles, photos, matching, chat, notifications, safety (blocks/reports). See README.md and docs/ARCHITECTURE.md.

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
  - `users/`
  - `photos/`
  - `chat/`
  - `profiles/`
  - `matching/`
  - `safety/`
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
  - see `.env.example` for the optional ones

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes (all defined in migrations):
- `users.email`
- `messages (match_id, created_at, id)`
- `notifications (user_id, created_at, id)`
- `swipes (from_user_id, to_user_id)` (primary key) and likes received
- `matches (user_a, user_b)` unique
- `profiles` discovery indexes (visibility, latitude)

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/me`)
- Profiles, photos, discovery (`/me/profile`, `/me/photos`, `/discover`)
- Matching (`/discover/swipes`) and chat (`/conversations`, `/conversations/:id/messages`)
- Safety (`/blocks`, `/reports`), notifications (`/notifications`), account deletion (`DELETE /me`)
- Browser scenario in `e2e/` (sign up -> match -> chat)

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
  - mobile API fallback (`apps/mobile/src/api/client.ts`)
  - README
- A frontend is only "ready" after a real smoke test:
  - register
  - login
  - onboarding (profile, photo, location)
  - discover, like, match
  - chat send/read
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
