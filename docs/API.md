# API contract

Base URL: `EXPO_PUBLIC_API_URL` (default dev port `18080`). JSON everywhere (camelCase), except photo upload (`multipart/form-data`).
All endpoints except `/health` and `/auth/*` require `Authorization: Bearer <accessToken>`.

## Conventions

- **Errors**: `{"error": "human readable message", "code": "machine_code", "fields": {"displayName": "reason"}}` (`fields` only on `validation_failed`).
  Codes: `validation_failed` (400/422→400), `unauthorized` (401), `forbidden` (403), `not_found` (404), `conflict` (409), `profile_incomplete` (409), `rate_limited` (429, header `Retry-After`), `payload_too_large` (413), `internal` (500).
- **Pagination**: `?limit=` (1..50, default 20) and opaque `?cursor=`; responses contain `nextCursor` (`null` when finished).
- **IDs**: UUID strings. **Timestamps**: RFC 3339 UTC.
- **Access token**: 15 min. Refresh with `/auth/refresh` (rotation: the old refresh token stops working). A 401 with `code: "unauthorized"` on a normal call means "refresh then retry once".
- **Age gate**: 18+ only (server enforced).
- **Everything is free**: no billing / quota / boost endpoint exists.

## Types

```ts
type Gender = "man" | "woman" | "nonbinary";
type Photo = { id: string; position: number; url: string; thumbUrl: string }; // url relative to base URL, requires Authorization header
type Interest = { slug: string; label: string };
type Preferences = { interestedIn: Gender[]; minAge: number; maxAge: number; maxDistanceKm: number }; // 18..99, 1..500
type Me = { id: string; email: string; profileCompleted: boolean; createdAt: string };
type AuthResponse = { accessToken: string; refreshToken: string; user: Me };
type OwnProfile = {
  userId: string; displayName: string; birthDate: string /* YYYY-MM-DD */; age: number; gender: Gender;
  bio: string; city: string; interests: Interest[]; photos: Photo[];
  showDistance: boolean; discoverable: boolean; hasLocation: boolean;
  preferences: Preferences; completed: boolean; createdAt: string; updatedAt: string;
};
type PublicProfile = {
  userId: string; displayName: string; age: number; gender: Gender; bio: string; city: string;
  interests: Interest[]; photos: Photo[]; distanceKm: number | null; commonInterests: number;
};
type ProfileCard = { userId: string; displayName: string; age: number; photo: Photo | null };
type Message = { id: string; conversationId: string; senderId: string; body: string; createdAt: string; readAt: string | null };
type MatchSummary = {
  id: string; conversationId: string; matchedAt: string; user: ProfileCard;
  lastMessage: Message | null; unreadCount: number; updatedAt: string;
};
type Notification = {
  id: string; type: "match" | "system"; title: string; body: string;
  data: { matchId?: string; conversationId?: string; userId?: string }; readAt: string | null; createdAt: string;
};
```

`distanceKm` is **approximate** (rounded up to 5 km steps, min 5) and `null` when either side hides distance or has no location.
`completed` / `profileCompleted` = profile exists **and** has at least one photo. Only completed profiles can use `/discover`, `/swipes` and `/profiles/:userId` (otherwise `409 profile_incomplete`).

## Auth

| Method & path | Body | Response |
|---|---|---|
| `POST /auth/register` | `{email, password}` (password 8 chars min, 72 bytes max) | `201 AuthResponse`; `409 conflict` if email exists |
| `POST /auth/login` | `{email, password}` | `200 AuthResponse`; `401` invalid credentials |
| `POST /auth/refresh` | `{refreshToken}` | `200 AuthResponse` |
| `POST /auth/logout` | `{refreshToken}` | `204` |
| `POST /auth/forgot-password` | `{email}` | always `202` (never reveals whether the account exists) |
| `POST /auth/reset-password` | `{token, password}` | `204`; revokes every session; `400` if token invalid/expired |
| `GET /me` | – | `200 Me` |
| `POST /me/password` | `{currentPassword, newPassword}` | `204`; revokes other sessions |
| `DELETE /me` | `{password}` | `204`; deletes account, photos, matches, messages, likes… |

Auth endpoints are rate limited (per IP).

## Profile & preferences

| Method & path | Body | Response |
|---|---|---|
| `GET /profile` | – | `200 OwnProfile`, or `404 not_found` if the profile was never created |
| `PUT /profile` | `{displayName(2..40), birthDate, gender, bio(<=500), city(<=80), interests: string[] (slugs, <=10), showDistance, discoverable}` | `200 OwnProfile` (creates or updates; creating also creates default preferences) |
| `PUT /profile/location` | `{latitude, longitude}` | `204` (server rounds to ~1 km before storing) |
| `DELETE /profile/location` | – | `204` |
| `GET /preferences` | – | `200 Preferences` |
| `PUT /preferences` | `Preferences` (`minAge<=maxAge`) | `200 Preferences` |
| `GET /interests` | – | `200 {interests: Interest[]}` |

## Photos (max 6, position 0 = main photo)

| Method & path | Notes |
|---|---|
| `POST /profile/photos` | multipart field `file` (jpeg/png/webp, <= 8 MB, <= 8000px). Server re-encodes to JPEG (EXIF stripped), creates full + thumbnail. Appends at the end. `201 Photo` |
| `DELETE /profile/photos/:id` | `204`; remaining photos are compacted |
| `PUT /profile/photos/order` | `{ids: string[]}` must contain every current photo id exactly once → `200 {photos: Photo[]}`; first = main |
| `GET /photos/:id/file?size=thumb\|full` | image/jpeg, `Cache-Control: private, max-age=86400, immutable`. Only the owner or a user allowed to see that profile (not blocked) |

## Discovery, likes, matches

| Method & path | Body | Response |
|---|---|---|
| `GET /discover?limit=10` (max 20) | – | `200 {profiles: PublicProfile[]}` (never returns already-swiped, matched, blocked or non-compatible profiles) |
| `GET /profiles/:userId` | – | `200 PublicProfile` (full profile; 404 if blocked/not visible) |
| `POST /swipes` | `{targetId, action: "like"\|"pass"}` | `200 {action, matched: boolean, match: MatchSummary \| null}`. Idempotent: repeating returns the stored result, never duplicates |
| `GET /matches?limit&cursor` | – | `200 {matches: MatchSummary[], nextCursor}` (newest first) |
| `DELETE /matches/:id` | – | `204` unmatch (conversation closes for both) |

## Chat

| Method & path | Body | Response |
|---|---|---|
| `GET /conversations?limit&cursor` | – | `200 {conversations: MatchSummary[], nextCursor}` — conversations with >=1 message, not hidden by me, ordered by latest activity |
| `GET /conversations/:id/messages?limit&cursor` | – | `200 {messages: Message[] (newest first), nextCursor}`; `404` if not a participant |
| `POST /conversations/:id/messages` | `{body (1..2000, trimmed)}` | `201 Message`; `403 forbidden` if the match was removed (unmatch/block); rate limited (60/min) |
| `POST /conversations/:id/read` | – | `204` marks everything received as read |
| `DELETE /conversations/:id` | – | `204` hides the conversation and clears history **for me only** |
| `GET /conversations/unread-count` | – | `200 {unreadCount: number}` |

## Safety

| Method & path | Body | Response |
|---|---|---|
| `GET /blocks` | – | `200 {blocks: [{userId, displayName, blockedAt}]}` |
| `POST /blocks` | `{userId}` | `204` (also removes any match) |
| `DELETE /blocks/:userId` | – | `204` |
| `POST /reports` | `{userId, reason: "spam"\|"fake"\|"harassment"\|"inappropriate"\|"underage"\|"other", details?(<=1000), block?: boolean}` | `201 {id}` |

## Notifications

| Method & path | Response |
|---|---|
| `GET /notifications?limit&cursor` | `200 {notifications: Notification[], nextCursor}` |
| `GET /notifications/unread-count` | `200 {unreadCount}` |
| `POST /notifications/:id/read` | `204` |
| `POST /notifications/read-all` | `204` |

## Realtime (WebSocket)

`GET /ws` upgraded to WebSocket. First frame from the client **within 5 s**: `{"type":"auth","token":"<accessToken>"}`.
Server answers `{"type":"ready"}`, or closes (code 4401) when the token is invalid. The server sends pings; clients only need to answer pongs (done by the browser/RN stack).
Client → server frames other than `auth` are ignored: send messages through REST.

Server events `{ "type": string, "data": object }`:

| type | data |
|---|---|
| `message.new` | `Message` |
| `message.read` | `{conversationId, readAt, readerId}` (a participant read the conversation; also delivered to the reader's other devices) |
| `match.new` | `MatchSummary` |
| `match.removed` | `{matchId, conversationId}` (unmatch or block) |
| `notification.new` | `Notification` |

`message.read` and `message.new` are delivered to **both** participants (so a user's other devices stay in sync): compare `senderId` / `readerId` with your own id.

The client should reconnect with exponential backoff and re-fetch conversations on reconnect (events are not replayed).

## Health

`GET /health` → `200 {"status":"ok"}`.
