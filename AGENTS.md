# AGENTS.md - go-react-saas

This repository is `go-react-saas`, a reusable fullstack boilerplate.
The architecture must remain scalable, secure, maintainable, and easy to evolve across product types.

---

# PRODUCT GOAL

This repository is the backend and client of a 100% free dating app:
- no payments, no paywall, no quotas on likes
- the server never trusts the client (authorization on every resource, server-side 18+ enforcement, bucketed distances only, coordinates rounded to ~1 km)
- the HTTP contract is `docs/API.md`; it is the single source of truth for API and mobile

Core modules:
- auth
- profiles (profile, preferences, interests, location)
- photos
- discovery (discover queue, swipes)
- matches
- chat (REST + WebSocket `realtime`)
- notifications
- safety (blocks, reports)

Legacy generic-SaaS modules (`billing`, `posts`, `comments`, `files`, `users`, organizations/tenants) are unwired from the API and left on disk until the owner validates their removal (see `docs/PENDING_REMOVAL.md`). Do not re-wire them and do not add product features to them.

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
  - `chat/`
  - `realtime/`
  - `notifications/`
  - `safety/`
  - (legacy, unwired: `users/`, `files/`, `posts/`, `comments/`, `billing/`)

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
- Shared concerns belong in `platform` (`httpx` error/pagination helpers, `middleware`, `jwtauth`, `imaging`, `mailer`, `events`, `geo`, `validate`)
- Services publish realtime events through the `events.Publisher` interface, never through the hub directly
- Migrations are append-only (`schema_migrations` tracks them); never edit an applied file
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

Required indexes (legacy tables keep theirs; the dating schema adds discover/chat equivalents in migration 014):
- `users.email`
- `posts (author_id, created_at)`
- `comments (post_id, created_at)`
- `conversation_participants (conversation_id, user_id)`
- `messages (conversation_id, created_at)`
- `notifications (user_id, created_at)`
- `files (owner_user_id, created_at)`
- `profiles (latitude, longitude)`, `swipes (from_user_id, to_user_id)`
- `match_messages (conversation_id, created_at)`, `notifications (user_id, created_at)` (chat uses `match_conversations`/`match_messages`; the legacy `conversations`/`messages` tables are untouched)

---

# TESTS

Before any backend delivery:

- `gofmt ./...`
- `go test ./...`
- `go build ./cmd/api`
- Frontend/API changes must also keep `npm ci` and `npx tsc --noEmit` healthy in `apps/mobile`

If tests fail: fix them before continuing.

Integration tests need `DATABASE_URL` (they skip without it; each test gets its own scratch database).

Integration tests to keep healthy:
- Health (`/health`)
- Auth (`/auth/register`, `/auth/login`, `/auth/refresh`, `/me`, password reset)
- Profile, preferences, location, photos
- Discovery filters, swipes, concurrent-like match creation (`/discover`, `/swipes`)
- Matches and chat permissions (`/matches`, `/conversations/:id/messages`)
- Notifications (`/notifications`)
- Safety (`/blocks`, `/reports`) and account deletion cascade
- WebSocket (`/ws`)

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
  - register
  - login
  - profile + location + photo
  - discover load
  - swipe / match
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
