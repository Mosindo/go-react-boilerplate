# Lueur architecture

This document describes how the repository is organised, how data flows, and
which rules every change must preserve. The product overview and setup guide
live in the [README](../README.md).

## 1. Repository layout

| Path | Content |
| --- | --- |
| `services/api` | Go API (`cmd/api`), demo seed (`cmd/seed`), features and platform packages |
| `apps/mobile` | Expo app for iOS, Android and web |
| `docker-compose.yml` | Postgres + API for local runs (`infra/docker-compose.yml` includes it) |
| `scripts/qa.sh`, `scripts/qa.ps1` | Full quality gate |
| `.github/workflows/ci.yml` | CI: API tests on Postgres, mobile checks, E2E, Docker health |

## 2. API

### 2.1 Layers

```
HTTP request
  → global middleware (recovery, security headers, CORS, request id, metrics, logging)
  → route middleware (RequireUser with session check, RateLimit)
  → handler   (bind + shape validation, error → HTTP mapping)
  → service   (business rules, normalisation, orchestration, events)
  → repository (SQL only, transactions, locks)
```

- One Gin engine, one HTTP server, one `pgxpool` shared by every repository.
- Handlers never contain SQL; repositories never know about HTTP.
- Features never import each other. Cross-feature needs are expressed as small
  interfaces owned by the consumer (`Notifier`, `URLSigner`, `MediaRemover`,
  `SessionChecker`, `realtime.Publisher`) and wired in `cmd/api/main.go`.
- Errors are returned as `{"error": "...", "code": "stable_code"}` via `httpx.Fail`.

### 2.2 Platform packages (`internal/platform`)

| Package | Responsibility |
| --- | --- |
| `config` | Environment parsing and validation (rejects weak/placeholder secrets) |
| `db` | Pool creation, tracked migrations (`schema_migrations` + advisory lock), `LockPair` |
| `middleware` | JWT HS256 validation with mandatory expiry and active-session check, rate limiting, CORS, security headers, logging |
| `httpx` | Error envelope, pagination parsing, UUID validation |
| `ratelimit` | In-memory token buckets with idle-bucket sweeping |
| `media` | Disk storage behind the `Storage` interface, upload sanitising, HMAC-signed URLs |
| `realtime` | WebSocket hub (per-user fan-out, ping/pong, slow-consumer eviction) and single-use tickets |
| `mailer` | SMTP delivery, development log mailer, disabled mailer |
| `people` | Age computation, coordinate rounding, displayed-distance bucketing, haversine |
| `logger`, `handlers` | Structured, redacted logs; `/health` |

### 2.3 Features (`internal/features`)

| Feature | Routes | Notes |
| --- | --- | --- |
| `auth` | `/auth/*`, `/me`, `/me/password` | bcrypt, refresh-token rotation, reset codes (HMAC-hashed, 15 min, 5 attempts), session revocation |
| `users` | `DELETE /me` | Account erasure: DB rows in one transaction, then photo files and sockets |
| `profiles` | `/me/profile`, `/me/preferences`, `/me/privacy`, `/me/location`, `/me/interests`, `/interests` | Validation (18+, immutable birthdate, names, text budgets), rounded coordinates |
| `photos` | `/me/photos*`, `/media/photos/:id` | Max 6, dense positions under an advisory lock, deferred unique constraint for reordering |
| `discovery` | `/discovery`, `/profiles/:userId` | Candidate query and public profile cards (batched photo/interest loading, no N+1) |
| `matching` | `/swipes`, `/matches*` | Swipe + reciprocal check + match + conversation in one transaction |
| `chat` | `/chats*` | Access only through an active, unblocked match; cursor pagination; read markers |
| `safety` | `/blocks*`, `/reports` | Block tears down match and conversation; report blocks and triggers moderation threshold |
| `notifications` | `/notifications*` | Server-generated only; message notifications coalesced while unread |
| `realtime` | `/realtime/ticket`, `/realtime/ws` | Ticket exchange and socket upgrade |

## 3. Data model

```
users ─┬─ sessions, password_reset_codes
       ├─ profiles (1:1)  ── coordinates rounded to 0.01°
       ├─ preferences (1:1)
       ├─ photos (≤ 6, unique (user_id, position) DEFERRABLE)
       ├─ user_interests ── interests (seeded catalogue)
       ├─ swipes (PK actor,target · decision like|pass)
       ├─ matches (user_low < user_high · unique pair · 1 conversation)
       ├─ conversation_participants ── conversations ── messages
       ├─ blocks (PK blocker,blocked)
       ├─ reports (reporter/reported ON DELETE SET NULL · one open report per pair)
       └─ notifications (JSONB data)
```

Every foreign key to `users` cascades on delete except `reports` (kept for
moderation history, anonymised). Indexes cover every hot path:
`users(email)`, `profiles(gender, birthdate) WHERE discoverable`,
`profiles(latitude, longitude)`, `swipes(target, actor) WHERE like`,
`matches(user_low|high, created_at)`, `messages(conversation_id, created_at)`,
`messages(conversation_id, sender, created_at)`,
`conversation_participants(conversation_id, user_id)`,
`notifications(user_id, created_at)`, `blocks(blocked, blocker)`,
`reports(reported_user_id, created_at)`, `user_interests(interest_id, user_id)`.

Migrations `001`–`013` are the historical boilerplate schema; `014` drops the
removed modules and `015` creates the dating schema. Add new numbered files
only; never edit an applied migration.

## 4. Key flows

### 4.1 Discovery

`discovery.Candidates` builds the viewer row (`me`), joins candidate profiles
and applies hard filters, all **mutual**:

1. same organisation, not self, discoverable, not hidden by moderation, at least one photo;
2. candidate gender ∈ viewer preferences **and** viewer gender ∈ candidate preferences;
3. candidate age within viewer range **and** viewer age within candidate range;
4. if the viewer shared a location: candidate must have one and be within
   `min(viewer radius, candidate radius)` (bounding box prefilter, then haversine);
5. never swiped by the viewer; no block in either direction.

Ranking score: `1.5 × liked_viewer + recency(last_active) + proximity + 0.2 × min(shared_interests, 5)`,
ties broken by user id. The feed returns at most 20 cards; swiped profiles
disappear from the next page automatically, so no cursor is needed.

### 4.2 Swipe and match

```
BEGIN
  pg_advisory_xact_lock(hash(pair))          -- serialises the pair
  check actor profile + target eligibility    -- same org, visible, no block
  INSERT swipe ON CONFLICT DO NOTHING         -- 0 rows → 409 already_swiped
  IF like AND reciprocal like:
     upsert direct conversation + participants
     INSERT match ON CONFLICT DO NOTHING
COMMIT
→ publish match.created + notification to both members
```

Unmatch and block take the same pair lock, delete the match and its
conversation (messages cascade) and emit `match.removed`.

### 4.3 Chat and realtime

- Every chat call starts with `Access(user, other)`: an existing match row,
  both participants, and no block. Anything else is a `404`.
- Messages are trimmed, stripped of control characters, ≤ 2000 runes.
- History uses a `(created_at, id)` cursor (`?before=<messageId>`).
- Read state: `conversation_participants.last_read_at`; the API returns the
  other member's marker so the client can show "Vu".
- After commit, the service publishes `message.created` to both members and a
  coalesced `message.received` notification to the recipient.
- Clients obtain a 30-second single-use ticket (`POST /realtime/ticket`) and
  open `GET /realtime/ws?ticket=…`. The socket is server-push only. Events:
  `connection.ready`, `message.created`, `conversation.read`, `match.created`,
  `match.removed`, `notification.created`.

### 4.4 Photos

Upload → `MaxBytesReader` (10 MB) → content sniffing (JPEG/PNG/WebP) →
`DecodeConfig` bounds (≥ 200 px, ≤ 12 000 px per side, ≤ 50 MP) → decode →
Catmull-Rom downscale to ≤ 1600 px → JPEG q85 (drops EXIF/GPS) → random key →
atomic write → row insert under a per-user advisory lock. URLs are
`/media/photos/:id?exp=…&sig=…` with an HMAC-SHA256 signature, valid until the
end of the next hour (stable for caching).

## 5. Mobile app

- `App.tsx` wires `SafeAreaProvider → ThemeProvider → QueryClientProvider → FeedbackProvider → AuthProvider → RootNavigator`.
- **API access** only through `src/api` (`api()` client + typed `endpoints.ts`).
  The client attaches the access token, performs a single-flight refresh on
  `401`, retries once and signals expiry to the auth provider.
- **State**: server state in TanStack Query (`src/hooks/queries.ts`, keys in
  `queryClient.ts`); mutations patch caches directly when the API returns the
  new state. Session state in `useAuth`.
- **Realtime**: `RealtimeProvider` keeps one socket while the app is in the
  foreground, folds events into the query cache, reconnects with exponential
  backoff and enables polling while disconnected.
- **Navigation**: auth stack → onboarding (until the profile is complete and
  the optional steps are done) → tabs (Découvrir, Matchs, Activité, Profil) +
  stacked screens (profile detail, chat, settings).
- **Design system**: `src/theme` (light/dark palettes, typography, spacing) and
  `src/ui` primitives; `makeStyles` memoises styles per palette.
  Cross-platform dialogs, toasts and sheets live in `src/feedback` (React
  Native's `Alert` buttons do not work on web).

## 6. Security checklist

- JWT: HS256 only, signing method type checked, expiry mandatory, `uid/oid/sid` claims required, session must be active.
- Passwords: bcrypt, 8–72 bytes with a letter and a digit, timing-equalised login.
- Inputs: e-mails trimmed and lower-cased, strict validation in services, pagination clamped.
- Privacy: no e-mail, birthdate or coordinates in any response about another member.
- Abuse: per-IP and per-user rate limits; set `TRUSTED_PROXIES` behind a proxy.
- Files: content validation, re-encoding, random names, signed expiring URLs, path validation.
- Secrets: only from the environment; the mobile app never holds secrets.

## 7. Scaling notes

The realtime hub and rate limiter are in-memory: run a single API instance, or
replace them behind their interfaces (Postgres `LISTEN/NOTIFY`/Redis pub-sub,
shared rate-limit store) before scaling horizontally. Disk media storage can be
swapped for object storage by implementing `media.Storage`. For very large
member bases, move distance filtering to PostGIS (`geography` + GiST index).
