# AGENTS.md - Lueur

This repository is **Lueur**, a free dating application built on the former
`go-react-saas` boilerplate (Go API + Expo app). The architecture must remain
scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

A modern dating application that is 100% free for its members:
no subscription, no paywall, no paid boost, no artificial like limit,
no essential feature locked, no mandatory advertising.

Rate limits exist only as anti-abuse protection and must never become quotas.

Core modules:
- auth (accounts, sessions, password reset, account deletion)
- profiles (profile, preferences, privacy, interests, coarse location)
- photos (secure upload pipeline, signed media URLs)
- discovery (eligibility + ranking, swipes)
- matches (mutual likes, conversations, messages, read receipts)
- safety (blocks, reports, moderation status)
- notifications (in-app, realtime)
- realtime (WebSocket fan-out)

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
  - `matches/`
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
- Expected environment variables:
  - `DATABASE_URL`
  - `JWT_SECRET`
  - `PORT` (default `8080`)

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email` (unique)
- `swipes (swiper_id, target_id)` (primary key) and `(target_id, swiper_id) WHERE action = 'like'`
- `matches (user_a_id, user_b_id)` (unique) and `(user_x_id, last_activity_at)`
- `messages (match_id, created_at)`
- `notifications (user_id, created_at)`
- `photos (user_id, position)` (unique)
- `blocks (blocker_id, blocked_id)` and `(blocked_id, blocker_id)`
- `profiles (latitude, longitude)` partial index for discoverable profiles

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci` and `npx tsc --noEmit` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/password/*`, `/me`)
- Profiles & photos (`/profile`, `/profile/photos`, `/media/:key`)
- Discovery (`/discovery`, `/discovery/swipes`)
- Matches & chat (`/matches`, `/matches/:matchId/messages`)
- Safety (`/blocks`, `/reports`)
- Notifications (`/notifications`)
- Realtime (`/realtime/ticket`, `/realtime`)

These live in `services/api/internal/app/app_integration_test.go`; the UI
critical path lives in `apps/mobile/e2e/critical-flow.spec.ts`.

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
  - `docker-compose.yml` (`API_HOST_PORT`)
  - mobile API fallback
  - startup scripts
  - README
- A frontend is only "ready" after a real smoke test:
  - register
  - onboarding (profile, photo, preferences, location)
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
- Introducing paid features, paywalls or artificial usage limits
- Exposing exact user locations or raw coordinates

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
