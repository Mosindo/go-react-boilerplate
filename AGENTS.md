# AGENTS.md - Lueur

This repository is **Lueur**, a 100 % free dating application (Go API + Expo app),
built on the former `go-react-saas` boilerplate at the founder's request.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Two people can: sign up, build a profile (photos, details, interests,
preferences), discover compatible profiles, like or pass, match when the
interest is mutual, chat in real time, get notifications, manage privacy,
block or report, and delete their account.

Non-negotiable: the product stays free. No subscription, paywall, paid boost,
artificial like limit, locked essential feature or mandatory ads.

Feature modules:
- auth
- profiles
- photos
- matching (discovery, swipes, matches)
- chat
- safety (blocks, reports)
- notifications

`platform/` stays domain-agnostic; dating logic lives in `features/`.

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
  - `httpx/`
  - `realtime/`
  - `storage/`
  - `mailer/`
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

Privacy:
- Never expose another member's e-mail, exact birthdate or coordinates
- Locations are snapped to a ~1 km grid before storage; only rounded distances are shown
- Uploaded images are sniffed, size-checked and re-encoded (metadata stripped)
- Conversations are readable by their two participants only (404 otherwise)

Secrets:
- Never hardcode secrets
- Expected environment variables (see `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - `APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*` (optional)

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `profiles (gender, birthdate)` (discoverable) and `profiles (latitude, longitude)`
- `swipes (swiper_user_id, target_user_id)` and likes received
- `matches (user_low_id, user_high_id)` unique
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `blocks` in both directions
- `photos (user_id, position)`

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go vet ./...`
- `go test ./...` (with `DATABASE_URL_TEST` so integration tests run)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck` and `npm run lint` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/password/*`, `/me`)
- Profiles and photos (`/me/profile`, `/me/photos`, `/media/photos/:id`)
- Matching (`/discovery`, `/discovery/swipes`, `/matches`)
- Chat (`/matches/:matchId/messages`, `/matches/:matchId/read`, `/ws`)
- Safety (`/blocks`, `/reports`)
- Notifications (`/notifications`)
- Browser E2E: `apps/mobile/e2e/critical-flow.mjs`

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
  - register and complete onboarding (with a photo)
  - login
  - discovery loads
  - like → match
  - chat send/read
  - notifications read flow
- `node scripts/smoke-api.mjs --api http://<LAN_IP>:<PORT_HOST_API>` automates the API part

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
- Dating-specific logic inside `platform/`
- Any paywall, subscription or artificial limit on essential features

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
