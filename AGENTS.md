# AGENTS.md - Lueur

This repository is **Lueur**, a free dating app built on the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Lueur lets adults create a profile, discover people nearby, like or pass, match when the
interest is mutual, chat in real time, and stay safe (block, report, delete their account).

The app is **100 % free**: never add subscriptions, paywalls, paid boosts, artificial like
limits, locked essential features or mandatory ads. Rate limits exist only against abuse.

Core modules:
- auth
- users (account deletion)
- profiles
- photos
- discovery
- matching
- chat
- safety (blocks, reports, moderation)
- notifications
- realtime

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
  - `profiles/`
  - `photos/`
  - `discovery/`
  - `matching/`
  - `chat/`
  - `safety/`
  - `notifications/`
  - `realtime/`

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

Privacy:
- Never return e-mails, birthdates or coordinates of other members
- Store coordinates rounded (~1 km); only show bucketed distances
- Members under 18 are rejected server-side; birthdate is immutable
- Chat access requires an active match without any block

Secrets:
- Never hardcode secrets
- Expected environment variables (see `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - optional: `APP_ENV`, `MEDIA_DIR`, `MEDIA_SIGNING_SECRET`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*`

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `profiles (gender, birthdate)` (discoverable only) and `profiles (latitude, longitude)`
- `swipes (target_user_id, actor_user_id)` (likes)
- `matches (user_low_id | user_high_id, created_at)`
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `blocks (blocked_user_id, blocker_user_id)`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must print nothing)
- `go vet ./...`
- `go test ./...` with `DATABASE_URL` set so integration tests run
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint` and `npm run format:check` healthy in `apps/mobile`
- `./scripts/qa.sh` runs all of the above; with `E2E_API_URL` it also runs the API and web UI end-to-end suites

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/me`, password reset)
- Profiles and photos (`/me/profile`, `/me/preferences`, `/me/photos`)
- Discovery and matching (`/discovery`, `/swipes`, `/matches`)
- Chat (`/chats`, `/chats/:userId/messages`, `/chats/:userId/read`) and realtime (`/realtime/ws`)
- Safety (`/blocks`, `/reports`), account deletion (`DELETE /me`)
- Notifications (`/notifications`)

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
  - register and onboarding (profile, photo)
  - login
  - discovery loads
  - like → match
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
- Introducing paid features, paywalls or artificial limits
- Exposing private member data (e-mail, birthdate, coordinates) to other members

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
