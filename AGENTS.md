# AGENTS.md - Seren

This repository is Seren, a free dating application (Expo mobile/web client + Go API) built on the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Seren is a 100% free dating app: no subscription, no paywall, no paid boost,
no artificial like quota, no mandatory ads. Never introduce monetization gates.

Core modules:
- auth (accounts, sessions, password reset, account deletion)
- profiles (profile, preferences, interests, approximate location, visibility)
- photos
- matching (discovery, likes/passes, matches)
- chat (match-gated conversations, realtime)
- safety (blocks, reports)
- notifications
- realtime (WebSocket)

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
  - `tokens/`, `httpx/`, `ratelimit/`, `storage/`, `media/`, `mailer/`, `realtime/`
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
  - `APP_ENV`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `UPLOAD_DIR`, `SMTP_*` (optional)

Privacy:
- Never expose e-mail, birthdate or coordinates of other members
- Store and transmit only rounded (~1 km) coordinates; display approximate distances
- Re-encode uploaded images (strip metadata); serve them only via signed URLs

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `swipes (swiper_id, target_id)` (primary key)
- `matches (user_low_id, user_high_id)` (unique)
- `match_members (match_id, user_id)`
- `messages (match_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` (unique)

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (empty) and `go vet ./...`
- `go test ./...` (set `DATABASE_URL_TEST` to run integration tests)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint`, `npm run format:check` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/*_integration_test.go`):
- Health (`/health`)
- Auth (`/auth/*`, `/me`), password reset, account deletion
- Profiles and photos (`/me/profile`, `/me/photos`, `/media/:key`)
- Matching (`/discover`, `/swipes`), including concurrent mutual likes
- Chat (`/conversations`, `/conversations/:matchId/messages`) and permissions
- Safety (`/blocks`, `/reports`)
- Notifications (`/notifications`) and realtime (`/realtime`)

End-to-end: `npm run e2e:smoke` (API) and `npm run e2e:web` (Playwright UI flow).

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
  - register
  - login
  - onboarding (profile, preferences, photo)
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
