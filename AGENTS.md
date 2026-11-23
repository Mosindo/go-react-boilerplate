# AGENTS.md - Hearth (built on go-react-saas)

This repository hosts **Hearth**, a free dating app, built on the `go-react-saas` fullstack foundation.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

The founder explicitly requested a dating product: a 100% free dating app (no subscription,
no paywall, no paid boost, no artificial like limits, no mandatory ads).

Keep the `platform` layer domain-agnostic. Dating-specific language belongs in `features`.

Core modules:
- auth
- profiles (profile, preferences, privacy settings, location, interests)
- photos
- matching (discovery, swipes, matches)
- chat
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
  - `httpx/`, `ratelimit/`, `storage/`, `media/`, `mailer/`, `realtime/`
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
- Expected environment variables (see `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - `APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*`

Privacy:
- Never return exact coordinates; store them snapped (~1 km) and expose whole-km distances only
- Photos are re-encoded (metadata stripped) and served through signed, expiring URLs
- Chat access requires participation in an active match and no block in either direction

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `swipes (swiper_id, target_id)` and `swipes (target_id, action, swiper_id)`
- `matches (user_a_id, user_b_id)` unique
- `blocks (blocker_id, blocked_id)` and `blocks (blocked_id, blocker_id)`
- `photos (user_id, position)` unique
- `reports (reported_id, created_at)`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must print nothing)
- `go vet ./...`
- `go test ./...` with `DATABASE_URL` set so integration tests run
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npm run lint` and `npm run format:check` healthy in `apps/mobile`
- Shortcut: `node scripts/qa.js` (add `--smoke` with a running API)

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/integration_test.go`):
- Health (`/health`)
- Auth (`/auth/*`, `/me`, password recovery, account deletion)
- Profiles and photos (`/profile`, `/profile/photos`, `/media/photos/:id`)
- Discovery and matching (`/discover`, `/swipes`, `/matches`)
- Chat (`/conversations/:id/messages`, `/conversations/:id/read`, `/realtime`)
- Safety (`/blocks`, `/reports`)
- Notifications (`/notifications`)
- Browser E2E: `apps/mobile/e2e/web-critical-flow.js`

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
  - startup scripts (`apps/mobile/scripts/start-phone.js`)
  - README
- A frontend is only "ready" after a real smoke test:
  - register
  - login
  - onboarding (photo + location)
  - discovery loads
  - like → match
  - chat send/read (realtime)
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
- Reintroducing product-specific naming into `platform` modules without approval
- Paywalls, quotas or paid features (the product is free by founder decision)

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
