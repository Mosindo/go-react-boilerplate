# AGENTS.md - Lueur

This repository is `Lueur`, a 100% free dating application built on the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Lueur lets two people create an account, build a profile with photos, discover
compatible profiles, like or pass, match when interest is mutual, chat in real
time, get notifications, manage privacy, block or report, and delete their account.

Non-negotiable product rules (decided by the founder):
- 100% free: no subscription, no paywall, no paid boost, no ads, no artificial like limit
- adults only (18+), enforced server-side
- exact locations are never stored or exposed (coarse ~1 km grid, approximate distances)

Core modules:
- auth
- profiles
- photos
- matching (discovery, swipes, matches)
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

`cmd/api/main.go` (server) and `cmd/seed/main.go` (demo data)
`internal/`
- `app/` (composition root + integration tests)
- `platform/`
  - `config/`, `db/`, `middleware/`, `logger/`, `httpx/`, `authtoken/`
  - `realtime/`, `storage/`, `mediaurl/`, `mailer/`
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
- Validate file metadata and upload constraints

Secrets:
- Never hardcode secrets
- Expected environment variables (see `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - optional: `APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*`

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email` (unique)
- `profiles (gender, birthdate) WHERE discoverable`, `profiles (location_lat, location_lng)`
- `swipes` primary key `(swiper_id, target_id)` and likes received `(target_id, swiper_id)`
- `matches (user_a_id, user_b_id)` unique
- `conversation_participants (conversation_id, user_id)` and `(user_id, conversation_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` unique
- `blocks (blocker_id, blocked_id)` and reverse

---

# TESTS

Before any backend delivery (`services/api`):

- `gofmt -l .` (must be empty)
- `go vet ./...`
- `DATABASE_URL_TEST=... go test -race ./...` (integration tests need a dedicated database)
- `go build ./cmd/api ./cmd/seed`

Frontend/API changes must also keep these healthy in `apps/mobile`:

- `npm ci`, `npm run typecheck`, `npm run lint`, `npm run format:check`, `npm test`, `npm run build:web`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`internal/app`):
- Health (`/health`)
- Auth (`/auth/*`, `/me`, password reset, account deletion)
- Profiles and photos (`/profile`, `/profile/photos`, `/media/photos/:id`)
- Discovery and matching (`/discovery`, `/swipes`, `/likes/received`, `/matches/:id`)
- Chat (`/conversations`, `/conversations/:id/messages`)
- Notifications (`/notifications`)
- Safety (`/blocks`, `/reports`)
- Realtime (`/realtime`)

End-to-end: `npm run e2e:api` and `npm run e2e:web` (see README).

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
  - onboarding (profile, photo)
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
- Reintroducing paid features, paywalls or artificial limits without founder approval

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
