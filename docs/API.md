# API contract

Single source of truth shared by the Go API and the mobile client. JSON is camelCase.
All authenticated endpoints need `Authorization: Bearer <accessToken>`.
Errors are always `{"error": "<human readable message>"}` with a meaningful status:
`400` malformed/invalid input, `401` unauthenticated, `403` forbidden, `404` not found
(also used when the caller is not allowed to know a resource exists), `409` conflict,
`413` payload too large, `422` business-rule violation, `429` rate limited (`Retry-After` header).

Timestamps are RFC 3339 UTC. IDs are UUID strings.

## Shared shapes

```ts
type Gender = "woman" | "man" | "non_binary";
type Photo = { id: string; url: string; position: number };   // url is relative & signed; prefix with API base URL
type PublicProfile = {
  userId: string;
  firstName: string;
  age: number | null;            // null when the user hides their age
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;     // approximate: ceil to 1 km under 10 km, to 5 km above; null when hidden/unknown
  interests: string[];           // interest slugs
  photos: Photo[];               // ordered, photos[0] is the main photo
};
type Message = {
  id: string; conversationId: string; senderId: string; body: string;
  createdAt: string; readAt: string | null; // readAt: when the other participant read it
};
type ConversationSummary = {
  id: string;                    // conversation id
  matchId: string;
  matchedAt: string;
  user: { userId: string; firstName: string; age: number | null; photo: Photo | null };
  lastMessage: Message | null;   // null => "new match", nobody wrote yet
  unreadCount: number;           // messages from the other user not yet read by me
  updatedAt: string;             // lastMessage.createdAt or matchedAt
};
type AppNotification = {
  id: string; type: "match" | "message"; title: string; body: string;
  data: { conversationId?: string; userId?: string }; isRead: boolean; createdAt: string;
};
```

## Auth (public; rate limited per IP)

| Method | Path | Body | Success |
|---|---|---|---|
| POST | `/auth/register` | `{email, password (8-128), birthDate "YYYY-MM-DD"}` | 201 `AuthResponse` |
| POST | `/auth/login` | `{email, password}` | 200 `AuthResponse` |
| POST | `/auth/refresh` | `{refreshToken}` | 200 `AuthResponse` (refresh token rotates) |
| POST | `/auth/logout` | `{refreshToken}` | 204 |
| POST | `/auth/forgot` | `{email}` | 202 always (never reveals whether the account exists) |
| POST | `/auth/reset` | `{token, password}` | 204; revokes every session of the user |

`AuthResponse = {accessToken, refreshToken, token (alias of accessToken), user: Me}`.
Register: `422` if under 18 or birth date in the future/absurd (>120 years); `409` if email exists.
Access token TTL 15 min (HS256, claims `uid`, `sid`, `exp`); refresh token 30 days.

## Account (authenticated)

| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/me` | | 200 `Me = {id, email, birthDate, age, createdAt, profileComplete}` |
| POST | `/me/password` | `{currentPassword, newPassword}` | 204; revokes the user's other sessions |
| DELETE | `/me` | `{password}` | 204; permanently deletes the account, photos (files + rows), likes, matches, messages, notifications, blocks, reports |

`profileComplete` = profile exists AND at least one photo AND a location is set.

## Profile (authenticated)

| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/interests` | | 200 `{interests: [{slug, label}]}` |
| GET | `/me/profile` | | 200 `MyProfile` or 404 (onboarding not started) |
| PUT | `/me/profile` | `{firstName (1-40), gender, bio (<=500), city (<=80), interests: string[] (<=10 slugs), showDistance, showAge, discoverable}` | 200 `MyProfile` (upsert; first creation also creates default preferences: interestedIn = all genders, age 18-99, 50 km) |
| PUT | `/me/location` | `{latitude, longitude}` | 204. Coordinates are rounded to 2 decimals server-side before being stored |
| GET | `/me/preferences` | | 200 `Preferences` (404 before the profile exists) |
| PUT | `/me/preferences` | `{interestedIn: Gender[] (>=1), ageMin (18-99), ageMax (>=ageMin, <=99), maxDistanceKm (1-500)}` | 200 `Preferences` |
| GET | `/profiles/:userId` | | 200 `PublicProfile` |

`MyProfile = PublicProfile-like + {hasLocation, showDistance, showAge, discoverable, isComplete, age (always set for self), distanceKm: null}`.
`GET /profiles/:userId` is allowed for the caller themself, matched users and discoverable users; it is 404 when either side blocked the other or the target has no profile.

## Photos

| Method | Path | Notes |
|---|---|---|
| POST | `/me/photos` | multipart field `file`; JPEG/PNG/WebP only (validated by content sniffing, not by extension/Content-Type); max 5 MB upload; decoded and re-encoded to JPEG (EXIF/metadata stripped), longest side <= 1080 px; max 6 photos per user (`422` beyond). 201 `Photo` appended at the end |
| PUT | `/me/photos/order` | `{photoIds: string[]}` must be a permutation of the caller's photos; 200 `{photos: Photo[]}`; index 0 becomes the main photo |
| DELETE | `/me/photos/:id` | 204; removes the file; remaining photos are re-packed (positions 0..n-1) |
| GET | `/photos/:id/file?exp=&sig=` | **No bearer token needed**: authorised by the HMAC signature issued together with the URL (valid 6-12 h). Serves `image/jpeg`, `Cache-Control: private, max-age=3600`, `X-Content-Type-Options: nosniff`. 404 on bad/expired signature |

Signed URLs are only minted for photos the caller may see (self, discoverable candidates, matches, not blocked).
Replace a photo = delete + upload then reorder.

## Discovery & matching

| Method | Path | Body / query | Success |
|---|---|---|---|
| GET | `/discover?limit=10` | `limit` 1-20 | 200 `{profiles: PublicProfile[]}` |
| POST | `/swipes` | `{userId, action: "like" | "pass"}` | 200 `{matched: boolean, conversation: ConversationSummary | null}` |
| DELETE | `/matches/:matchId` | | 204 (unmatch: deletes the conversation for both; the pair never reappears in discovery) |

`/discover` returns only profiles that are: not me, complete (>=1 photo, has location), `discoverable`, never swiped by me (like OR pass), not blocked in either direction,
mutually compatible (their gender is in my `interestedIn` and mine is in theirs; their age within my range and mine within theirs), within
`min(myMaxDistance, theirMaxDistance)` km. Ordering is delegated to a `Ranker` (default: shared interests, then proximity, then recent activity; deterministic tie-break on user id) so the algorithm can evolve.
The viewer must have a complete profile themself (`422` otherwise).

`/swipes`: `400` on self-swipe or bad action; `404` if the target is not eligible/visible (blocked, missing, not discoverable and not already interested in me); `409` if I already swiped this user (no duplicates, no changing your mind through this endpoint).
Match rule: a `like` when the target already `like`d me creates exactly one match (row ordered user_a < user_b, unique) + one conversation + two participants, inside one transaction, then notifies both users (`match` notification + realtime).
Concurrent mutual likes must yield exactly one match.

## Conversations & messages

| Method | Path | Notes |
|---|---|---|
| GET | `/conversations?limit=30&before=<RFC3339 updatedAt>` | 200 `{conversations: ConversationSummary[]}` newest activity first; excludes conversations I hid locally (a new message from the other user un-hides it) and users I blocked / who blocked me |
| GET | `/conversations/:id/messages?limit=30&before=<messageId>` | 200 `{messages: Message[], nextCursor: string | null}` newest first. 404 if I'm not a participant |
| POST | `/conversations/:id/messages` | `{body}` trimmed, 1-2000 chars. 201 `Message`. 404 if not participant; 403 if blocked either way. Rate limited per user |
| POST | `/conversations/:id/read` | 204: marks everything from the other user as read, emits `conversation.read` to them |
| DELETE | `/conversations/:id` | 204: local deletion only (hides it for me, does not delete messages for the other user or unmatch) |

## Notifications

| Method | Path | Success |
|---|---|---|
| GET | `/notifications?limit=30&before=<RFC3339 createdAt>` | 200 `{notifications: AppNotification[], unreadCount: number}` |
| POST | `/notifications/:id/read` | 204 |
| POST | `/notifications/read-all` | 204 |

Clients cannot create notifications; the server raises them (`match`; `message`, coalesced to at most one unread notification per conversation).

## Safety

| Method | Path | Body | Success |
|---|---|---|---|
| POST | `/blocks` | `{userId}` | 204 (idempotent). Removes any match/conversation between the two |
| DELETE | `/blocks/:userId` | | 204 |
| GET | `/blocks` | | 200 `{blocks: [{userId, firstName, blockedAt}]}` |
| POST | `/reports` | `{userId, reason: "spam"|"fake_profile"|"harassment"|"inappropriate_content"|"underage"|"scam"|"other", details?: string (<=1000), block?: boolean}` | 201 `{id}`; `block:true` also blocks |

## Realtime

1. `POST /ws/ticket` (bearer) -> `{ticket, expiresInSeconds: 60}`
2. Open `ws(s)://<host>/ws?ticket=<ticket>`; server-to-client only, JSON `{type, data}`:
   - `message.new` `{message: Message}`
   - `conversation.read` `{conversationId, readAt}`
   - `match.new` `{conversation: ConversationSummary}`
   - `match.removed` `{conversationId}`
   - `notification.new` `{notification: AppNotification}`
3. Delivery is best effort: clients refetch over REST after every (re)connect and poll while the socket is down. Sending always goes through REST.

## Health

`GET /health` -> 200 `{"status":"ok"}`.
