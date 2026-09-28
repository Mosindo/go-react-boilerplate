# API contract

Single source of truth shared by `services/api` and `apps/mobile`. JSON only, camelCase keys, timestamps RFC 3339 UTC, ids are UUID strings.

## Conventions

- Auth: `Authorization: Bearer <accessToken>` (JWT HS256, 15 min). Refresh tokens are opaque, rotated on each refresh (30 days).
- Errors: HTTP status + `{"error": "<human readable message>", "code": "<machine_code>"}`. Codes used: `invalid_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `profile_incomplete`, `underage`, `blocked`, `payload_too_large`, `unsupported_media`, `internal`.
- Pagination: lists take `limit` (default 20, max 50) and either `cursor` (opaque string, from the previous `nextCursor`) or nothing. Responses: `{ "items": [...], "nextCursor": "..." | null }` unless noted.
- Rate limits return 429 + `Retry-After`.
- Everything below requires auth unless marked **public**.

## Enums

- gender: `man` | `woman` | `non_binary`
- report reason: `spam` | `fake_profile` | `harassment` | `inappropriate_content` | `underage` | `other`
- notification type: `match` | `message`

## Auth

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/auth/register` **public** | `{email, password}` (password 8..128) | 201 `AuthResponse` |
| POST | `/auth/login` **public** | `{email, password}` | 200 `AuthResponse` |
| POST | `/auth/refresh` **public** | `{refreshToken}` | 200 `AuthResponse` |
| POST | `/auth/logout` **public** | `{refreshToken}` | 204 |
| POST | `/auth/forgot-password` **public** | `{email}` | 204 always (no account enumeration) |
| POST | `/auth/reset-password` **public** | `{email, code, newPassword}` | 204; revokes all sessions. 400 on bad/expired code |
| POST | `/me/password` | `{currentPassword, newPassword}` | 204; revokes other sessions |
| DELETE | `/me` | `{password}` | 204; hard-deletes the account and all data (cascade) |
| GET | `/me` | | `Me` |

`AuthResponse = { accessToken, refreshToken, user: { id, email, createdAt } }`

`Me = { id, email, createdAt, profile: Profile | null, preferences: Preferences, profileComplete: boolean }`

## Profile (own)

`Profile = { userId, firstName, age, birthDate (YYYY-MM-DD, own profile only), gender, bio, locationLabel, hasLocation, showDistance, isDiscoverable, interests: Interest[], photos: Photo[] }`

`Interest = { id, slug, label }`  `Photo = { id, position, url }` (`url` is a path like `/photos/<id>/content`, requires auth header)

| Method | Path | Body | Response |
|---|---|---|---|
| PUT | `/me/profile` | `{firstName (1..50), birthDate, gender, bio (0..500), interestIds: number[] (0..10), showDistance?, isDiscoverable?}` | 200 `Profile`. `birthDate` must be >= 18 years ago (`underage` 422) and is immutable once set (changing it => 422 `invalid_request`) |
| PUT | `/me/location` | `{latitude, longitude, label?}` | 200 `Profile`. Server rounds coordinates to 2 decimals (~1 km) before storing |
| GET | `/me/preferences` | | `Preferences` |
| PUT | `/me/preferences` | `{interestedIn: gender[] (1..3), minAge (18..99), maxAge (minAge..99), maxDistanceKm (1..500)}` | 200 `Preferences` |
| GET | `/interests` | | `{items: Interest[]}` (full fixed catalogue, ~30 entries, seeded by migration) |

`Preferences = { interestedIn: gender[], minAge, maxAge, maxDistanceKm }`; defaults for a new user: all genders, 18..99, 50 km.

`profileComplete` = profile row exists + location set + at least 1 photo.

## Photos

| Method | Path | Notes |
|---|---|---|
| POST | `/me/photos` | multipart field `file`. jpeg/png only, <= 8 MB, sniffed by content (never trust the header), decoded and re-encoded to JPEG (EXIF/GPS stripped, longest side <= 1280). Max 6 photos. Appended at the end. 201 `Photo` |
| PUT | `/me/photos/order` | `{photoIds: string[]}` must be a permutation of own photos; first = main. 200 `{items: Photo[]}` |
| DELETE | `/me/photos/:id` | own only; positions are compacted. 204 |
| PUT | `/me/photos/:id` | multipart `file`: replaces the image in place, keeps position. 200 `Photo` |
| GET | `/photos/:id/content` | Owner, or a viewer who is not blocked either way and either the owner has a discoverable profile or the two users are matched (so a match keeps seeing photos after the other side hides their profile). Returns `image/jpeg`, `Cache-Control: private, max-age=86400`, `X-Content-Type-Options: nosniff`. 404 otherwise |

## Discovery & swipes

`Candidate = { userId, firstName, age, bio, distanceKm: number | null (bucketed to 5 km steps, min 5), locationLabel, interests: Interest[], sharedInterestCount, photos: Photo[] }`

| Method | Path | Notes |
|---|---|---|
| GET | `/discover?limit=10` | `limit` defaults to 10 (max 50). Returns `{items: Candidate[]}`. Excludes: self, already swiped, blocked either way, non-discoverable, incomplete profiles, mismatching gender/age/distance preferences (reciprocal: they must also fit the viewer). 409 `profile_incomplete` if viewer's `profileComplete` is false. Pure queue: swiped profiles disappear, so no cursor. Ranking is behind a `Ranker` interface (default: shared interests, distance, recent activity) |
| GET | `/users/:id/profile` | Full `Candidate` of a user the viewer may see (discoverable or matched), 404 otherwise or if blocked |
| POST | `/swipes` | `{targetUserId, action: "like" \| "pass"}` -> 200 `{matched: boolean, match?: MatchSummary}`. Idempotent per (viewer,target): a repeated swipe returns the stored result and never creates duplicates. Cannot swipe self / blocked / unknown users. Rate limited (120/min) but no artificial like quota |
| DELETE | `/swipes/last` | Not implemented (no undo): always 501 |

Mutual likes create exactly one `match` (+ conversation) in a single transaction, and one `match` notification for each user.

## Matches & chat

`MatchSummary = { matchId, conversationId, createdAt, user: { userId, firstName, age, photo: Photo | null } , lastMessage: { body, senderId, createdAt } | null, unreadCount }`

| Method | Path | Notes |
|---|---|---|
| GET | `/matches?cursor&limit` | Newest activity first (`lastMessageAt` else match date). Excludes blocked pairs. Returns `{items: MatchSummary[], nextCursor}` |
| DELETE | `/matches/:matchId` | Unmatch: deletes match, conversation and messages for both. 204 |
| GET | `/conversations/:id/messages?before=<cursor>&limit=30` | Only participants (403/404 otherwise). Newest first. `{items: Message[], nextCursor}` |
| POST | `/conversations/:id/messages` | `{body (1..2000, trimmed)}` -> 201 `Message`. 403 `blocked` if either side blocked the other. Rate limited (30/min) |
| POST | `/conversations/:id/read` | Marks all incoming messages read. 204 |

`Message = { id, conversationId, senderId, body, createdAt, readAt | null }`

## Safety

| Method | Path | Body |
|---|---|---|
| POST | `/blocks` | `{userId}` -> 204 (idempotent). Hides the pair everywhere (discover, matches, chat, photos) |
| DELETE | `/blocks/:userId` | 204 |
| GET | `/blocks` | `{items: [{userId, firstName, blockedAt}]}` |
| POST | `/reports` | `{userId, reason, details? (<=1000)}` -> 201 `{id}`. Rate limited (10/hour) |

## Notifications

`Notification = { id, type, title, body, data: {matchId?, conversationId?, userId?}, readAt | null, createdAt }`

| Method | Path | Notes |
|---|---|---|
| GET | `/notifications?cursor&limit` | `{items, nextCursor, unreadCount}` |
| POST | `/notifications/:id/read` | 204 |
| POST | `/notifications/read-all` | 204 |

## Realtime (WebSocket)

`GET /ws` upgrades. Within 5 s the client must send `{"type":"auth","token":"<accessToken>"}`; server replies `{"type":"ready"}` or closes with 4401. Server -> client events (all `{type, data}`):

- `message.new` `data: Message`
- `messages.read` `data: {conversationId, readerId}`
- `match.new` `data: MatchSummary`
- `match.removed` `data: {matchId, conversationId}`
- `notification.new` `data: Notification`

Client -> server: `{"type":"ping"}` -> `{"type":"pong"}`. The hub is in-process (single API instance); scaling out requires a pub/sub backplane (documented, out of scope).

## Health

`GET /health` **public** -> 200 `{status:"ok"}` when the DB answers.

## Server behaviour notes (clarifications of the contract)

These are decisions the server makes where the contract above is silent. Clients may rely on them.

- **Auth**: `AuthResponse` has no other fields (no legacy `token`). Access tokens are HS256 with `uid`, `sid`, `iat`, `exp` only; every request re-checks that the session is not revoked and the user still exists. Passwords: 8..128 characters (any length is fully significant). `/auth/*` is rate limited per client IP (30 requests/minute). A wrong `currentPassword` (`/me/password`) or `password` (`DELETE /me`) is 403 `forbidden`, not 401, so clients do not treat it as an expired session.
- **Password reset**: the emailed code is 8 characters from the alphabet `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` (no `0`, `1`, `I`, `O`), valid 30 minutes, single use, max 5 attempts per code; requesting a new code invalidates the previous one. Input is case/space/dash tolerant.
- **Status codes**: unknown route 404 `not_found`; oversize JSON body (> 1 MB) 413 `payload_too_large`; photo > 8 MB 413; non JPEG/PNG or undecodable image 415 `unsupported_media`; 7th photo 409 `conflict`; `PUT /me/location` before a profile exists 409 `profile_incomplete`.
- **Conversations**: a user who is not a participant (or a blocked pair, for reading and marking read) gets 404 `not_found`, never 403, so conversation ids cannot be probed. Sending to a blocked pair is 403 `blocked`.
- **Swipes**: self or malformed target is 400; unknown, hidden (non-discoverable) or blocked target is 404 `not_found` (blocks are never revealed). Swipes require `profileComplete` (409 `profile_incomplete`).
- **Notifications**: message notifications are coalesced: while a user has an unread `message` notification for a conversation, new messages update it instead of adding rows. `POST /conversations/:id/read` also marks that conversation's message notifications read. Unmatching deletes the notifications that point at the match, and deleting an account removes notifications about that user.
- **Distance**: `distanceKm` is the great-circle distance rounded *up* to the next 5 km step (minimum 5) and `null` when either side lacks a location or the candidate set `showDistance=false`.
- **Rate limits** (per user unless noted): swipes 120/min, messages 30/min, reports 10/hour, `/auth/*` 30/min per IP. Limits are per API process.
- **Realtime**: the auth frame must arrive within 5 s (close code `4401` otherwise or on a bad/revoked token); at most 5 concurrent connections per user (`4429`); frames are limited to 4 KiB; the server pings every 30 s; browser origins must be in `ALLOWED_ORIGINS` (native clients send no `Origin`). `match.new` and `match.removed` are sent to both users, `message.new` to both participants (the sender receives an echo for multi-device sync), `messages.read` to the other participant, `notification.new` to its owner.

## Repository rules for this migration

- No file deletion (a human must validate removals). Legacy modules `billing`, `posts`, `comments`, `files` and the organizations/tenant model are **unwired** from the router and left on disk; they are listed in `docs/PENDING_REMOVAL.md` for the owner to validate.
- DB migrations are append-only: add new numbered files, never edit or delete applied ones. The runner must track applied files in a `schema_migrations` table.
