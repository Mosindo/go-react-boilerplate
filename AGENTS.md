# AGENTS.md - Sillage

This repository is `Sillage`, a free dating app (Go API + Expo mobile app) built from the former `go-react-saas` boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

Sillage is a dating app that is 100 % free for users:
- no subscription, paywall, paid boost, artificial like limit, mandatory ads
- accounts, profiles, photos, discovery, likes, matches, realtime chat, notifications
- blocking, reporting, privacy controls, account deletion

Core modules (`services/api/internal/features`):
- auth
- profiles
- photos
- discovery (candidates, swipes, matches)
- chat
- notifications
- safety (blocks, reports)

Business rules are enforced server-side. Never trust the client. Never expose exact coordinates.

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

`services/api/cmd/api/main.go` (and `cmd/seed` for demo data)  
`internal/`
- `platform/`
  - `config/`
  - `db/`
  - `middleware/`
  - `logger/`
  - `errors/`
  - `httpx/`, `realtime/`, `storage/`, `mailer/`
- `features/`
  - `auth/`
  - `profiles/`
  - `photos/`
  - `discovery/`
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

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes (see migrations):
- `users.email` (unique)
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `swipes (to_user_id, from_user_id)`, `profiles (latitude, longitude)`, `blocks (blocked_id, blocker_id)`

---

# TESTS

Before any delivery (`make lint test build` runs all of it):

- `gofmt -l .` (must print nothing), `go vet ./...`
- `DATABASE_URL_TEST=<dedicated db> go test ./...` (integration tests WIPE that database)
- `go build ./cmd/api ./cmd/seed` in `services/api`
- In `apps/mobile`: `npm ci`, `npx tsc --noEmit`, `npm run lint`, `npm run format:check`, `npm test`, `npm run build`
- `API_URL=... node scripts/smoke.mjs` against a running API for the end-to-end journey

If tests fail: fix them before continuing. Never skip tests or silence lint to get green.

Integration tests to keep healthy (`services/api/cmd/api/*_integration_test.go`):
- Health (`/health`)
- Auth (`/auth/*`, `/me`), password recovery, account deletion
- Profiles, preferences, photos (upload validation, ordering, access)
- Discovery filters, swipes, matches (including concurrent mutual likes)
- Chat (`/conversations/*`), permissions, read receipts, local deletion
- Notifications, blocks, reports, rate limiting, WebSocket events

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
- A frontend is only "ready" after a real smoke test (`scripts/smoke.mjs` plus the app on a device):
  - register and onboarding (profile, photo, preferences)
  - login
  - discovery loads, like, match
  - chat send/read in realtime
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
- Adding paid features (subscriptions, boosts, paywalls, like limits)
- Exposing exact coordinates, emails or birth dates of other users

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
