# AGENTS.md - Aimant

This repository is **Aimant**, a free dating application built on the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Aimant is 100% free for its members: no subscription, no paywall, no paid boost,
no artificial like limit, no mandatory advertising. Never introduce monetization
gates on core features.

Core modules:
- auth (accounts, sessions, recovery, deletion)
- profiles (profile, preferences, location, interests)
- photos
- matching (discovery, swipes, matches)
- chat
- notifications
- safety (blocks, reports, moderation)
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
  - `httpx/`
  - `ratelimit/`
  - `storage/`
  - `media/`
  - `realtime/`
  - `mailer/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `matching/`
  - `chat/`
  - `notifications/`
  - `safety/`
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

Secrets:
- Never hardcode secrets
- Expected environment variables (see `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - `APP_ENV`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `UPLOAD_DIR`, `SMTP_*`, `MAIL_FROM`

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `profiles` discovery partial index (`latitude, last_active_at`)
- `swipes (swiper_user_id, target_user_id)` and `(target_user_id, action)`
- `matches (user_a_id, user_b_id)` unique + per-user active indexes
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`
- `blocks (blocked_user_id)`, `reports (reported_user_id, created_at)`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must be empty), `go vet ./...`
- `go test ./...` with `DATABASE_URL_TEST` pointing to a disposable Postgres
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci` and `npm run check` (typecheck, lint, format) healthy in `apps/mobile`
- Full suite: `DATABASE_URL=... ./scripts/e2e-local.sh` (adds the API smoke test and the Playwright E2E)

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/*`, `/me`)
- Profiles and photos (`/me/profile`, `/me/location`, `/me/preferences`, `/me/photos`)
- Matching (`/discovery`, `/swipes`, `/matches/:userId`)
- Chat (`/chats`, `/chats/:userId/messages`, `/chats/:userId/read`)
- Notifications (`/notifications`)
- Safety (`/blocks`, `/reports`)
- Realtime (`/realtime`)

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
  - mobile API fallback (`apps/mobile/src/api/client.ts`)
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test:
  - register
  - onboarding (profile, photo, location)
  - discovery loads
  - like -> match
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
- Adding paid features, paywalls or like limits
- Exposing e-mails, birthdates or exact coordinates of a member to other members

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
