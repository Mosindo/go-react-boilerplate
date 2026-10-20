# API contract

JSON everywhere, camelCase. Errors: `{"error": "human readable message"}` with an
appropriate HTTP status (400 validation, 401 auth, 403 forbidden, 404 not found,
409 conflict, 413 too large, 415 unsupported media, 429 rate limited).
Authenticated routes need `Authorization: Bearer <accessToken>` (access token TTL 15 min,
refresh via `/auth/refresh`). Lists are paginated with an opaque `cursor` / `nextCursor`
(`nextCursor` is `null` on the last page) and a `limit` (default 20, max 50).

## Shapes

```
User            { id, email, createdAt, profileComplete: boolean }
Interest        { slug, label }
Photo           { id, position, url }            // url is relative: "/photos/<id>/file" (auth header required)
Profile         { userId, firstName, age, gender, bio, city, interests: Interest[],
                  photos: Photo[], distanceKm: number|null,
                  birthDate?: "YYYY-MM-DD" }      // birthDate only on the caller's own profile
Preferences     { interestedIn: Gender[], minAge, maxAge, maxDistanceKm,
                  discoverable: boolean, showDistance: boolean }
Gender          "man" | "woman" | "non_binary"
UserSummary     { userId, firstName, photoUrl: string|null }
Message         { id, conversationId, senderId, body, createdAt }
```

`distanceKm` is rounded to coarse buckets (1, 2, 3, 5, 10, 15, 25, 50, 100...) and is `null`
when unknown or when the other user disabled `showDistance`. Exact coordinates are never returned.

## Auth

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/auth/register` | `{email, password}` (password 8-128) | 201 `{accessToken, refreshToken, user: User}` |
| POST | `/auth/login` | `{email, password}` | 200 same as register |
| POST | `/auth/refresh` | `{refreshToken}` | 200 same (rotates refresh token) |
| POST | `/auth/logout` | `{refreshToken}` | 204 |
| POST | `/auth/password-reset/request` | `{email}` | 202 `{}` always (no account enumeration) |
| POST | `/auth/password-reset/confirm` | `{token, password}` | 204 (revokes all sessions) |
| GET | `/me` | | `User` |
| DELETE | `/me` | `{password}` | 204, deletes the account and every associated row/file |

## Profile

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/interests` | | `{interests: Interest[]}` |
| GET | `/profile/me` | | `{profile: Profile \| null, preferences: Preferences, complete: boolean}` |
| PUT | `/profile/me` | `{firstName, birthDate, gender, bio, city, interests: string[] (slugs, max 10)}` | `Profile` (age >= 18 enforced) |
| PUT | `/profile/preferences` | `Preferences` | `Preferences` |
| PUT | `/profile/location` | `{latitude, longitude}` | 204 (stored rounded to ~1 km) |
| GET | `/profiles/:userId` | | `Profile` (404 if missing, blocked either way, incomplete or deleted) |

A profile is `complete` when profile fields exist, it has at least 1 photo and a stored location.

## Photos

| Method | Path | Notes |
|---|---|---|
| POST | `/photos` | multipart field `file` (JPEG or PNG, max 8 MB, max 6 photos). Re-encoded server side (EXIF stripped, max 1280 px). Returns `Photo`. |
| PUT | `/photos/order` | `{photoIds: string[]}` must contain exactly the caller's photos; first is primary. Returns `{photos: Photo[]}` |
| DELETE | `/photos/:id` | 204 |
| GET | `/photos/:id/file` | image bytes; only owner or users allowed to see the owner's profile |

Replacing a photo = upload the new one, reorder, delete the old one.

## Discovery, swipes, matches

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/discover?limit=10` | | `{profiles: Profile[]}` already filtered (mutual gender/age/distance preferences, not swiped, not blocked, discoverable, complete) |
| POST | `/swipes` | `{targetUserId, action: "like"\|"pass"}` | 200 `{matched: boolean, match: MatchItem \| null}`; repeated swipe on same target = 409 |
| GET | `/matches?cursor&limit` | | `{matches: MatchItem[], nextCursor}` |
| DELETE | `/matches/:id` | | 204 (unmatch; the pair never reappears in discovery) |

`MatchItem { id, user: UserSummary, conversationId, createdAt }`

## Chat

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/conversations?cursor&limit` | | `{conversations: ConversationItem[], nextCursor}` (hidden ones excluded) |
| GET | `/conversations/:id/messages?cursor&limit` | | `{messages: Message[] (newest first), otherLastReadAt: string\|null, nextCursor}` |
| POST | `/conversations/:id/messages` | `{body}` (1-2000 chars) | 201 `Message` |
| POST | `/conversations/:id/read` | | 204 |
| DELETE | `/conversations/:id` | | 204 (hides for the caller only; reappears on next message) |

`ConversationItem { id, user: UserSummary, lastMessage: Message|null, unreadCount, updatedAt }`
Non-participants always get 404.

## Safety

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/blocks` | `{userId}` | 204 (also unmatches and hides conversation) |
| GET | `/blocks` | | `{blocks: UserSummary[]}` |
| DELETE | `/blocks/:userId` | | 204 |
| POST | `/reports` | `{userId, reason, details?}` reason in `spam, fake_profile, inappropriate_content, harassment, underage, other` | 201 `{}` |

## Notifications

`Notification { id, type: "match"|"message", actor: UserSummary|null, data: object, readAt: string|null, createdAt }`
(`data` carries `conversationId` / `matchId`). Message notifications are coalesced to one unread per conversation.

| Method | Path | Response |
|---|---|---|
| GET | `/notifications?cursor&limit` | `{notifications, unreadCount, nextCursor}` |
| POST | `/notifications/:id/read` | 204 |
| POST | `/notifications/read-all` | 204 |

## Real time

`GET /ws` upgrades to WebSocket. The first client frame must be `{"type":"auth","token":"<accessToken>"}`
within 5 seconds, the server answers `{"type":"ready"}`. Server events (JSON text frames):

```
{"type":"message.new","message":Message}
{"type":"message.read","conversationId":"..","userId":"..","readAt":".."}
{"type":"match.new","match":MatchItem}
{"type":"notification.new","notification":Notification}
```

Clients send `{"type":"ping"}` periodically (server answers `{"type":"pong"}`). Sending messages
goes through the REST endpoint.

## Health

`GET /health` → 200 `{"status":"ok"}`.
