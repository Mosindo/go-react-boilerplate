# Architecture

Aurore is a single Go API (Gin, one HTTP server) backed by PostgreSQL, consumed by an Expo app that
runs on iOS, Android and web. See the root `README.md` for setup, API tables and security notes; this
document explains how the pieces fit and where to extend them.

## Backend (`services/api`)

```
cmd/api            HTTP server (graceful shutdown, timeouts)
cmd/seed           demo accounts (refuses APP_ENV=production)
internal/app       NewRouter: wires config → repositories → services → handlers
internal/platform  config · db (migrations) · middleware · httpx · storage · mailer · realtime · logger
internal/features  auth · profiles · photos · discovery · chat · notifications · moderation
```

Layering per feature: `routes.go` → `handler.go` (HTTP only) → `service.go` (rules) → `repository.go`
(SQL only) with `model.go` for types. Features talk to each other through small interfaces declared
by the consumer (`Cards`, `Notifier`, `Publisher`, `AccountCleaner`), never by importing repositories.

### Request flow

`SecurityHeaders → CORS → RequestID → metrics/logging → body cap → [rate limit] → RequireUser (JWT HS256,
mandatory exp) → handler`. Authorization is always re-checked in SQL (membership, blocks, eligibility).

### Discovery and matching

- `discovery/repository.go: candidateSQL` is the **only** definition of "who may be shown to whom":
  visible and complete profiles, mutual gender interest, mutual age ranges, mutual distance limits,
  not already swiped, not blocked either way. It feeds both the feed and the swipe authorisation.
- Ranking is the trailing `ORDER BY` (`rankTail`). To plug in another recommender, change that clause
  or have a service return an ordered id list and keep `candidateSQL` as the filter.
- `Swipe` takes a per-pair advisory lock, inserts the swipe, and when the other side already liked,
  creates `matches` (ordered pair, unique) + `conversations` + participants in the same transaction.

### Chat and realtime

Conversations exist only for matches (`conversations.match_id` unique, cascade). Membership is checked
on every route and returns 404 to non-members. Writes are REST; delivery is WebSocket
(`/realtime/ticket` → single-use 30 s ticket → `/ws`). The in-memory `realtime.Hub` is the single
integration point for a future pub/sub.

### Photos

`photos.Process` sniffs the bytes, bounds the dimensions, decodes and re-encodes JPEG variants
(1080 px and 360 px thumbnail). Files live behind `storage.Store` (local disk implementation rejects
any key escaping its root) and are only served by `/photos/:id/(image|thumb)` after a permission
check. A replaced photo gets a new id so URLs are immutable and cacheable.

### Data lifecycle

Account deletion removes files, then the `users` row; every dependent table cascades. `reports`
use `ON DELETE SET NULL` to keep moderation history without personal links.

## App (`apps/aurore`)

```
src/api         client (single-flight refresh, typed errors) + endpoints + query keys
src/auth        session provider (SecureStore / localStorage)
src/realtime    WebSocket + cache updates (setQueryData / invalidate)
src/components  design system (ui.tsx), AuthImage (authenticated images), forms, profile cards, feedback
src/screens     auth, onboarding, discover (swipe deck), matches, chat, notifications, profile/settings
src/navigation  root navigator (auth → onboarding → tabs + stacks)
```

State is server-state first (TanStack Query); the discover deck is local state that resets when
preferences or location change. UI code never calls `fetch`: everything goes through `src/api`.
