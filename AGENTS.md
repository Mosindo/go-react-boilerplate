# AGENTS.md - Lueur

This repository is **Lueur**, a free dating application built on the former `go-react-saas`
boilerplate (product pivot requested by the founder).
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

A modern dating app that is **100% free**: no subscription, no paywall, no paid boost, no
artificial like limits, no mandatory ads. Never add a feature that locks essentials behind payment.

Core modules:
- auth (accounts, sessions, password reset, account deletion)
- profiles (profile, preferences, interests, approximate location)
- photos
- discovery
- matching
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
  - `httpx/`, `ratelimit/`, `storage/`, `imaging/`, `media/`, `mailer/`, `realtime/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `discovery/`
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
- Never expose email, birthdate or coordinates of other users
- Store locations coarsened (~1 km); show only approximate distances
- Photos are re-encoded server-side and served only through signed, expiring URLs
- Every authorization rule (blocks, conversation membership, visibility) is enforced server-side

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
- `swipes (swiper_id, target_id)` (PK) and `swipes (target_id, action, swiper_id)`
- `matches (user_a_id, user_b_id)` (unique) and per-user activity indexes
- `messages (match_id, created_at, id)` and unread partial index
- `notifications (user_id, created_at)`
- `photos (user_id, position)` (unique, deferred)
- `profiles` discovery and location partial indexes

---

# TESTS

Before any delivery run `./scripts/qa.sh` (with `DATABASE_URL_TEST` set), i.e.:

- `gofmt -l .` empty, `go vet ./...`, `go test ./...`, `go build ./cmd/api`
- `npm ci`, `npm run typecheck`, `npm run lint`, `npm run format:check` in `apps/mobile`
- For user-facing flows: `npm run e2e:smoke` and `npm run e2e:web` against a running API

If tests fail: fix them before continuing. Never disable tests or lint to get a green build.

Integration tests to keep healthy (`services/api/cmd/api/integration_test.go`):
- Health, auth lifecycle, password reset
- Profile onboarding, photos, discovery filters
- Like / match / no duplicates / concurrent mutual likes
- Chat permissions, read receipts, notifications
- Block, report, account deletion, realtime, rate limiting

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
  - `apps/mobile/scripts/start-phone.js`
  - README
- A frontend is only "ready" after a real smoke test:
  - register, onboarding (photo upload)
  - discover, like, match
  - chat send/read in real time
  - notifications, block/report, account deletion

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
- Adding paid features, paywalls, like limits or mandatory ads
- Exposing private user data (email, birthdate, coordinates) to other users

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
