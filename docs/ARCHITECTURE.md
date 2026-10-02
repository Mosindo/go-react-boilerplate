# Alba architecture

## Overview

```text
Expo app (iOS / Android / web) ── REST + WebSocket ──► Gin API ──► PostgreSQL
                                                         └──────► private photo storage (UPLOAD_DIR)
```

One Go process, one HTTP server, one shared `pgxpool`. Features are wired in `internal/app`.

## Backend rules

- `handler → service → repository`; handlers parse/validate HTTP and map errors, services hold business rules, repositories hold explicit SQL.
- Shared concerns live in `internal/platform` (config, db, middleware, httpx, ratelimit, realtime hub, storage, mailer, token). Domain code lives in `internal/features`.
- Cross-feature needs go through small interfaces (`CardSource`, `PhotoSource`, `realtime.Publisher`), never through another feature's repository.
- Every authorization decision is made server-side with the user id from the verified token; resources of others answer 404.
- List endpoints are paginated (cursor or `before` id) and batch-load related data (cards, photos, interests): no N+1.

## Authentication

Access token: JWT HS256, 15 min, claims `uid`, `sid`, `typ=access`; expiry mandatory, algorithm pinned. The middleware also checks the session is active (15 s cache; logout, password change and account deletion invalidate immediately on the instance). Refresh tokens are random, stored hashed, rotated on use. WebSocket tickets are separate 60 s tokens (`typ=ws`).

## Matching

`swipes(swiper, target, action)` is unique per ordered pair. A like checks the reverse like inside one transaction under a per-pair advisory lock; if present it inserts the match (`user_a < user_b`, unique) and both notifications. Discovery (`discovery/repository.go`) applies all hard rules in SQL, and ordering comes from `ranking.go`.

## Real time

`platform/realtime.Hub` maps user id → connections. Services publish `message`, `match`, `read`, `unmatched` events after committing. Slow clients are dropped (they resync by refetching). Single-instance by design; swap the hub for Postgres LISTEN/NOTIFY or Redis to scale out.

## Photos

Upload → size cap → content sniffing → full decode (pixel cap) → resize ≤ 1280 → JPEG re-encode → store under a random key. Clients receive `/photos/{id}/file?exp&sig` (HMAC of id+expiry, key derived from `JWT_SECRET`).

## Mobile

`api/` (typed client with single-flight refresh), `auth/` (secure session storage), `realtime/` (socket → TanStack Query cache), `navigation/` (auth / onboarding / main), `screens/`, `ui/` (design system: warm paper + terracotta, plum accent, dark mode), `lib/` (validation, formatting). UI code never calls `fetch` directly.
