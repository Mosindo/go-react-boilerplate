# Lueur — Architecture

## Overview

```text
Expo app (iOS / Android / web)
  └─ src/api/client.ts ── HTTPS/JSON ──► Gin API (single http.Server)
  └─ src/api/realtime.ts ── SSE (/events) ─┘     ├─ features/* (handler → service → repository)
                                                 ├─ platform/* (config, db, middleware, media, mailer, realtime)
                                                 └─ PostgreSQL (single pgxpool) + UPLOAD_DIR
```

## Backend

### Features

| Feature | Responsibility |
|---|---|
| `auth` | Register/login, refresh-token rotation, logout, password policy, recovery codes, password change, account deletion, session validation for the middleware |
| `profiles` | Own profile, preferences, approximate location, interests catalog, public cards, shared visibility SQL (`sql.go`) |
| `photos` | Upload/replace/delete/reorder, signed media serving |
| `matching` | Discovery pool + `Ranker`, swipes, matches, likes received, unmatch, pass reset |
| `chat` | Match-gated conversations, messages, read state, local clear |
| `safety` | Blocks and reports (+ auto review) |
| `notifications` | In-app notifications, `Notify` used by other features, real-time push |
| `users` | Matched contacts (no e-mail, no directory) |
| `posts`, `comments`, `files` | Generic boilerplate modules, unchanged, not used by the app |

Cross-feature dependencies only point "down": `matching`, `safety`, `chat` → `profiles`; `safety` → `matching.DeleteMatchTx`. Composition happens in `cmd/api/main.go`.

### Platform

- `config`: environment loading and validation (rejects weak `JWT_SECRET`, requires SMTP in production).
- `db`: pool + embedded migrations tracked in `schema_migrations` under an advisory lock.
- `middleware`: auth (JWT HS256 + live session check), CORS, security headers, request id/log/metrics, token-bucket rate limiting, body limits (1 MiB JSON, ~10 MiB multipart).
- `httpx`: error envelope `{error, code}`, UUID and pagination parsing.
- `media`: `Storage` interface (local disk implementation), image sanitisation, URL signer.
- `mailer`: SMTP, log (development) and in-memory (tests) mailers.
- `realtime`: per-user in-process hub implementing `Publisher`, SSE handler.

## Data model

```text
users ─1:1─ profiles            (identity, rounded location, privacy, moderation_status)
      ─1:1─ preferences         (interested_in[], min/max age, max distance)
      ─1:N─ photos              (position 0..5, unique per user, deferrable)
      ─N:M─ interests via user_interests
      ─1:N─ swipes (swiper)     PK (swiper_id, target_id), action like|pass
matches (user_a_id < user_b_id, unique) ─1:1─ conversations ─1:N─ messages
                                               └─ conversation_participants (last_read_at, cleared_at)
blocks (blocker, blocked) · reports (reporter SET NULL, reported CASCADE, one open per pair)
notifications · sessions · password_reset_tokens
```

Every user-owned row cascades on account deletion; match conversations are deleted explicitly in the same transaction and photo files are removed afterwards.

## Key flows

**Discovery** — one SQL query builds an eligible pool (≤ 200) for the viewer: same workspace, discoverable, active moderation, complete (name, birthdate, gender, preferences, location, ≥ 1 photo), genders and age ranges mutually compatible, within both members' max distance (bounding box on an index, then haversine `geo_distance_km`), never swiped by the viewer, no block in either direction. The `Ranker` orders the pool; only the requested page is turned into cards (3 batched queries: rows, photos, interests).

**Swipe / match** — in one transaction: per-pair `pg_advisory_xact_lock`, swiper completeness check, target eligibility check (mutual genders or target already liked the swiper), `INSERT … ON CONFLICT DO NOTHING` for idempotency (changing an answer ⇒ 409), reciprocal-like check, `INSERT` match + conversation + participants. After commit: notifications and real-time events.

**Chat access** — every chat call resolves the conversation through the `matches` row of the two members; no match ⇒ 404. Unmatch and block delete the match and conversation.

**Real time** — `GET /events` streams `message.created`, `conversation.read`, `match.created`, `match.removed`, `like.received`, `notification.created`. Streams close after 15 minutes (the access-token lifetime); the client reconnects with a fresh token and revalidates its caches on reconnect and app foreground, so missed events are recovered.

## Security model

- Passwords: bcrypt; policy ≥ 8 chars with a letter and a digit; constant-time-ish login (dummy bcrypt for unknown e-mails).
- Tokens: 15-minute HS256 access tokens with mandatory `exp`, `uid`, `oid`, `sid`; 30-day refresh tokens stored hashed and rotated on use; sessions checked on each request.
- Recovery: 6-digit code, hashed with the user id, 15-minute TTL, 5 attempts, single use, revokes all sessions; same response for unknown e-mails.
- Authorization is always derived from the token's user id; ids in paths only name the *other* member and are checked against matches/blocks/visibility.
- Uploads: content sniffing (JPEG/PNG), dimension and pixel limits, re-encoding (metadata stripped), random server-side keys validated against a strict pattern.
- Privacy: no e-mail, birthdate or coordinates in public payloads; rounded coordinates at rest; rounded distances; optional hidden distance; profile pause.
- Abuse: rate limits per IP (auth, recovery) and per user (swipes, messages, reports, uploads); `TRUSTED_PROXIES` controls `X-Forwarded-For` trust.

## Mobile

- `api/client.ts` is the only place that performs HTTP: it attaches the token, refreshes it proactively (< 60 s left) or on a 401 with a single in-flight refresh, maps error codes to French copy, and reports expired sessions to the auth provider.
- Server state lives in TanStack Query (`hooks/queries.ts`); `hooks/useRealtime.ts` patches caches from SSE events (optimistic chat messages are reconciled by id).
- Navigation: auth stack → (onboarding if identity incomplete) → tabs (Découvrir, J'aime, Messages, Profil) + stack screens (chat thread, profile detail modal, edit profile, preferences, settings, blocked users, password, deletion, notifications).
- Images use `expo-image` with memory+disk caching and prefetch of the next cards; lists are virtualised (`FlatList`), chat history is paginated.
