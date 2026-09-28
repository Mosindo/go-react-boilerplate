# Pending removal (owner validation required)

The repository was pivoted from a generic SaaS boilerplate to a free dating app. Nothing below was deleted (`AGENTS.md`: no file deletion without validation). Everything listed is **unwired** from `cmd/api/main.go` (no route serves it) and is safe to remove once the owner agrees. Removal is mechanical: delete the paths, run `gofmt`, `go vet ./...`, `go test ./...`.

## Go code, unwired (still compiles, still has its own tests where noted)

| Path | Why obsolete |
|---|---|
| `services/api/internal/features/billing/` (incl. `service_test.go`) | Stripe checkout/subscriptions. The app has no payments. |
| `services/api/internal/features/posts/` | Generic feed; not part of the dating product. |
| `services/api/internal/features/comments/` | Comments on posts. |
| `services/api/internal/features/files/` | Generic file storage; photos now live in `photos` (bytea, re-encoded). |
| `services/api/internal/features/users/` | Generic user directory (`GET /users`); replaced by `discovery` (`/discover`, `/users/:id/profile`). Its SQL still references the dropped `users.organization_id`, so it would fail at runtime if wired again. |
| `services/api/internal/platform/errors/errors.go` | Tiny `Wrap` helper, now unused (`platform/httpx` carries error conventions). |
| Stripe redaction patterns in `services/api/internal/platform/logger/logger.go` | Harmless; remove with billing. |

## Database objects (migrations are append-only; drop with a NEW migration, never edit old ones)

| Object | Notes |
|---|---|
| tables `organizations`, `subscriptions` (migrations 012, 013) | Tenant and billing model. `users.organization_id` was already dropped by migration 014, so nothing references them any more. |
| tables `posts`, `comments`, `votes`, `files` (002, 003, 006, 010) | Legacy generic content. |
| tables `conversations`, `conversation_participants`, `messages` (004, 008, 009, 011) | Legacy chat. Incompatible with the dating chat, which uses `match_conversations` and `match_messages`. Untouched. |
| migrations 001..013 themselves | Must stay while any database may still need to replay them. Squashing them into a baseline is an owner decision. |

Note: the `notifications` table is **reused** (not legacy): migration 014 drops `is_read` (read state is `read_at`) and constrains `type` to `match | message`.

## Config, infra and docs

| Path | Notes |
|---|---|
| `.codex/config.toml` (`[mcp_servers.stripe]`) | Stripe MCP reference config. |
| `docs/agents/payments-agent.md` | Agent brief for payments; there are none. |
| `docs/ARCHITECTURE.md` | Still describes organizations, billing and tenant isolation; needs a rewrite for the dating architecture (kept untouched to avoid an unrequested large doc rewrite). |
| `scripts/smoke-api.ps1`, `scripts/smoke-all.ps1`, `scripts/qa-lite.ps1` | Smoke flow calls `/users` and `/posts`; rewrite around register, profile, discover, swipe, chat. |
| `apps/mobile` legacy screens/endpoints for posts, users list, billing | Owned by the mobile track; remove once the new screens land. |

## Already removed from the live surface (no action needed)

- Stripe env vars (`STRIPE_*`, `APP_BASE_URL`) are gone from `platform/config`, `.env.example`, `infra/.env.example`, `docker-compose.yml`, `infra/docker-compose.yml`, `README.md`.
- The `oid` (organization) JWT claim is gone; register/login no longer need organizations.
