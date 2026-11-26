# AGENTS.md - Lueur

This repository is **Lueur**, a free dating application (Go API + Expo app) built on the former
`go-react-saas` boilerplate. The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

A dating app that is 100% free for its members: no subscription, paywall, paid boost,
artificial like quota or mandatory ads. Any change must preserve that.

Core modules:
- auth (accounts, sessions, password reset, account deletion)
- profiles (profile, preferences, interests, coarse location, privacy)
- photos (upload, ordering, signed media URLs)
- matching (discovery, swipes, matches, likes received)
- chat (match-gated conversations, read receipts)
- safety (blocks, reports, moderation escalation)
- notifications (in-app + realtime)

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
  - `httpx/`, `media/`, `realtime/`, `mailer/`
- `app/` (composition root + HTTP integration tests)
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
- Expected environment variables (full list in `.env.example`):
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)
  - `APP_ENV`, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `UPLOAD_DIR`, `SMTP_*`

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `swipes (swiper_id, target_id)` (primary key) and likes received `(target_id, created_at)`
- `matches (user_a_id, user_b_id)` unique
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)`

---

# TESTS

Before any delivery, run `scripts/qa.sh` (or `scripts/qa.ps1`), which covers:

- `gofmt -l .`, `go vet ./...`
- `go test ./...` with `DATABASE_URL_TEST` set so HTTP integration tests run
- `go build ./cmd/api` and `go build ./cmd/seed`
- `apps/mobile`: `npm ci`, `npm run typecheck`, `npm run lint`, `npm run format:check`, `npm test`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/internal/app`):
- Health (`/health`)
- Auth lifecycle and password reset
- Profiles, photos and discovery filters
- Like -> match -> chat with realtime, permissions, notifications
- Block/report, concurrent mutual likes, account deletion, rate limiting

End-to-end: `npm run e2e:smoke` (API) and `npm run e2e:web` (Playwright UI).

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
  - discovery deck loads
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
- Reintroducing paid features, quotas or paywalls
- Exposing exact location, e-mail or birthdate of a member to another member

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
