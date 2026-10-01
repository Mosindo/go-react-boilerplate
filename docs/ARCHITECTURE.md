# Lumen Architecture

Lumen is a monorepo: an Expo/React Native client (`apps/mobile`) and a Go API (`services/api`) backed by PostgreSQL.
See the [README](../README.md) for setup, environment variables and the endpoint list; this document explains the boundaries.

## Backend (`services/api`)

```text
cmd/api            HTTP server + maintenance CLI (promote-admin)
cmd/seed           Development-only demo data
internal/platform  Shared concerns: config, db (+ migrations), middleware, realtime, storage, mailer, httpx, logger
internal/features  Domain modules, each handler -> service -> repository (+ model.go, routes.go)
  auth             Credentials, sessions, refresh rotation, password recovery
  users            Account self-service (deletion)
  profiles         Profile, preferences, interests, coarse location, public/own views
  photos           Upload validation + re-encoding, ordering, protected serving
  matching         Discovery, swipes, matches (single source of truth for eligibility)
  chat             Conversations/messages, restricted to active, non-blocked matches
  notifications    Stored notifications + live push
  safety           Blocks, reports, admin moderation
```

Rules that keep it maintainable:

- Handlers parse and map errors; services validate and decide; repositories own SQL (always parameterized).
- Cross-feature calls go through small interfaces declared by the consumer (`Notifier`, `Publisher`, `ProfileReader`, `Access`), wired in `cmd/api/main.go`.
- Authorization is data-driven: chat/photo/profile queries join `matches`/`blocks`, so a non-member cannot even learn a resource exists (404).
- `matching.eligibleFrom` defines who may see whom; discovery adds swipe-history exclusion and ranking (`candidateScore`), swipes reuse the same predicate.
- Match creation is serialized per user pair (`pg_advisory_xact_lock`) and protected by `UNIQUE (user_a, user_b)` with `user_a < user_b`.
- Realtime is one-way (server -> client). The in-memory hub is the only single-instance component besides the rate limiter.

### Legacy boilerplate code

`internal/features/{posts,comments,billing,files}` and migrations `001`-`013` come from the original generic boilerplate.
The packages are no longer routed or wired; the tables they created are inert. They can be deleted together with
`services/api/internal/features/billing` tests once the founder validates the removal.

## Mobile (`apps/mobile`)

- `src/api`: the only place that touches the network (`client.ts` handles bearer tokens, single-flight refresh, error mapping). UI never calls `fetch`.
- `src/hooks`: `useAuth` (session + bridge to the client), `useRealtime` (socket -> React Query cache), `useData` (shared queries with polling fallback).
- `src/screens` and `src/components`: feature UI; `src/shared`: design system (`ui`), layout, feedback, protected image component.
- Photos are private: `PhotoImage` fetches them with the bearer token.
- Theme: light/dark follow the system; colors are read through `useTheme()`.

## Data flow of the critical journey

1. Register -> `GET /me` reports `profileComplete=false` -> onboarding (profile, preferences, photos, location).
2. `GET /discover` returns ranked candidates; the client queues them and refills when low.
3. `POST /swipes` (like) -> if the other side already liked, a match + conversation are created in one transaction; both sides get a notification and a `match` event.
4. `POST /conversations/:id/messages` -> stored, pushed to the recipient over WebSocket, collapsed into one unread notification.
5. `POST /conversations/:id/read` -> read receipt pushed to the sender.
