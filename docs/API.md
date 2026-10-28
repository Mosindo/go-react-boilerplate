# API contract

JSON everywhere (camelCase), except photo upload (`multipart/form-data`) and photo download (image bytes).
All endpoints except `/health`, `/auth/*` require `Authorization: Bearer <accessToken>`.
Rate limiting answers 429 with a `Retry-After` header.
Errors: `{"error": "<human message>", "code": "<machine_slug>"}` with the proper HTTP status
(400 validation, 401 auth, 403 forbidden, 404 not found, 409 conflict, 413 too large, 429 rate limited).
Timestamps are RFC 3339 UTC. IDs are UUID strings (interests: integers).
Cursor pagination: `?limit=` (1..50, default 20) and `?cursor=` (opaque string from `nextCursor`; absent/`null` = end).

## Auth
| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/auth/register` | `{email, password}` (password 8..72 bytes — bcrypt limit; email is trimmed and lower-cased) | 201 `AuthResponse` |
| POST | `/auth/login` | `{email, password}` | 200 `AuthResponse` |
| POST | `/auth/refresh` | `{refreshToken}` | 200 `AuthResponse` (rotated refresh token) |
| POST | `/auth/logout` | `{refreshToken}` | 204 |
| POST | `/auth/forgot-password` | `{email}` | 202 always (no account enumeration) |
| POST | `/auth/reset-password` | `{email, code, password}` | 204 (revokes all sessions) |
| GET | `/me` | | `Account` |
| POST | `/me/password` | `{currentPassword, newPassword}` | 204 |
| DELETE | `/me` | `{password}` | 204 – deletes account, profile, photos (files too), messages, matches… |

`AuthResponse = {accessToken, refreshToken, user: Account}`  
`Account = {id, email, createdAt, profileComplete: boolean}` (`profileComplete` = profile exists with at least 1 photo and preferences)

## Profile
| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/interests` | | `{interests: Interest[]}` where `Interest = {id, slug, label, category}` |
| GET | `/profile` | | `OwnProfile` or 404 `profile_not_found` |
| PUT | `/profile` | `ProfileInput` | 200 `OwnProfile` (create or update) |
| PUT | `/location` | `{latitude, longitude, city?}` | 200 `OwnProfile`. Stored coarsened (~1 km grid). |
| GET | `/preferences` | | `Preferences` (defaults if none) |
| PUT | `/preferences` | `Preferences` | 200 `Preferences` |
| GET | `/profiles/:userId` | | `PublicProfile` (404 if blocked either way, hidden or unknown) |

`ProfileInput = {firstName (1..40), birthDate "YYYY-MM-DD" (must be ≥ 18 years old), gender, bio (≤ 500), city? (≤ 80), interestIds: number[] (≤ 10), showDistance: boolean, discoverable: boolean}`  
`gender ∈ "man" | "woman" | "nonbinary"`  
`Preferences = {interestedIn: gender[] (≥1), minAge (18..99), maxAge (minAge..99), maxDistanceKm (1..500)}`  
`OwnProfile = {userId, firstName, birthDate, age, gender, bio, city, hasLocation, showDistance, discoverable, interests: Interest[], photos: Photo[]}`  
`PublicProfile = {userId, firstName, age, gender, bio, city, distanceKm: number|null, interests: Interest[], photos: Photo[]}`  
`distanceKm` is an approximation (rounded up to 5 km steps, min 1; `null` when either side hides distance or has no location). Never exact coordinates.  
`Photo = {id, position, width, height, url}`; `url` is a relative path (`/photos/:id/file`) that must be fetched with the Bearer token.

## Photos
| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/photos` | multipart field `photo` (JPEG/PNG/WebP, ≤ 8 MB, 200..12000 px per side) | 201 `Photo` (max 6 per user; re-encoded server side as JPEG ≤ 1600 px, metadata stripped) |
| GET | `/photos/:id/file` | | image bytes (owner, or viewer allowed to see the owner) |
| PUT | `/photos/order` | `{photoIds: string[]}` (exactly the user's photos; first = main) | 200 `{photos: Photo[]}` |
| DELETE | `/photos/:id` | | 204 (remaining positions compacted) |

Requires an existing profile (404 `profile_not_found` otherwise).

## Discovery & swipes
| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/discover?limit=10` | | `{profiles: PublicProfile[]}` — queue of candidates matching both users' preferences, excluding already-swiped, blocked, matched. Empty array = nothing left. |
| POST | `/swipes` | `{targetUserId, action: "like"\|"pass"}` | 200 `{action, matched: boolean, conversation?: Conversation}` — idempotent; repeating returns the stored result |
| POST | `/swipes/reset-passes` | | 204 — passed profiles may reappear |

Requires `profileComplete`, otherwise 403 `profile_incomplete`.

## Conversations (a match = one conversation)
`Conversation = {id, matchedAt, user: {userId, firstName, age, photoUrl: string|null}, lastMessage: Message|null, unreadCount}`  
`Message = {id, conversationId, senderId, body (1..2000), createdAt, readAt: string|null}`

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/conversations?limit&cursor` | | `{conversations: Conversation[], nextCursor}` ordered by last activity desc |
| GET | `/conversations/:id/messages?limit&cursor` | | `{messages: Message[] (newest first), nextCursor}` |
| POST | `/conversations/:id/messages` | `{body}` | 201 `Message` |
| POST | `/conversations/:id/read` | | 204 — marks the other side's messages as read |
| DELETE | `/conversations/:id` | | 204 — clears history and hides it for me only (reappears on new message) |
| POST | `/conversations/:id/unmatch` | | 204 — removes the match for both (they will not see each other again) |

Non-participants get 404. After a block, the conversation disappears and sending returns 403 `blocked`.

## Notifications
`Notification = {id, type: "match"|"message", conversationId, actor: {userId, firstName}, readAt, createdAt}`

| Method | Path | Response |
|---|---|---|
| GET | `/notifications?limit&cursor` | `{notifications, nextCursor, unreadCount}` |
| POST | `/notifications/:id/read` | 204 |
| POST | `/notifications/read-all` | 204 |

## Safety
| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/blocks` | `{userId}` | 204 (idempotent; removes match/conversation access) |
| DELETE | `/blocks/:userId` | | 204 |
| GET | `/blocks` | | `{blocks: [{userId, firstName, createdAt}]}` |
| POST | `/reports` | `{userId, reason, details? (≤1000), conversationId?}` `reason ∈ spam, fake_profile, harassment, inappropriate_content, underage, other` | 201 `{id}` |

## Realtime — `GET /ws` (WebSocket upgrade)
1. Client connects, then must send within 5 s: `{"type":"auth","token":"<accessToken>"}`. Server replies `{"type":"ready"}` or closes (code 4401).
2. Server pushes events `{"type":"message"|"match"|"read"|"notification", "data": …}`:
   - `message` → `Message` (sent to the recipient; also echoed to the sender's other devices)
   - `match` → `Conversation`
   - `read` → `{conversationId, readerId, readAt}`
   - `notification` → `Notification`
3. Client may send `{"type":"ping"}`; server answers `{"type":"pong"}`. Sending messages stays on the REST endpoint (single source of validation).
Single-instance hub (in-process); see README for scaling notes.
