# AGENTS.md - Liane

This repository is **Liane**, a 100% free dating application (Go API + Expo/React Native client).
The architecture must remain scalable, secure, maintainable, and easy to evolve.

---

# PRODUCT GOAL

A modern, fully free dating app: accounts, profiles, photos, discovery, likes/matches, real-time chat,
notifications, preferences, privacy, block/report, account deletion.

Product rules that must never be broken:
- No subscription, paywall, paid boost, artificial like limit, locked essential feature, or mandatory ads.
  (Rate limits exist only as anti-abuse safeguards and must stay generous.)
- Adults only (18+), enforced server-side.
- Never expose exact location, coordinates, email or birth date to other users.
- Every business rule is enforced server-side; the client is never trusted.

Backend features:
- auth, account, profiles, photos, discovery (swipes + matches), chat (+ WebSocket), notifications, safety (blocks + reports)

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
`cmd/seed/main.go` (demo data, refuses production)  
`internal/`
- `platform/` (config, db, middleware, httpx, errors, logger, mailer, media, geo, realtime)
- `features/` (auth, account, profiles, photos, discovery, chat, notifications, safety)

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

Indexes live in `internal/platform/db/migrations/` (users.email, photos(user_id, position), swipes, matches,
conversation_participants(user_id, conversation_id), messages(conversation_id, created_at, id), notifications(user_id, created_at), …).
New list endpoints need an index that matches their ORDER BY.

---

# TESTS

Before any backend delivery (from `services/api`):

- `gofmt -l .` (must print nothing)
- `go vet ./...`
- `TEST_DATABASE_URL=postgres://app:app@localhost:5432/app_test?sslmode=disable go test -race ./...`
  (the integration tests WIPE that database; without the variable they are skipped)
- `go build ./cmd/api ./cmd/seed`
- Frontend/API changes must also keep `npm ci`, `npx tsc --noEmit`, `npm run lint` and `npm test` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/*_test.go`): auth lifecycle and recovery, profile validation,
location privacy, matching (incl. concurrent likes), chat permissions, block/report, photos, account deletion, notifications, WebSocket.
`scripts/smoke-api.sh` runs the same journey over real HTTP against a running stack.

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
  - register, onboarding (profile, preferences, photo)
  - discover loads, like, match
  - chat send/read (real time)
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
- Adding any paid/premium mechanic

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
