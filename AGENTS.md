# AGENTS.md - Lueur

This repository is `Lueur`, a free dating application built on the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

A 100% free dating app: no subscription, no paywall, no paid boost, no artificial like limit, no mandatory ads.
Members create a profile, discover others, like or pass, match on mutual interest, chat in real time,
and keep full control over privacy, blocking, reporting and account deletion.

Core modules:
- auth
- profiles
- photos
- matching (discovery, likes, matches)
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

`cmd/api/main.go`  
`internal/`
- `platform/`
  - `config/`
  - `db/`
  - `middleware/`
  - `logger/`
  - `errors/`
  - `httpx/`, `media/`, `mailer/`, `realtime/`, `storage/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
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
- Validate uploads by content (type sniffing, size and pixel limits) and re-encode images
- Never expose exact coordinates, birthdates or e-mails of other members
- Every business rule (age >= 18, match reciprocity, conversation membership, blocks) is enforced server-side

Secrets:
- Never hardcode secrets
- Expected environment variables:
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - `APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*` (see `.env.example`)

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email` (unique)
- `swipes (actor_id, target_id)` (primary key) and `swipes (target_id, action, actor_id)`
- `matches (user_a_id, user_b_id)` (unique)
- `conversation_participants (match_id, user_id)` and `(user_id, match_id)`
- `messages (match_id, created_at, id)`
- `notifications (user_id, created_at)`
- `blocks (blocker_id, blocked_id)` and `(blocked_id, blocker_id)`
- `photos (user_id, position)` (unique, deferrable)

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must be empty)
- `go vet ./...`
- `go test ./...` (with `DATABASE_URL_TEST` set so integration tests run)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint` and `npm run build:web` healthy in `apps/mobile`
- The web end-to-end flow (`npm run e2e:web`) must stay green

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/integration_test.go`):
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/password/*`, `/me`)
- Profiles and photos (`/me/profile`, `/me/preferences`, `/me/location`, `/me/photos`, `/media/photos/:id`)
- Matching (`/discovery`, `/swipes`, `/matches/:id`)
- Chat (`/conversations`, `/conversations/:matchId/messages`, `/conversations/:matchId/read`)
- Notifications (`/notifications`)
- Safety (`/blocks`, `/reports`) and account deletion (`DELETE /me`)
- Realtime (`/ws`)

---

# NETWORK AND MOBILE

Mandatory rules to avoid recurring environment issues:

- Never use `localhost` for tests on a physical phone.
- Use `EXPO_PUBLIC_API_URL=http://<LAN_IP>:<PORT_HOST_API>`.
- Verify no local service is hijacking the API port before calling the app "ready":
  - `docker compose ps`
  - `netstat -ano | findstr :<PORT_HOST_API>` (Windows) or `lsof -i :<PORT_HOST_API>` (macOS/Linux)
  - `curl http://<LAN_IP>:<PORT_HOST_API>/health`
- If a conflict is detected, change the Docker host port (example: `18080:8080`) and align:
  - `docker-compose.yml` / `.env` (`API_HOST_PORT`)
  - mobile API fallback
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test:
  - register
  - onboarding (photo upload, location)
  - discovery, like, match
  - chat send/receive/read
  - notifications
  - block, account deletion

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
- Reintroducing paid features, paywalls or artificial limits without approval

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
