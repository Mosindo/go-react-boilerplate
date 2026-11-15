# Architecture

## Overview

```text
 Expo app (iOS / Android / web)
   src/api ──REST (JSON, Bearer JWT)──►  Gin router (internal/app)
   src/api/realtime ──WebSocket /ws──►   realtime.Hub (in-process)
                                          │
                     features/* handler → service → repository
                                          │
                                   PostgreSQL (pgxpool)      Local disk / volume (photos)
```

A single Go process serves REST, WebSocket and signed media. Every feature follows
`handler.go → service.go → repository.go` with `model.go` and `routes.go`:

- **handlers** bind/validate transport input and map typed `apperr` errors to HTTP responses;
- **services** hold business rules (age gate, mutual preferences, match creation, permissions);
- **repositories** own all SQL (no ORM) and return plain structs.

Shared concerns live in `internal/platform`: configuration, migrations, auth middleware,
CORS/security headers, request logging with secret redaction, rate limiting, the realtime hub,
storage with HMAC-signed URLs, the mailer and text normalisation.

## Data model

| Table | Purpose | Key constraints / indexes |
| --- | --- | --- |
| `users` | account (email, bcrypt hash, `is_demo`, activity) | unique `email` (normalised check) |
| `sessions` | refresh sessions (SHA-256 of token) | unique `token_hash`, `(user_id, created_at)` |
| `password_reset_codes` | 6-digit codes (hashed), attempts, expiry | `(user_id, created_at)` |
| `profiles` | public profile + rounded coordinates + privacy flags | checks on gender, lengths, height, lat/lng; partial discovery index |
| `preferences` | genders sought, age range, max distance | checks (18–99, 1–500 km, allowed genders) |
| `interests` / `user_interests` | catalogue / selection | PK `(user_id, interest_id)`, reverse index |
| `photos` | ordered photos (≤ 6) | unique `storage_key`, deferred unique `(user_id, position)` |
| `swipes` | like/pass, one per pair direction | PK `(swiper_id, target_id)`, no self-swipe, `(target_id, action, created_at)` |
| `matches` | mutual likes | `user_a_id < user_b_id`, unique pair |
| `conversations` | one per match | unique `match_id` (cascade) |
| `conversation_participants` | membership, `last_read_at`, `cleared_at` | PK `(conversation_id, user_id)`, `(user_id, conversation_id)` |
| `messages` | text messages (1–2000 chars) | `(conversation_id, created_at DESC, id DESC)` |
| `blocks` | directional blocks | PK `(blocker_id, blocked_id)`, reverse index |
| `reports` | moderation reports (kept, anonymised on deletion) | reason/status checks, `(status, created_at)` |
| `notifications` | match / message / like | `(user_id, created_at)`, partial unread index, unique unread `group_key` |

Deleting a user cascades to every personal row; `reports` keep the row with nulled references.

## Key flows

### Discovery

`GET /discovery` refuses incomplete profiles, then one SQL query builds a candidate pool (max 100):
complete, not paused, not already swiped, not blocked either way, gender/age/distance preferences
satisfied **in both directions** (bounding-box pre-filter + haversine `distance_km()`), with ranking
features (shared interests, liked-me, activity, photo count). `matching.Ranker` orders the pool and
`profiles.Service.LoadCards` hydrates the page with exactly three queries (profiles, photos, interests).

### Swipe and match

`POST /swipes` runs in a transaction holding `pg_advisory_xact_lock(hash(pair))`: eligibility check,
`INSERT … ON CONFLICT DO NOTHING` on the swipe (a repeat returns `409 already_swiped`), then, for a like,
a reciprocal-like check that inserts the match, its conversation and both participants. Concurrent mutual
likes therefore always produce exactly one match (covered by an integration test). Notifications and
realtime events are emitted after commit.

### Chat

Every chat query is gated by `conversation_participants` membership for the caller; outsiders get 404.
Messages are keyset-paginated newest first. Sending updates `last_message_at`, the sender's
`last_read_at`, pushes `message.new` to both sides and upserts a grouped notification. Reading sets
`last_read_at` and pushes `conversation.read` (read receipts). "Delete conversation" sets `cleared_at`
for the caller only. Unmatching, blocking or reporting deletes the match (cascade).

### Realtime

`GET /ws` upgrades, expects `{"type":"auth","token":…}` as the first frame (10 s), validates the JWT and
its session, and registers the connection in the hub. Slow consumers drop events rather than block
publishers; clients re-sync over REST after reconnecting. The mobile client reconnects with exponential
backoff, refreshes the token on close code 4001 and falls back to polling while disconnected.

### Photos

Multipart upload (≤ 10 MiB) → content sniffing (JPEG/PNG/WebP only) → `DecodeConfig` bounds
(≥ 200 px, ≤ 10 000 px, ≤ 40 MP) → decode → downscale (≤ 1600 px) → JPEG re-encode (drops all metadata)
→ random key `[a-f0-9]{32}.jpg` → storage. Media are served only through `/media/:key?exp&sig` with an
HMAC signature aligned on 6-hour windows (cacheable, short-lived).

## Security summary

- JWT HS256 only (`SigningMethodHMAC`, `alg` pinned, `exp` required, `uid`/`sid` claims required) and an
  active-session check on every request; refresh tokens rotate and are stored hashed.
- bcrypt passwords; constant-time behaviour on unknown e-mails; 8+ chars with letter and digit.
- Input normalisation (`textx.Clean` strips control and bidi characters) and strict validation on every
  field; pagination limits clamped; UUID path parameters validated.
- Token-bucket rate limits: credentials 10/min/IP, recovery e-mails 5/h/IP, uploads 30/h, swipes
  120/min, messages 40/min, reports 10/h (per user). Likes are not artificially limited for normal use.
- `TRUSTED_PROXIES` controls which `X-Forwarded-For` headers are honoured.
- Security headers, strict CORS allow-list, `Cache-Control: no-store` on private responses, secrets
  redacted from logs.
