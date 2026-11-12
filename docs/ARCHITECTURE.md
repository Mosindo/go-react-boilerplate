# Lueur — Architecture

Lueur is a free dating app built on the `go-react-saas` boilerplate: a Go/Gin/PostgreSQL API (`services/api`) and an Expo/React Native client (`apps/mobile`).

## 1. Backend

```text
services/api/
├─ cmd/api/          composition root (main.go) + HTTP integration tests
├─ cmd/seed/         demo data loader (refuses APP_ENV=production)
└─ internal/
   ├─ platform/      cross-cutting concerns, no business rules
   │  ├─ config/     env parsing + validation (JWT secret strength, APP_ENV, SMTP)
   │  ├─ db/         pgxpool connection, embedded + tracked migrations (advisory-locked)
   │  ├─ middleware/ auth (HS256 verifier + session check), CORS, security headers, logging
   │  ├─ httpx/      pagination parsing, UUID validation, error envelope
   │  ├─ ratelimit/  sliding-window limiter + Gin middleware (by IP / by user)
   │  ├─ media/      image sanitization, local storage, HMAC-signed expiring URLs
   │  ├─ realtime/   per-user WebSocket hub (in-band auth)
   │  ├─ mailer/     SMTP / dev log / disabled
   │  └─ logger/     structured logs with secret redaction
   └─ features/      one package per domain: handler → service → repository (+ model, routes)
      ├─ auth  profiles  photos  matching  chat  safety  notifications
      └─ users posts comments files   (generic boilerplate modules)
```

Rules: handlers only bind/validate transport input and map errors to status codes; services hold business rules; repositories own SQL. Features depend on each other only through small interfaces declared by the consumer (e.g. `matching.ProfileProvider`, `chat.Notifier`), wired in `cmd/api/main.go`.

### Data model

| Table | Purpose | Key constraints / indexes |
| --- | --- | --- |
| `users` | credentials (bcrypt), tenant | unique `email` |
| `sessions` | rotating refresh tokens (SHA-256 hashed) | unique `token_hash`, partial index on active |
| `profiles` | 1:1 user profile, rounded coordinates, privacy flags | checks on gender/goal/lengths, partial index `(gender, birthdate) WHERE discoverable` |
| `preferences` | interested_in[], age range, max distance | `min_age >= 18`, `min_age <= max_age` |
| `interests`, `user_interests` | curated catalogue (reference data) and selection | PK `(user_id, interest_id)` |
| `photos` | ≤ 6 per user, position 0 = main | unique `(user_id, position)` **deferrable** (reordering in one tx) |
| `swipes` | likes and passes | PK `(actor, target)` → one decision per pair |
| `matches` | one row per pair | `user_low_id < user_high_id`, unique pair |
| `conversations`, `conversation_participants`, `messages` | chat; `last_read_at` = read cursor, `hidden_at` = local delete | `messages (conversation_id, created_at)` |
| `blocks` | symmetric invisibility | PK `(blocker, blocked)` + reverse index |
| `reports` | moderation queue, kept anonymized after deletion | one open report per pair |
| `notifications` | in-app notifications (`match`, `message`) | `(user_id, created_at)`, unread partial indexes |
| `password_resets` | HMAC-hashed 6-digit codes, attempts counter | `(user_id, created_at)` |

All user-owned rows cascade on account deletion; conversations are deleted explicitly; reports switch to `NULL` ids.

### Key flows

- **Discovery** (`matching/repository.go#Discover`): mutual filters (each side's gender, age and distance preferences must accept the other), excludes swiped/blocked/non-discoverable/photo-less profiles, bounding box + haversine distance, ranked by "already liked you", shared interests and recency. Cards are hydrated in three batched queries (profiles, interests, photos).
- **Swipe → match** (`matching/repository.go#Swipe`): in one transaction, take `pg_advisory_xact_lock(pair)`, check availability (tenant, blocks), insert the swipe (`ON CONFLICT DO NOTHING` → 409 if repeated), and on reciprocal like create the conversation, participants and match. Both members get a notification and a `match.new` event.
- **Chat**: every chat call resolves the conversation through `matches` (`ConversationForPair`); no match → 404. Messages page backwards with a keyset cursor (`before=<message id>`). Sending pushes `message.new` to both members and refreshes one unread `message` notification per conversation.
- **Unmatch / block / report** delete the match and the conversation for both sides and turn the actor's like into a pass. Reporting always blocks.
- **Photos**: bytes are sniffed, dimension-checked before decode, EXIF-rotated, resized to ≤ 1440 px and re-encoded to JPEG (metadata stripped), stored under a random key, served via `/media/:key?exp&sig`.
- **Sessions**: access JWT (15 min) carries `uid`, `oid`, `sid`; the middleware also checks the session row, so logout, password reset and account deletion revoke access immediately.

### Scaling notes

Single-process components with an interface ready for a shared backend: rate limiter (`ratelimit.Limiter`), realtime fan-out (`realtime.Publisher` — e.g. Postgres `LISTEN/NOTIFY`), file storage (`media.Storage` — e.g. S3). Discovery scoring is computed per query (fine for a city-scale community); PostGIS/earthdistance or a precomputed candidate table are the next steps for large volumes.

## 2. Mobile client

```text
apps/mobile/src/
├─ api/          client.ts (only network entry point: token injection, single-flight refresh on 401,
│                friendly errors), endpoints.ts (typed calls per domain), types.ts
├─ hooks/        useAuth (session lifecycle), queries.ts (React Query keys + hooks),
│                useRealtime (WebSocket → cache updates), realtimeStatus (polling fallback)
├─ navigation/   RootNavigator: Auth stack → Onboarding (until profile complete) → App stack + tabs
├─ screens/      auth/, onboarding/, discover/ (deck, card, match modal), matches/ (list, chat),
│                profile/ (public view), me/ (settings & edit screens), notifications/
├─ components/   PhotoGrid, PreferencesForm, InterestPicker, SafetySheet, location helper
├─ ui/ theme/    design system: palette (light/dark), typography, Button, TextField, Chip, Photo…
└─ store/        tokenStore (SecureStore on device)
```

- Server state lives in React Query; realtime events update or invalidate cached queries; polling only runs while the socket is disconnected.
- UI code never calls `fetch` directly: everything goes through `src/api`.
- Images use `expo-image` with a cache key based on the storage key, so hourly-rotating signed URLs keep hitting the disk cache.
- Discovery deck prefetches the next page when ≤ 3 cards remain, excluding cards already in the deck.
