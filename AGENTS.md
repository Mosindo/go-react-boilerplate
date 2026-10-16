# AGENTS.md - Lien

This repository is `Lien`, a 100% free dating application (Go API + Expo mobile app), built from the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Lien lets two people create a profile, discover each other, like, match, chat in real time, and stay safe (block, report, delete account).

Non-negotiable product rules:
- 100% free: no subscription, paywall, paid boost, artificial like limit, locked essential feature or mandatory ad.
  Never add a payment provider or billing code.
- Server-side enforcement of every business rule (age gate, preferences, matching, chat membership, blocking).
- Never expose exact location, email or birth date of a user to another user.

Core modules (`services/api/internal/features`):
- auth
- profiles
- photos
- discover
- matching
- chat
- safety
- notifications
- realtime
- account

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
- gorilla/websocket (realtime only)
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
  - `discover/`
  - `matching/`
  - `chat/`
  - `safety/`
  - `notifications/`
  - `realtime/`
  - `account/`

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
- Features never import each other: declare a small interface on the consumer side and wire it in `cmd/api`
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
  - see `.env.example` for the rest (`APP_ENV`, `UPLOAD_DIR`, `SMTP_*`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`)

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `matches (user_a, activity_at)` and `matches (user_b, activity_at)`
- `messages (match_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`
- `swipes (target_id, created_at) WHERE action = 'like'`
- `profiles (latitude) WHERE discoverable`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must print nothing), `go vet ./...`
- `go test ./...` with `DATABASE_URL_TEST` set (integration tests are skipped without it)
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint`, `npm run format:check` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api`, `internal/features/auth`):
- Health, auth (register/login/refresh/reset/delete), protected routes
- Dating journey: onboarding, discover, like, match, chat, read receipts, realtime
- Permissions: chat membership, photo ownership, blocked/hidden profiles
- Discovery filters, concurrent mutual likes (exactly one match), uploads, block/report, account deletion
- Browser journey (`npm run e2e:web` in `apps/mobile`) for signup → onboarding → match → chat

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
- A frontend is only "ready" after a real smoke test (`npm run e2e:smoke` and, when possible, `npm run e2e:web`):
  - register and login
  - profile onboarding with a photo
  - discover loads, like produces a match
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
- Reintroducing payment, subscription or paywall code
- Exposing exact location, email or birth date to other users

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
