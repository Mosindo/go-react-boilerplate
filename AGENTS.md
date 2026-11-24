# AGENTS.md - Lueur

This repository is `Lueur`, a free dating application (Expo mobile app + Go API) built on the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Lueur lets people create an account and a profile, add photos, discover nearby
profiles, like or pass, match on mutual likes, chat in real time, receive
notifications, manage preferences and privacy, block or report someone and
delete their account.

The product is 100% free: no subscription, paywall, paid boost, artificial like
cap, locked essential feature or mandatory advertising. Anti-abuse rate limits
are allowed only far above human usage.

Core modules:
- auth
- profiles
- photos
- matching
- chat
- safety
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
  - `httpx/`, `tokens/`, `ratelimit/`, `storage/`, `media/`, `mailer/`, `realtime/`, `health/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
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

Secrets:
- Never hardcode secrets
- Expected environment variables (see `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - `APP_ENV`, `UPLOAD_DIR`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `SMTP_*`

Privacy:
- Never expose email, birthdate or coordinates of a member to another member
- Store locations rounded to ~1 km; show only approximate distances
- Photos are re-encoded (metadata stripped) and served through signed, expiring URLs
- Enforce 18+ on the server

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email` (unique)
- `profiles (latitude, longitude)` for discoverable profiles
- `swipes (swiper_id, target_id)` (primary key) and likes by target
- `matches (user_low_id, user_high_id)` (unique)
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` (unique)

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go vet ./...`
- `go test ./...` (with `DATABASE_URL_TEST` to run integration tests)
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npx eslint .` and `npx jest` healthy in `apps/mobile`
- `scripts/qa.sh` (or `scripts/qa-lite.ps1`) runs the whole gate

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/*`, `/me`, password recovery, account deletion)
- Profiles and photos (`/me/profile`, `/me/photos`, `/media/photos/:id`)
- Matching (`/discovery`, `/swipes`, concurrent likes)
- Chat (`/conversations`, `/conversations/:id/messages`) and permissions
- Safety (`/blocks`, `/reports`)
- Notifications (`/notifications`) and realtime (`/realtime`)

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
- A frontend is only "ready" after a real smoke test (`npm run e2e:api`, `npm run e2e:web`):
  - register and onboarding
  - discovery loads
  - like and match
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
- Paid features, paywalls or artificial usage caps

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
