# AGENTS.md - Lueur

This repository is Lueur, a free dating app built on the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve across product types.

---

# PRODUCT GOAL

This repository now hosts **Lueur**, a free dating app (founder decision, see README).
The platform layer stays generic; product language belongs to `features/` and the mobile app.

Non-negotiable product rules:
- 100% free: no subscription, paywall, paid boost, artificial like limit or mandatory ads
- 18+ only, enforced server-side
- privacy by default: approximate location only, no exact coordinates stored or returned

Core modules:
- auth
- profiles
- photos
- matching
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
  - `httpx/`, `token/`, `urlsign/`, `storage/`, `mailer/`, `realtime/`
- `app/` (composition root)
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
- Expected environment variables:
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - optional: `APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*` (see `.env.example`)

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email` (unique)
- `swipes (swiper_id, target_id)` (primary key) and `swipes (target_id, action)`
- `matches (user_a_id, user_b_id)` (unique)
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`

---

# TESTS

Before any delivery run `./scripts/qa.sh` (or `scripts/qa-lite.ps1` on Windows), which covers:

- `gofmt -l .` (must be empty), `go vet ./...`, `go test ./...`, `go build ./cmd/api`
- `apps/mobile`: `npm ci`, `npx tsc --noEmit`, `npx eslint .`, `npx prettier --check .`, `npx jest`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`internal/app/app_integration_test.go`, needs `DATABASE_URL_TEST`):
- Health (`/health`)
- Auth (`/auth/*`, `/me`)
- Profiles and photos (`/me/profile`, `/me/photos`)
- Matching (`/discover`, `/swipes`)
- Chat (`/conversations`, `/conversations/:id/messages`)
- Notifications (`/notifications`)
- Safety (`/blocks`, `/reports`), account deletion

End-to-end: `apps/mobile/scripts/e2e-smoke.js` (API) and `apps/mobile/e2e/web/critical-flow.js` (UI).

---

# NETWORK AND MOBILE

Mandatory rules to avoid recurring environment issues:

- Never use `localhost` for tests on a physical phone.
- Use `EXPO_PUBLIC_API_URL=http://<LAN_IP>:<PORT_HOST_API>`.
- Verify no local service is hijacking the API port before calling the app "ready":
  - `docker compose ps`
  - `netstat -ano | findstr :<PORT_HOST_API>`
  - `curl http://<LAN_IP>:<PORT_HOST_API>/health`
- If a conflict is detected, change the Docker host port (`API_HOST_PORT`, `POSTGRES_HOST_PORT`) and align:
  - `docker-compose.yml` / `.env`
  - mobile API fallback
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test:
  - register and onboarding (profile, photo, location)
  - discover, like, match
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
