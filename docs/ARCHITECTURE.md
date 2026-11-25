# Lueur — Architecture

Lueur is a free dating app built as a monorepo: a Go API (`services/api`) and an Expo/React Native client (`apps/mobile`, also exported to the web).

## 1. Backend

### 1.1 Layout

```text
services/api/
├─ cmd/api/main.go          single Gin HTTP server, graceful shutdown, header timeouts
├─ cmd/seed/main.go         demo data (refuses to run in production)
└─ internal/
   ├─ app/                  composition root (wires platform + features) and HTTP integration tests
   ├─ platform/             cross-cutting concerns, no domain knowledge
   │  ├─ config/            environment parsing and validation
   │  ├─ db/                pgxpool connection, embedded + tracked migrations
   │  ├─ errors/            typed application errors (package apperr)
   │  ├─ httpx/             shared handler helpers (errors, binding, ids, pagination)
   │  ├─ middleware/        auth, CORS, security headers, rate limiting, body limit, logs
   │  ├─ token/             strict HS256 access tokens
   │  ├─ urlsign/           HMAC-signed expiring URLs (photos)
   │  ├─ storage/           object storage interface + local disk implementation
   │  ├─ mailer/            SMTP / dev log / disabled mailers
   │  ├─ realtime/          WebSocket hub, single-use tickets, Publisher interface
   │  ├─ logger/, handlers/ structured logs, health endpoint
   └─ features/
      ├─ auth/              accounts, sessions, recovery, password change, deletion
      ├─ profiles/          profile, preferences, privacy, location, interests, public cards
      ├─ photos/            uploads, image sanitising, ordering, signed serving
      ├─ matching/          discovery query, swipes, matches, unmatch
      ├─ chat/              conversations, messages, read state, local clear
      ├─ notifications/     server-generated notifications
      └─ safety/            blocks and reports
```

Every feature contains `model.go`, `repository.go`, `service.go`, `handler.go`, `routes.go`:

- **handler**: binds/validates transport input, calls one service method, maps errors with `httpx.Fail`. No SQL, no business rules.
- **service**: business rules and authorisation decisions; returns typed `apperr.Error` values (status + stable code + message).
- **repository**: SQL only (pgx, no ORM). Multi-step writes use transactions; concurrency-sensitive writes use advisory locks.

Features depend on each other only through small interfaces defined by the consumer (e.g. `matching.ProfileService`, `chat.PhotoLister`, `matching.Notifier`), wired in `internal/app`.

### 1.2 Request pipeline

`Recovery → SecurityHeaders → CORS → RequestID → metrics/logging → MaxJSONBody(1 MiB) → route guards (RequireUser, rate limiters) → handler`.

`RequireUser` validates the JWT (alg HS256, issuer, audience, expiry, subject, session id) **and** checks the session row is still active, so revocation is immediate.

### 1.3 Data model

```text
users 1─1 profiles, 1─1 preferences, 1─n photos, n─n interests (user_interests)
users 1─n sessions, password_reset_tokens
swipes (swiper, target) unique ─► matches (user_a < user_b) unique ─► conversations 1─1 ─► messages
                                                                      └─► conversation_participants (last_read_at, cleared_at)
blocks (blocker, blocked) · reports · notifications
```

Deleting a user cascades to all personal rows; reports keep a nullable reference for moderation.

### 1.4 Key flows

- **Discovery** (`matching/repository.go`): one SQL query applies mutual filters (genders, age ranges, distance on both sides), excludes self, already-swiped profiles, profiles that passed on the viewer, blocks in both directions, profiles without photos and hidden profiles; ranks with `rankingOrder`. Photos and interests for the page are loaded with one query each (`profiles.Service.Hydrate`) — no N+1.
- **Swipe → match**: in a transaction, an advisory lock on the user pair serialises concurrent swipes; the swipe is inserted once; on a reciprocal like, the match (canonical order), its conversation and both participants are created. After commit, notifications and realtime events are published.
- **Chat**: membership is checked for every read/write (participant of an *active* match). Messages are paginated newest-first with a `(created_at, id)` keyset cursor. `last_read_at` provides unread counts and read receipts; `cleared_at` implements local deletion.
- **Realtime**: `POST /realtime/ticket` returns a 60-second single-use ticket; `GET /realtime?ticket=` upgrades to a WebSocket. The hub fans out events per user (bounded buffers, ping/pong, max 5 connections per user). Services depend only on `realtime.Publisher`.
- **Photos**: decode-config check (type, size, pixel count) → decode → EXIF orientation → downscale to 1440 px → JPEG re-encode (strips metadata) → atomic write under a random key → row insert in a locked transaction (max 6). URLs are `/photos/{id}?exp=&sig=` signed with a key derived from `JWT_SECRET`, aligned on hourly buckets for client caching.
- **Location**: coordinates are rounded to 0.01° before storage; distances are computed in SQL (`distance_km`) and further rounded for display.

### 1.5 Scaling notes

Single instance by design today (in-memory hub and rate limiter). Replace `realtime.Hub` with a Postgres `LISTEN/NOTIFY` or broker-backed `Publisher`, and the rate limiter with a shared store, to run several instances. `storage.Storage` can be backed by an object store.

## 2. Mobile client

```text
apps/mobile/src/
├─ api/          client.ts (single fetch wrapper: base URL, timeouts, JSON, error codes, transparent token refresh),
│                one module per domain (auth, profile, matching, chat, notifications, safety), realtime.ts (WebSocket client)
├─ hooks/        useSession (auth state), queryClient (React Query + keys), queries (data hooks), useRealtime (cache sync)
├─ navigation/   RootNavigator: booting → signed-out stack → onboarding (until profile complete) → app stack + tabs
├─ screens/      auth/, onboarding/, discover/, chat/, notifications/, profile/
├─ components/   domain components (PhotoGrid, ProfileFields, PreferencesFields, SafetyActions, Stepper, location, image picking)
├─ ui/           design system primitives (Text, Button, IconButton, TextField, Screen, Header, Chip, Avatar, RemoteImage, States, Toast, Dialog/Sheet, ListRow)
├─ theme/        colour, spacing, radius, typography tokens
├─ store/        token storage (SecureStore on native, memory on web)
└─ utils/        formatting, labels, validation
```

Rules:

- Screens never call `fetch`; they use hooks or `src/api/*` modules.
- Server state lives in React Query; realtime events update or invalidate cached queries (`applyRealtimeEvent`). When the socket is down, lists poll at a low rate.
- Profile mutations return the full `MeProfile`, written straight into the cache (`useSetMyProfile`).
- Images go through `RemoteImage` (expo-image, memory+disk cache, signed URLs).
- Every interactive element has an accessibility label; destructive actions use the cross-platform `Dialog`.

## 3. Quality gates

| Layer | Tooling |
| --- | --- |
| Go | gofmt, go vet, unit tests, HTTP integration tests against PostgreSQL (`DATABASE_URL_TEST`) |
| Mobile | `tsc --noEmit` (strict), ESLint (Expo config), Prettier, Jest |
| End-to-end | `scripts/e2e-smoke.js` (API), `e2e/web/critical-flow.js` (Playwright on the web build) |
| Delivery | Docker image build + compose boot + seed in CI |
