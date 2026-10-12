# AGENTS.md - Mellow (dating app, ex go-react-saas)

This repository started as the `go-react-saas` boilerplate and is now a free dating application
(Go API + Expo mobile client). The architecture must remain scalable, secure and maintainable.

---

# PRODUCT GOAL

A 100 % free dating app: no subscription, paywall, paid boost, artificial like limit or mandatory ads.
Never reintroduce monetisation or the former multi-tenant (organization) concept without founder approval.

Core modules (`services/api/internal/features`):
- auth (accounts, sessions, password reset, account deletion)
- profiles (profile, preferences, privacy, location)
- photos (upload pipeline, signed media URLs)
- discovery (candidate feed)
- matching (swipes, matches)
- messaging (chat inside a match, WebSocket push)
- safety (blocks, reports)
- alerts (in-app notifications)

Legacy generic modules (`billing`, `chat`, `comments`, `files`, `notifications`, `posts`, `users`) and their
migrations 002-006 / 008-013 are no longer wired into the router and are pending removal.

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
`cmd/seed/main.go` (demo data, refuses `APP_ENV=production`)
`internal/`
- `platform/`
  - `config/`, `db/` (tracked migrations), `middleware/` (auth, rate limit, CORS), `logger/`, `errors/`
  - `realtime/` (WebSocket hub), `mediaurl/` (signed media URLs), `httpx/` (handler helpers)
- `features/`
  - `auth/`, `profiles/`, `photos/`, `discovery/`, `matching/`, `messaging/`, `safety/`, `alerts/`

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
- Expected environment variables: `DATABASE_URL`, `JWT_SECRET`, `PORT` (default `8080`), `UPLOAD_DIR`,
  `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `MAIL_LOG_TOKENS` (dev only). See `.env.example`.

Privacy:
- Location is stored rounded to ~1 km and only exposed as a coarse distance bucket
- Photos are only served through signed, expiring URLs
- Every business rule (age gate, onboarding, blocks, membership) is enforced server-side

---

# PERFORMANCE

- Use a single shared DB pool
- No DB connection per request
- No N+1 queries
- Prefer explicit pagination on list endpoints
- Keep common API paths O(n) over page size

Required indexes:
- `users.email`
- `swipes (swiper_id, target_id)` (primary key) and `(target_id, swiper_id) WHERE action = 'like'`
- `matches (user_a, user_b)` (unique) and `(user_b)`
- `match_members (user_id, match_id)`
- `chat_messages (match_id, created_at, id)`
- `user_notifications (user_id, created_at, id)`
- `photos (user_id, position)`
- `blocks (blocked_id, blocker_id)`

---

# TESTS

Before any backend delivery:

- `gofmt -l .` (must print nothing), `go vet ./...`, `go test ./...` (with `DATABASE_URL`), `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci`, `npm run typecheck`, `npm run lint`, `npm run format:check`, `npm test` healthy in `apps/mobile`
- `node scripts/qa.js` runs all of the above

If tests fail: fix them before continuing.

Integration tests to keep healthy (`services/api/cmd/api/main_integration_test.go`):
- Health, auth lifecycle, password reset, profile validation, location privacy
- The dating journey: discover -> like -> match -> chat (with WebSocket) -> report/block
- Concurrent mutual likes, photo management, account deletion (data and files)
Browser E2E: `npm run e2e:web` in `apps/mobile`; API smoke: `npm run e2e:smoke`.

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
- A frontend is only "ready" after a real smoke test:
  - register, onboarding (profile, photo, preferences)
  - discovery loads, like, match
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
