# AGENTS.md - Aurore

This repository is **Aurore**, a free dating application (originally the `go-react-saas` boilerplate).
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Aurore is a 100% free dating app: accounts, profiles, photos, discovery, likes, matches,
real-time chat, notifications, blocking, reporting and account deletion.

Hard product rules: no subscription, no paywall, no paid boost, no artificially limited likes,
no mandatory ads, no payment integration.

Core modules:
- auth
- profiles
- photos
- discovery (swipes, matches, recommendations)
- chat
- notifications
- moderation (blocks, reports)

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
- `app/` (router wiring + integration tests)
- `platform/`
  - `config/`
  - `db/` (versioned migrations)
  - `middleware/`
  - `logger/`
  - `errors/`
  - `httpx/`, `storage/`, `mailer/`, `realtime/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `discovery/`
  - `chat/`
  - `notifications/`
  - `moderation/`

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

Required indexes (see `internal/platform/db/migrations/`):
- `users.email` (unique)
- `swipes (from_user_id, to_user_id)` unique, `swipes (to_user_id)` for likes
- `matches (user_a_id, user_b_id)` unique, `matches (user_b_id)`
- `conversation_participants (conversation_id, user_id)` and `(user_id, conversation_id)`
- `messages (conversation_id, created_at, id)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` unique, `profiles` partial indexes for discovery

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must print nothing)
- `go vet ./...`
- `DATABASE_URL_TEST=... go test ./...` (integration tests DROP the public schema of that database)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npm run lint` and `npm test` healthy in `apps/aurore`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/internal/app`):
- Health, auth (register/login/refresh/recovery/JWT), profiles, photos
- Discovery eligibility, swipes/matches (including concurrency), chat permissions
- Blocking, reports, notifications, account deletion, realtime, rate limiting

Critical path E2E (real browser): `apps/aurore/e2e/run.mjs`.

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
  - mobile API fallback (`apps/aurore/src/api/client.ts`)
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test (`npm run e2e` in `apps/aurore`):
  - register and onboarding
  - discover, like, match
  - chat send/receive in real time
  - notifications
  - account deletion

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
- Any payment, subscription or paywall feature

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
