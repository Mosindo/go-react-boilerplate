# API contract

JSON over HTTPS, camelCase, `Authorization: Bearer <accessToken>` on everything except the
routes marked *public*. Timestamps are RFC 3339 UTC. IDs are UUID strings.

Errors always look like `{"error": "human readable message", "code": "machine_code"}`.
Codes: `validation_error` 400, `unauthorized` 401, `forbidden` 403, `not_found` 404,
`conflict` 409, `already_swiped` 409, `profile_incomplete` 403, `underage` 400, `blocked` 403,
`rate_limited` 429 (with `Retry-After` header), `internal` 500.

Lists: `{"items": [...]}`. `/matches`, `/conversations` and `/notifications` take `limit` (default 20, max 50) and `offset`. `/discover` takes `limit` (default 10, max 20).

## Shared types

```ts
type Gender = "man" | "woman" | "nonbinary";
type Photo = { id: string; url: string; position: number };   // url is relative ("/media/…?exp=…&sig=…"); prefix with the API base URL, no auth header needed
type Interest = { slug: string; label: string };
type UserSummary = { userId: string; firstName: string; age: number; photo: Photo | null };
type User = { id: string; email: string; createdAt: string; hasProfile: boolean; profileComplete: boolean };
type SelfProfile = {
  userId: string; firstName: string; birthDate: string /* YYYY-MM-DD */; age: number;
  gender: Gender; bio: string; city: string; hasLocation: boolean;
  interests: Interest[]; photos: Photo[]; complete: boolean;   // complete = has >= 1 photo
};
type Preferences = {
  interestedIn: Gender[];      // non-empty
  minAge: number; maxAge: number;   // 18..99, minAge <= maxAge
  maxDistanceKm: number;       // 1..500
  discoverable: boolean;       // false = paused, not shown to anyone
  showDistance: boolean;       // false = others never see distanceKm
};
type PublicProfile = {
  userId: string; firstName: string; age: number; gender: Gender; bio: string; city: string;
  distanceKm: number | null;   // approximate (rounded), null if unknown or hidden
  interests: Interest[]; sharedInterests: string[] /* slugs */; photos: Photo[];
};
type Match = { id: string; user: UserSummary; conversationId: string; createdAt: string };
type Message = { id: string; conversationId: string; senderId: string; content: string; createdAt: string };
type Conversation = {
  id: string; matchId: string; user: UserSummary; lastMessage: Message | null;
  unreadCount: number; peerLastReadAt: string | null; updatedAt: string;
};
type Notification = {
  id: string; type: "match" | "message"; title: string; body: string;
  data: { conversationId?: string; userId?: string; matchId?: string };
  isRead: boolean; createdAt: string;
};
```

## Auth
| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/auth/register` *public* | `{email, password(8..72 bytes)}` | 201 `{accessToken, refreshToken, user: User}` |
| POST | `/auth/login` *public* | `{email, password}` | 200 same |
| POST | `/auth/refresh` *public* | `{refreshToken}` | 200 same (token rotated) |
| POST | `/auth/logout` *public* | `{refreshToken}` | 204 |
| POST | `/auth/password-reset/request` *public* | `{email}` | 202 `{ok: true}` always (no account enumeration); a reset code is e-mailed |
| POST | `/auth/password-reset/confirm` *public* | `{token, newPassword}` | 204; revokes every session |
| GET | `/me` | | `User` |
| DELETE | `/me` | `{password}` | 204; deletes the account and all associated data |

Rate limits answer 429 `rate_limited` with `Retry-After`. Access token lifetime is 15 minutes (and a logout, password reset or account deletion invalidates it immediately); refresh tokens 30 days and rotate on each use.

## Profile, preferences, photos
| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/interests` | | `{items: Interest[]}` (catalog) |
| GET | `/profile/me` | | `SelfProfile`, 404 `not_found` until created |
| PUT | `/profile/me` | `{firstName(1..40), birthDate, gender, bio(0..500), city(0..80), interests: slug[] (<=10)}` | `SelfProfile` (create or update). `birthDate` must make the user >= 18 (`underage`) |
| PUT | `/profile/me/location` | `{latitude, longitude}` | 204. Server rounds to ~1 km before storing |
| GET / PUT | `/preferences` | `Preferences` | `Preferences` (defaults: all genders, 18–99, 50 km, discoverable, showDistance) |
| GET | `/photos` | | `{items: Photo[]}` ordered; position 0 = main photo |
| POST | `/photos` | multipart field `file` (JPEG or PNG, <= 5 MB, max 6 photos) | 201 `Photo` (server re-encodes to JPEG) |
| PUT | `/photos/:id` | multipart field `file` | `Photo` (replace, keeps position) |
| PUT | `/photos/order` | `{photoIds: string[]}` (all own photos) | `{items: Photo[]}` |
| DELETE | `/photos/:id` | | 204 |
| GET | `/media/:id?exp&sig` *public (signed)* | | image/jpeg |

## Discovery and matching
Needs a complete profile (`profile_incomplete` otherwise).

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/discover?limit=10` | | `{items: PublicProfile[]}` — next best candidates, never already swiped, blocked, or outside either side's preferences. Empty list = nobody left |
| GET | `/users/:userId/profile` | | `PublicProfile` for a discoverable candidate or a match; 404 otherwise |
| POST | `/swipes` | `{targetUserId, action: "like"\|"pass"}` | `{matched: boolean, match: Match \| null}`; 409 `already_swiped` on repeat |
| GET | `/matches` | | `{items: Match[]}` newest first |
| DELETE | `/matches/:id` | | 204 unmatch (conversation deleted; person never reappears) |

## Chat
| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/conversations` | | `{items: Conversation[]}` ordered by latest activity; hidden conversations excluded |
| GET | `/conversations/:id/messages?limit=30&before=<messageId>` | | `{items: Message[] /* newest first */, nextCursor: string \| null}` |
| POST | `/conversations/:id/messages` | `{content(1..2000)}` | 201 `Message`; 403 `blocked` if either side blocked |
| POST | `/conversations/:id/read` | | 204 |
| DELETE | `/conversations/:id` | | 204 hides it for me only; reappears when a new message arrives |

## Realtime
`POST /realtime/ticket` → `{ticket, expiresIn}` (single use, 60 s). Open
`ws(s)://<host>/ws?ticket=<ticket>`; the server pushes `{ "type": string, "data": … }`:
`message` (`Message`), `read` (`{conversationId, readerId, readAt}`), `match` (`Match`),
`notification` (`Notification`), `unmatch` (`{matchId?, conversationId}`: the other person unmatched or blocked you; drop that conversation). Clients may send `{"type":"ping"}`; the server answers `{"type":"pong"}`.
Sending messages is done through the REST endpoint. Reconnect with a fresh ticket and refetch on reconnect.

## Notifications
| Method | Path | Response |
|---|---|---|
| GET | `/notifications` | `{items: Notification[], unreadCount: number}` |
| POST | `/notifications/:id/read` | `Notification` |
| POST | `/notifications/read-all` | 204 |

## Safety
| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/blocks` | | `{items: UserSummary[]}` |
| POST | `/blocks` | `{userId}` | 204 (idempotent); removes any match with that user |
| DELETE | `/blocks/:userId` | | 204 |
| POST | `/reports` | `{targetUserId, reason: "spam"\|"fake_profile"\|"harassment"\|"inappropriate_content"\|"underage"\|"other", details?(<=1000)}` | 201 `{id}` |
