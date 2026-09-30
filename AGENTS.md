# AGENTS.md - Lueur

This repository is `Lueur`, a free dating application (Go API + Expo app), built from the former `go-react-saas` boilerplate at the founder's explicit request.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Lueur is a dating app that must stay 100 % free: no subscription, paywall,
paid boost, artificial like limit or mandatory ads. Rate limits are anti-abuse
only. Product decisions are recorded in `docs/DECISIONS.md`.

Core modules (`services/api/internal/features`):
- auth (accounts, sessions, password reset, account deletion)
- profiles (profile, preferences, interests, coarse location, privacy)
- photos (secure upload/processing, signed media URLs)
- discovery (candidate query + pluggable Scorer)
- matching (swipes, matches, unmatch)
- chat (match-bound conversations, read state, local hide)
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
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `discovery/`
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
- Expected environment variables:
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - see `.env.example` for the optional ones (`APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*`)

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `swipes (swiper_id, target_id)` and `swipes (target_id, action, created_at)`
- `matches (user_low_id, user_high_id)`
- `conversation_participants (conversation_id, user_id)` and `(user_id, conversation_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`
- `blocks (blocker_id, blocked_id)` and `(blocked_id, blocker_id)`

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npx eslint .` and `npx jest` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/auth/refresh`, `/me`, password reset, account deletion)
- Profile and photos (`/profile`, `/profile/photos`)
- Discovery and matching (`/discovery`, `/swipes`, `/matches/:id`)
- Chat (`/conversations`, `/conversations/:id/messages`) including permissions
- Notifications (`/notifications`)
- Safety (`/blocks`, `/reports`)
- Realtime (`/realtime/ticket`, `/realtime`)

Integration tests run when `DATABASE_URL_TEST` is set (`go test ./...`).

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
  - onboarding (profile + photo)
  - discovery loads
  - like -> match
  - chat send/receive in real time
  - notifications read flow
  (automated by `apps/mobile/e2e/web` with Playwright)

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
