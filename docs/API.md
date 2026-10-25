# API contract

REST + one WebSocket. JSON is camelCase. All timestamps are RFC 3339 UTC. All endpoints except `/health` and `/auth/*` need
`Authorization: Bearer <accessToken>`.

Errors: HTTP status + `{"error": "<human readable message>", "code": "<machine_code>"}`.
Common codes: `invalid_request` (400), `unauthorized` (401), `forbidden` (403), `not_found` (404), `conflict` (409),
`rate_limited` (429, `Retry-After` header), `payload_too_large` (413), `internal` (500).

Paginated lists return `{"items": [...], "nextCursor": "<opaque>" | null}`; pass `?limit=` (1..50, default 20) and `?cursor=`.

## Types

```ts
type Gender = "man" | "woman" | "nonbinary";

type Photo = { id: string; position: number; isPrimary: boolean; url: string; thumbUrl: string }; // urls are API-relative, need the Bearer header

type Interest = { slug: string; label: string };

type MyProfile = {            // never contains coordinates
  userId: string; firstName: string; birthDate: string /* YYYY-MM-DD */; age: number;
  gender: Gender; bio: string; city: string; hasLocation: boolean;
  discoverable: boolean; showDistance: boolean;
  interests: Interest[]; photos: Photo[];
};

type Preferences = { interestedIn: Gender[]; minAge: number; maxAge: number; maxDistanceKm: number | null };

type PublicProfile = {        // what other people see
  userId: string; firstName: string; age: number; bio: string; city: string;
  distanceKm: number | null;  // approximate (rounded up to a multiple of 5), null when unknown or hidden
  interests: Interest[]; photos: Photo[];
  sharedInterests: string[];  // slugs
};

type Message = { id: string; conversationId: string; senderId: string; body: string; createdAt: string; readAt: string | null };

type ProfileSummary = { userId: string; firstName: string; thumbUrl: string | null };
```

## Auth

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/auth/register` | `{email, password}` (password 8..72 bytes, bcrypt limit) | 201 `AuthResponse` |
| POST | `/auth/login` | `{email, password}` | 200 `AuthResponse` |
| POST | `/auth/refresh` | `{refreshToken}` | 200 `AuthResponse` (rotates refresh token) |
| POST | `/auth/logout` | `{refreshToken}` | 204 |
| POST | `/auth/password/forgot` | `{email}` | 202 always (no account enumeration). A one-time code is e-mailed (logged server-side when SMTP is not configured) |
| POST | `/auth/password/reset` | `{email, code, password}` | 204; revokes every session. 400 `invalid_code` on a wrong/expired code |
| GET | `/me` | | `Me` |
| POST | `/me/password` | `{currentPassword, newPassword}` | 204 |
| DELETE | `/me` | `{password}` | 204. Deletes the account and every associated row |

`AuthResponse = {accessToken, refreshToken, user: Me}`; `Me = {id, email, createdAt, profileComplete: boolean}`.
Register requires nothing else: the user must be 18+, which is enforced when the profile is saved (`birthDate`).
Access token lives 15 minutes; refresh with `/auth/refresh`. Codes: `email_taken` (409), `invalid_credentials` (401), `weak_password` (400). Emails are trimmed and lower-cased server-side.

## Profile and preferences

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/profile` | | `{profile: MyProfile \| null, preferences: Preferences \| null, complete: boolean}` |
| PUT | `/profile` | `{firstName(1..40), birthDate, gender, bio(0..500), interests: string[] (slugs, max 10), discoverable, showDistance}` | `MyProfile` (creates or updates). 400 `underage` if < 18, 400 on age > 99 |
| PUT | `/preferences` | `{interestedIn: Gender[] (1..3), minAge(18..99), maxAge(minAge..99), maxDistanceKm: number(1..500) \| null}` | `Preferences` |
| PUT | `/profile/location` | `{latitude, longitude, city(0..80)}` | `MyProfile`. Coordinates are rounded to 2 decimals (~1 km) before storage |
| DELETE | `/profile/location` | | 204 |
| GET | `/interests` | | `{items: Interest[]}` (catalog) |
| GET | `/profiles/:userId` | | `PublicProfile`. 404 if blocked either way, unknown, or not visible |

`complete` is true when profile + preferences exist and the user has at least one photo. Incomplete profiles are not shown in discovery.

## Photos (max 6 per user, position 0 is the primary photo)

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/photos` | multipart, field `file` (JPEG or PNG, <= 8 MB) | 201 `Photo` (appended). 409 `photo_limit` |
| PUT | `/photos/:id` | multipart, field `file` | `Photo` (replaces the picture and keeps its position; **the id changes**, so drop the old one from caches) |
| PUT | `/photos/order` | `{photoIds: string[]}` (exactly the user's photo ids) | `{items: Photo[]}` |
| DELETE | `/photos/:id` | | 204 (positions are compacted) |
| GET | `/photos/:id/image?size=full\|thumb` | | image/jpeg bytes. Needs the Bearer header |

Uploads are decoded, resized (max 1080 px, thumb 320 px) and re-encoded as JPEG (metadata stripped). 400 `invalid_image`, 413 `payload_too_large`.

## Discovery, swipes, matches

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/discovery?limit=10` | | `{items: PublicProfile[]}` (limit 1..20). Excludes self, already-swiped, blocked (both ways), incomplete/non-discoverable profiles, and non-mutual age/gender/distance preferences |
| POST | `/discovery/swipes` | `{userId, action: "like" \| "pass"}` | `{result: "liked" \| "passed" \| "matched" \| "already_swiped", match: MatchItem \| null}` |
| GET | `/matches` | | paginated `MatchItem` (newest first) |
| DELETE | `/matches/:matchId` | | 204 unmatch: removes the conversation and its messages |

`MatchItem = {matchId, conversationId, createdAt, profile: PublicProfile}`. The first swipe on a person is final.
A swipe on yourself, an unknown, blocked or ineligible user returns 404 `not_found`. Profiles must be complete to swipe (403 `profile_incomplete`).

## Chat

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/conversations` | | paginated `ConversationItem`, most recently active first. Only conversations that contain at least one message are listed (brand-new matches come from `/matches`). A locally deleted conversation (`DELETE`) clears the caller's history and reappears with the next message |
| GET | `/conversations/:id/messages` | | paginated `Message`, newest first (cursor goes back in time) |
| POST | `/conversations/:id/messages` | `{body}` (trimmed, 1..2000 chars) | 201 `Message` |
| POST | `/conversations/:id/read` | | 204 marks the other person's messages as read |
| DELETE | `/conversations/:id` | | 204 clears the history for the caller only (the other person keeps theirs) |
| GET | `/conversations/unread-count` | | `{unreadCount: number}` (total unread messages) |

`ConversationItem = {id, matchId, otherUser: ProfileSummary, lastMessage: Message, unreadCount, updatedAt}`.
Non-participants get 404 on every conversation endpoint.

### WebSocket `GET /ws`

1. Open the socket, then send within 5 s: `{"type":"auth","token":"<accessToken>"}`. The server replies `{"type":"ready"}` or closes with code 4401.
2. Client may send `{"type":"ping"}` (reply `{"type":"pong"}`).
3. Server pushes `{"type": <event>, "data": ...}`:
   - `message` -> `Message` (also sent to the sender's other devices)
   - `read` -> `{conversationId, readerId, readAt}`
   - `match` -> `MatchItem`
   - `conversation_removed` -> `{conversationId}` (unmatch or block)
   - `notification` -> `Notification`

REST remains the source of truth: refetch after reconnecting.

## Safety

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/blocks` | | paginated `{userId, firstName, thumbUrl, createdAt}` |
| POST | `/blocks` | `{userId}` | 204. Also removes any match/conversation |
| DELETE | `/blocks/:userId` | | 204 |
| POST | `/reports` | `{userId, reason: "spam" \| "fake_profile" \| "harassment" \| "inappropriate_content" \| "underage" \| "other", details?(0..1000)}` | 201 `{id}` |

## Notifications

| Method | Path | Response |
|---|---|---|
| GET | `/notifications` | `{items: Notification[], nextCursor, unreadCount}` |
| POST | `/notifications/:id/read` | 204 |
| POST | `/notifications/read-all` | 204 |

`Notification = {id, type: "match" | "message", actor: ProfileSummary | null, conversationId: string | null, readAt: string | null, createdAt: string}`.
Message notifications are coalesced: at most one unread per conversation.

## Health

`GET /health` -> `{status:"ok"}` (503 when the database is unreachable).
