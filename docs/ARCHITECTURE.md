# Architecture

## Overview

```text
Expo app ──REST (JSON)──────────────▶ Gin API ──pgxpool──▶ PostgreSQL
    ▲                                   │
    └────────WebSocket (server push)────┘      photos on disk (UPLOAD_DIR, served via authenticated endpoint)
```

One HTTP server (Gin), one shared connection pool, no ORM, explicit SQL. Features follow `handler -> service -> repository`:

- **handler** – binds/validates transport concerns, maps errors to HTTP statuses. No SQL, no business rules.
- **service** – business rules (validation, normalization, authorization decisions, orchestration).
- **repository** – SQL only; every query is parameterized.
- **model / routes** – DTOs and route registration (middleware such as rate limits is attached here).

Cross-feature wiring happens in `cmd/api/main.go` through small interfaces (`matching.Notifier`, `chat.Notifier`, `auth.DeletionHook`, `realtime.Publisher`), so features never import each other's internals.

## Backend packages

| Package | Responsibility |
| --- | --- |
| `features/auth` | register, login, refresh rotation, logout, `/me`, forgot/reset/change password, delete account |
| `features/profiles` | profile, preferences, privacy, interests, photo upload/replace/delete/reorder/serve |
| `features/matching` | discovery feed, swipes, matches, unmatch, public profile view |
| `features/chat` | conversations, messages, read state, local clear, `/ws` |
| `features/notifications` | list, mark read, creation on match/message |
| `features/safety` | block / unblock / list, reports |
| `platform/config` | environment parsing and validation |
| `platform/db` | pool, retrying connect, embedded migrations with tracking |
| `platform/middleware` | security headers, CORS, request id/logging, JWT (`ParseAccessToken`), rate limiting |
| `platform/httpx` | error payload, pagination limit, UUID path parameters |
| `platform/realtime` | in-memory WebSocket hub (`Publisher` interface for a future pub/sub backend) |
| `platform/storage`, `platform/imaging` | file store with strict keys; upload validation and re-encoding |
| `platform/mailer` | `Mailer` interface (log mailer for dev, discard in production until wired) |

## API

All endpoints except `/health`, `/auth/*` need `Authorization: Bearer <access token>`. Errors are `{"error": "message"}`. Lists use `limit` (≤ 50) and an opaque `cursor`.

| Method & path | Purpose |
| --- | --- |
| `GET /health` | liveness + DB ping |
| `POST /auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` | sessions (access 15 min, refresh 30 days, rotated) |
| `POST /auth/forgot-password` · `/auth/reset-password` | recovery by emailed code |
| `POST /auth/change-password` | requires current password, revokes sessions |
| `GET /me` · `DELETE /me` | identity · delete account (password required) |
| `GET/PUT /profile/me` | own profile (create or update) |
| `PUT /profile/preferences` · `/profile/privacy` | discovery preferences · discoverable / show distance |
| `GET /interests` | selectable interests |
| `POST /photos` · `PUT /photos/:id` · `DELETE /photos/:id` · `PUT /photos/order` · `GET /photos/:id` | photo management and authorized download |
| `GET /discover` | next candidates (no cursor needed: handled profiles drop out) |
| `POST /swipes` | `{targetId, action: like\|pass}` → `{matched, matchId, conversationId}` |
| `GET /matches` · `DELETE /matches/:id` | matches · unmatch |
| `GET /users/:id/profile` | public profile (discoverable and not blocked, or matched) |
| `GET /conversations` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` · `DELETE /conversations/:id` | chat |
| `GET /notifications` · `POST /notifications/read` · `POST /notifications/:id/read` | notifications |
| `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` | safety |
| `GET /ws` | WebSocket; first frame `{"type":"auth","token":"…"}`; events `ready`, `message`, `read`, `match` |

Status conventions: 400 invalid input, 401 unauthenticated, 403 wrong password, 404 not found **or not yours** (no existence leak), 409 conflict (duplicate swipe, photo limit), 413 payload too large, 429 rate limited.

## Data model

```text
users 1─1 profiles 1─1 preferences          user_interests ─ interests
users 1─* photos (position 0 = main)         users 1─* sessions, password_resets
users ─ swipes(swiper, target, like|pass) ─ users          PK (swiper_id, target_id)
users ─ matches(user_a < user_b) ─ users                    UNIQUE (user_a, user_b)
matches 1─1 conversations 1─* conversation_participants(user, last_read_message_id, cleared_at)
conversations 1─* messages(id bigserial, sender, body ≤ 2000)
users ─ blocks(blocker, blocked) ─ users        reports(reporter, reported, reason, status)
users 1─* notifications(type match|message, actor, ref = conversation id, read_at)
```

Notable constraints: gender / reason / status enums via `CHECK`, age and distance ranges, location pair consistency, `matches_ordered`, `photos (user_id, position)` unique (deferrable so reordering is atomic), one unread `message` notification per conversation (partial unique index).

Hot-path indexes: `swipes(target_id, swiper_id) WHERE like`, `messages(conversation_id, created_at, id)`, `notifications(user_id, created_at DESC)` and `(user_id) WHERE unread`, `conversation_participants(user_id, conversation_id)`, `matches(user_a|user_b, created_at DESC)`, `blocks(blocked_id, blocker_id)`, `users(email)` (unique). Interests and photos for a page of profiles are loaded with one batched query each (no N+1).

## Realtime

`chat.Service.Send` commits the message, then publishes `message` to both participants through `realtime.Publisher`; `MarkRead` publishes `read`; a mutual like publishes `match`. The hub never blocks (slow clients drop events and re-sync over REST, which the app does on reconnect). Delivery is at-most-once; REST is the source of truth.

## Mobile app

```text
src/api        fetch client (timeout, single-flight refresh on 401), typed endpoints, secure token store
src/state      session (auth lifecycle), realtime (WebSocket → React Query cache), query hooks, cache keys
src/navigation auth stack / onboarding gate / tabs + stack (typed params)
src/screens    feature screens; src/components shared feature widgets (swipe deck, forms, photo manager)
src/ui         design system (Text, Button, Input, Chip, states, Toast, AuthImage) with light/dark palettes
```

UI code never calls `fetch` directly: everything goes through `src/api`. Photos are loaded with an `Authorization` header via `AuthImage`. The swipe deck uses `Animated` + `PanResponder` (no extra native dependency) and mirrors every gesture with accessible buttons.
