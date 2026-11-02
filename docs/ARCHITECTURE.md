# Architecture

## Overview

```text
Expo app (React Native / web)  ──REST──▶  Gin API  ──pgxpool──▶  PostgreSQL
              ▲                              │
              └────────── WebSocket ◀────────┘   (in-process realtime hub)
```

One HTTP server (Gin), one shared connection pool, no ORM. Every feature follows `handler → service → repository`; handlers only bind/validate and translate errors, services hold business rules, repositories own the SQL.

## Backend layout

```text
services/api/
├─ cmd/api          main + router wiring (newRouter is reused by the integration tests)
├─ cmd/seed         demo data, refuses to run in production
└─ internal/
   ├─ platform/
   │  ├─ config      env loading + validation (JWT secret strength, production requirements)
   │  ├─ db          pool, embedded migrations tracked in schema_migrations (advisory-locked), session checker
   │  ├─ middleware  auth (HS256 only), CORS, security headers, rate limiter, body limit, UUID param guard
   │  ├─ errors      client-safe error type + JSON envelope {"error","code"}
   │  ├─ realtime    hub, WebSocket handler, single-use tickets
   │  ├─ mail        SMTP / dev log mailer
   │  └─ logger      structured, secret-redacting logs
   └─ features/
      ├─ auth          register, login, refresh rotation, logout, password reset, delete account
      ├─ profiles      profile, preferences, interests, completeness
      ├─ photos        upload pipeline, ordering, authenticated serving
      ├─ discovery     deck, like/pass, match creation, public profile view
      ├─ chat          inbox, messages, read state, hide, unmatch
      ├─ notifications persisted + live notifications
      └─ safety        block, unblock, report
```

## Data model (migrations `001`–`004`)

| Table | Purpose / key constraints |
| --- | --- |
| `users` | unique lower-case e-mail (`CHECK email = lower(email)`), bcrypt hash |
| `sessions` | hashed refresh token (unique), revocable, expiring |
| `password_resets` | hashed single-use token, 1 h TTL |
| `profiles` | 1:1 with user; first name, birth date, gender, bio, city, rounded lat/lng (both or neither), `show_distance`, `discoverable` |
| `preferences` | `interested_in` ⊂ {woman, man, non_binary}, age range ≥ 18, distance 1–500 km |
| `interests`, `user_interests` | seeded catalogue + many-to-many |
| `photos` | `bytea` JPEG, `position` (0 = main), ≤ 6 per user (enforced under an advisory lock) |
| `swipes` | `PK (from_user_id, to_user_id)`, `action ∈ {like, pass}`; a decision is final |
| `matches` | `user_a_id < user_b_id` + `UNIQUE` pair → one match per pair; doubles as the conversation; per-user `hidden_at` for local deletion |
| `messages` | `match_id`, `sender_id`, 1–2000 chars, `read_at` |
| `blocks` | `PK (blocker_id, blocked_id)` |
| `reports` | reason enum, details ≤ 1000, status; `reported_id` is `SET NULL` on deletion so evidence survives |
| `notifications` | `match` / `message`, JSON payload, `read_at` |

All foreign keys from user-owned rows use `ON DELETE CASCADE`, so `DELETE FROM users` is the whole account-erasure implementation (verified by an integration test). Required indexes include `users(email)`, `messages(match_id, created_at DESC, id DESC)`, `notifications(user_id, created_at DESC)`, partial indexes for unread rows and discoverable profiles, and `swipes(to_user_id, created_at) WHERE action='like'`.

## Matching

1. `POST /discover/:id/like|pass` opens a transaction and takes `pg_advisory_xact_lock` on the **unordered pair**. Two people liking each other at the same instant are therefore serialised and exactly one match is created (tested with concurrent requests under `-race`).
2. If a swipe already exists, the call is idempotent when the action is identical and `409 already_decided` otherwise.
3. The target must satisfy the same predicate as the deck (`candidateSQL`): discoverable, has photo and location, mutual gender/age/distance preferences, no block in either direction. Anything else is a `404`, so ids cannot be probed.
4. A reciprocal like inserts `matches (LEAST, GREATEST)`; notifications and WebSocket events are sent **after** commit.

### Recommendation hook

`candidateSQL` in `features/discovery/repository.go` is the single place that orders candidates (already liked you → shared interests → distance → recent activity). To improve recommendations, change the `ORDER BY`/scoring there, or replace `Repository.Deck` with a precomputed candidate source; the API contract (`GET /discover`) does not change.

## Security model

- **Auth**: HS256 only (`SigningMethodHMAC` + `Alg()` check), mandatory expiry, claims `uid`/`sid`. Access tokens live 15 min; refresh tokens are random, stored hashed, rotated on every use. The middleware also checks that the **session is still active** on each request, so logout, password reset and account deletion take effect immediately.
- **Passwords**: bcrypt; 8–72 chars with a letter and a digit; login runs a dummy bcrypt when the e-mail is unknown (no timing oracle); `/auth/password/forgot` always answers 202.
- **Authorization**: every chat/photo/profile operation resolves membership or visibility in SQL; non-members get `404`, never `403`. Route `:id` parameters must be UUIDs.
- **Input**: server-side validation of every field (age ≥ 18 computed on the server, lengths, enums, coordinates, pagination limits); JSON bodies capped at 64 KB; photo uploads capped at 8 MB / 40 MP.
- **Uploads**: type is determined by decoding (JPEG/PNG/GIF), never by the client's header or extension; the image is downscaled to ≤ 1080 px and **re-encoded as JPEG**, which removes metadata and any embedded payload. Served with `nosniff`, `Content-Disposition: inline`, `Cache-Control: private`.
- **Abuse**: token-bucket rate limits (auth by IP, deck/likes/messages/uploads by user, global per-IP guard), 5 reports/day/user.
- **Privacy**: see README. Other users never receive e-mail, birth date, coordinates or exact distance.
- **Secrets**: only environment variables; the app contains none. Logs redact bearer tokens, JWTs and password/secret fields.

## Realtime

`POST /ws/ticket` (authenticated) returns a single-use ticket valid 30 s; `GET /ws?ticket=…` upgrades to a WebSocket. Events: `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`. Clients never send over the socket (REST is the write path), so permission checks live in one place. Slow consumers drop events and the app resynchronises with REST on reconnect.

Scaling out requires a shared bus (Postgres `LISTEN/NOTIFY` or Redis) feeding each instance's `Hub.Publish`; the in-memory rate limiter would also need a shared store.

## Mobile app

- `src/api` is the only code that touches the network; `client.ts` refreshes the session once on `401` (single-flight) and signs the user out when the refresh token is rejected.
- Tokens are stored in the OS keystore (`expo-secure-store`); on web they stay in memory.
- Authenticated images: native `Image` with an `Authorization` header; on web a blob URL fetched with the token and cached per photo id.
- State: TanStack Query for server data (conversations, notifications, profile), small React contexts for auth and realtime. Realtime events invalidate or patch the relevant caches.
- Design: warm "Tandem" palette with automatic dark mode, accessible labels/roles on interactive elements, loading/empty/error states on every list, toasts for action feedback.

## Product decisions made without asking

- Gender options: woman / man / non-binary; "interested in" is multi-select.
- The distance filter is mutual (both users' maximum distance and age ranges must accept each other).
- A decision is final: you cannot un-pass a profile in this version.
- A profile needs ≥ 1 photo and a location to be shown or to like.
- The last photo cannot be deleted (replace it instead).
- Hiding a conversation affects only the person who hid it and reverts when the other person writes; unmatching or blocking deletes the conversation for both.
- Message notifications are coalesced per match while unread.
- Moderation is report collection only (no admin console, no auto-suspension).
