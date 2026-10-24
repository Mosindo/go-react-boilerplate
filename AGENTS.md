# AGENTS.md - Alba

This repository is **Alba**, a 100 % free dating application (Go API + Expo/React Native client), built from the `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Alba lets two adults create a profile, discover each other, like or pass, match when interest is mutual, chat in real time, and stay safe (block, report, delete account).

Product rules that must never be broken:
- No subscription, paywall, paid boost, artificial like limit, locked essential feature or mandatory ad.
- Never expose exact location, birth date, email or coordinates of a user to another user.
- Every business rule is enforced server-side; the client is never trusted.

Core modules:
- auth (accounts, sessions, password recovery, account deletion)
- profiles (profile, preferences, interests, coarse location)
- photos (validated, re-encoded, access-controlled)
- matching (discovery feed, swipes, matches)
- chat (match-bound conversations, read state, realtime)
- safety (blocks, reports)
- notifications

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

Required indexes:
- `users.email`
- `swipes (actor_id, target_id)` (primary key) and `swipes (target_id, action)`
- `matches (user_low, user_high)` (unique) and `matches (user_high)`
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`
- `blocks (blocker_id, blocked_id)` and `blocks (blocked_id)`

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci` and `npx tsc --noEmit` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/forgot-password`, `/me`)
- Profiles and photos (`/me/profile`, `/me/preferences`, `/me/photos`)
- Discovery and matching (`/discover`, `/swipes`, `/matches/:id`)
- Chat (`/conversations`, `/conversations/:id/messages`, `/ws`)
- Safety (`/blocks`, `/reports`)
- Notifications (`/notifications`)
- Integration tests need `DATABASE_URL_TEST`; frontend checks are `npm run typecheck`, `npm run lint`, `npm test`, and the Playwright suite (`npm run e2e`)

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
  - login
  - onboarding (profile, photo, preferences)
  - discovery loads
  - like / match
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
- Reintroducing SaaS-style monetization (plans, subscriptions, paywalls, paid boosts)

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
