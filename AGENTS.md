# AGENTS.md - Lueur

This repository is `Lueur`, a free dating application (Go API + Expo app) built from the former
`go-react-saas` boilerplate. The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

A dating app that is 100% free for its members: no subscription, no paywall, no paid boost, no artificial
like limit, no mandatory ads. Never introduce monetisation that locks an essential feature.

Core modules:
- auth (accounts, sessions, password reset, account deletion)
- profiles (profile, preferences, privacy settings, interests)
- photos
- discovery
- matching
- chat (match-scoped conversations, realtime)
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
  - `authtoken/`, `httpx/`, `media/`, `storage/`, `mailer/`, `realtime/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `discovery/`
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
- Expected environment variables (see `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - `APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*`

Privacy:
- Never return exact coordinates, e-mail or birthdate of another member
- Every chat/profile access must be checked server-side (participant, active match, no block)
- Demo data must stay flagged (`is_demo`, `.invalid` e-mails) and never be seeded in production

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `swipes (swiper_id, target_id)` and `swipes (target_id, swiper_id)`
- `matches (user_a_id, user_b_id)` unique
- `conversation_participants (conversation_id, user_id)` and `(user_id, conversation_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`
- `blocks (blocker_id, blocked_id)` and `(blocked_id, blocker_id)`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must be empty) and `go vet ./...`
- `go test ./...` with `DATABASE_URL_TEST` set so integration tests run
- `go build ./cmd/api` and `go build ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint`,
  `npm run format:check` and `npm test` healthy in `apps/mobile`
- Critical journey changes: `./scripts/e2e.sh`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/main_integration_test.go`):
- Health (`/health`)
- Auth lifecycle, password reset/change, account deletion
- Profiles, preferences, location privacy, photos
- Discovery, swipes, matches (including concurrent mutual likes), undo, unmatch
- Chat (`/conversations`, messages, read receipts, local deletion, stranger access denied) and realtime
- Blocks, reports, notifications, rate limiting

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
- A frontend is only "ready" after a real smoke test (automated by `scripts/e2e.sh` on web):
  - register and onboarding (photo, location)
  - discovery, like, match
  - chat send/read (realtime)
  - notifications read flow
  - block/report and account deletion

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
- Introducing paid features, paywalls or artificial limits without explicit approval

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
