# AGENTS.md - Ember

This repository is Ember, a free dating app (Go API + Expo client), derived from the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Ember is a 100 % free dating app: profiles, photos, discovery, likes, matches, realtime chat, notifications, blocking and reporting.

Non-negotiable product rules:
- No subscription, paywall, paid boost, artificial like limit, locked essential feature or mandatory ad.
- Never expose exact location, birth date, e-mail or coordinates of another user.
- Business rules and authorization are enforced on the server.

Backend modules: auth, profiles, photos, discovery, chat, safety, notifications, realtime.

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

Structure:

`cmd/api/main.go` (+ `events.go` composition), `cmd/seed`
`internal/`
- `platform/` — `config`, `db` (+ migrations), `middleware`, `ratelimit`, `mailer`, `httpx`, `logger`, `errors`, `handlers`
- `features/` — `auth`, `profiles`, `photos`, `discovery`, `chat`, `safety`, `notifications`, `realtime`

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

Required indexes (see `001_init.sql`):
- `users.email`
- `conversation_participants (user_id, conversation_id)`
- `messages (conversation_id, created_at, id)`
- `notifications (user_id, created_at, id)`
- `swipes (target_id, swiper_id) WHERE action = 'like'`
- `blocks (blocked_id, blocker_id)`

---

# TESTS

Before any delivery:

- `gofmt -l .` (must print nothing), `go vet ./...`, `go build ./cmd/api ./cmd/seed`
- `DATABASE_URL_TEST=<throwaway db> go test ./...` (integration tests DROP the public schema — never point them at real data)
- In `apps/mobile`: `npm ci`, `npx tsc --noEmit`, `npm run lint`, `npm run format:check`, `npm test`

If tests fail: fix them before continuing. Never disable a test or lint rule to get green.

Integration suites to keep healthy (`services/api/cmd/api`): auth lifecycle, recovery, account deletion, profile validation,
photos, discovery rules, matching (duplicates + concurrency), chat permissions, block/report, notifications, WebSocket, privacy.
Browser journey: `apps/mobile/e2e/web-smoke.js`.

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
  - `docker-compose.yml` / `API_HOST_PORT` in `.env`
  - mobile API fallback
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test:
  - register → profile → preferences → photo
  - discover loads, like, match
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
- Adding any monetization mechanism (see PRODUCT GOAL)

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
