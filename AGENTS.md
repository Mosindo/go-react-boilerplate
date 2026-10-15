# AGENTS.md - Alma

This repository is `Alma`, a free dating application (Go API + Expo mobile app) built from the `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Alma is a modern dating app that is 100% free for its users:
- no subscription, paywall, paid boost, artificial like limit or mandatory ads
- no essential feature locked behind a payment

Core modules (`services/api/internal/features`):
- auth (sessions, recovery, account deletion)
- users (profile, preferences, interests, location, privacy)
- files (photos: validation, re-encoding, signed URLs)
- matching (discovery, likes, matches)
- chat (match-bound conversations, read state, realtime)
- notifications
- safety (blocks, reports)

Product rules that must keep holding:
- everything important is verified server-side (age 18+, eligibility, permissions, uniqueness)
- exact coordinates are never exposed; only an approximate distance is
- a user can only read conversations they belong to
- deleting an account deletes the data and the files

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
  - `files/`
  - `chat/`
  - `posts/`
  - `comments/`
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
- `posts (author_id, created_at)`
- `comments (post_id, created_at)`
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `files (owner_user_id, created_at)`

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go vet ./...`
- `go build ./cmd/api ./cmd/seed`
- Integration tests need PostgreSQL: set `DATABASE_URL_TEST` (they are skipped otherwise, never rely on that)
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npm run lint`, `npm run format:check` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/me`)
- Profile, preferences and photos (`/me/profile`, `/me/preferences`, `/photos`)
- Discovery and matching (`/discover`, `/swipes`, `/matches/:userId`)
- Chat (`/chats`, `/chats/:id/messages`, `/chats/:id/read`)
- Notifications (`/notifications`)
- Safety (`/blocks`, `/reports`) and account deletion (`DELETE /me`)

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
- A frontend is only "ready" after a real smoke test:
  - register
  - login
  - users load
  - posts load
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
- Introducing paid features (subscriptions, boosts, paywalls, like quotas) or ads: the product is free by design

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
