# Architecture

See the README for the overview and the list of technical decisions. This document describes how a request flows and how the data is organised.

## Request flow

```text
Expo app ──HTTPS──▶ Gin router ──▶ middleware ──▶ handler ──▶ service ──▶ repository ──▶ PostgreSQL
    │                               │ security headers, CORS, request id, logging
    │                               │ rate limit (IP for auth, user for swipes / messages / uploads / reports)
    │                               └ RequireUser: strict HS256 JWT + live-session check on every request
    └──WebSocket /ws (first frame = access token) ◀── realtime.Hub ◀── services (match.new, message.new, …)
```

- Handlers only bind/validate transport concerns and map errors to HTTP statuses.
- Services own every business rule (age gate, onboarding, blocks, visibility, conversation membership).
- Repositories contain all SQL (explicit, parameterised, no ORM). List queries use `ANY($1)` batching or `LATERAL` joins — no N+1.
- Services depend on small interfaces (`Profiles`, `Cards`, `realtime.Publisher`) so features stay decoupled.

## Data model

```text
users ─┬─ sessions                    refresh tokens (hashed), rotation, revocation
       ├─ password_reset_tokens       single use, 1 h, hashed
       ├─ profiles (1:1)              first_name, birth_date, gender, bio, rounded lat/lon, privacy flags
       ├─ preferences (1:1)           interested_in[], age range, max distance
       ├─ user_interests ─ interests  reference list, seeded by migration
       ├─ photos                      position (deferrable unique), position 0 = main photo
       ├─ swipes                      (swiper, target) → like | pass, primary key prevents duplicates
       ├─ matches ─ match_members     canonical pair (user_a < user_b); per-member read marker + local clear
       │          └ chat_messages
       ├─ blocks                      (blocker, blocked); blocking deletes the match
       ├─ reports                     reason, details, evidence snapshot (last 20 messages), status
       └─ user_notifications          match | message; at most one unread message notification per match
```

Every table referencing `users` uses `ON DELETE CASCADE`: deleting an account erases all of its data. Photo files are erased by the `photos` service through the account-deletion hook.

## Discovery query

One SQL statement (`discovery/repository.go`) joins the viewer with candidates and keeps those where both sides' preferences match (gender, age, distance), who have at least one photo, are visible and enabled, and are neither swiped on nor blocked in either direction. Ranking lives in its `ORDER BY` and is the extension point for a smarter recommender. Cards for the resulting ids are then loaded in four batched queries (profiles, interests, photos, distance).

## Realtime

Events are JSON envelopes `{ "type": …, "data": … }`: `ready`, `match.new`, `match.removed`, `message.new`, `message.read`, `notification.new`. The client patches its TanStack Query cache from them and fully invalidates on reconnection. REST remains the source of truth, so a missed event only delays an update.

## Mobile structure

`src/api` (client with single-flight token refresh, typed endpoints) → `src/hooks` (auth, realtime, queries) → `src/navigation` (auth / onboarding / main tabs + stack) → `src/screens` → `src/components` and `src/ui` (design system, light/dark theme in `src/theme`). UI code never calls `fetch` directly.
