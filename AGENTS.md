# AGENTS.md - Tendre (dating app)

This repository started as the `go-react-saas` boilerplate and is now **Tendre**, a 100% free dating application
(no subscription, paywall, paid boost, artificial like limit or mandatory ads: never introduce any).
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Free dating app: profiles, photos, discovery, likes, matches, real-time chat, notifications, blocking, reporting,
privacy controls and account deletion. See `README.md` for product decisions.

Backend feature modules (`services/api/internal/features`):
- auth
- profiles
- photos
- matching
- chat
- safety
- notifications

Server-side rules are the source of truth (age, mutual preferences, blocks, conversation membership, photo
ownership). Never trust the client.

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
  - `photos/`
  - `matching/`
  - `chat/`
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

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes (all created by migration `014_dating_schema.sql`):
- `users.email`
- `swipes (target_id, swiper_id) WHERE action = 'like'`
- `messages (conversation_id, created_at, id)`
- `conversation_participants (user_id, conversation_id)`
- `notifications (user_id, created_at, id)`
- `profiles (latitude, longitude)`
- `blocks (blocked_id, blocker_id)`

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit` and `npm run lint` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/main_integration_test.go`, real PostgreSQL via
`DATABASE_URL_TEST`):
- Health, auth lifecycle and recovery, profiles, photos, discovery and matching, chat, safety, account deletion,
  real-time delivery.
- Browser journey: `npm run e2e:web` in `apps/mobile` (sign up -> match -> chat -> block -> delete account).
- Also run `npm run lint` and `npm run format:check` in `apps/mobile`.

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
  - `infra/docker-compose.yml`
  - mobile API fallback
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test (`node apps/mobile/scripts/e2e-smoke.js` or `npm run e2e:web`):
  - register and complete the profile with a photo
  - discover loads
  - like / match
  - chat send/read
  - notifications read flow
  - block and account deletion

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
