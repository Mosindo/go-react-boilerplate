# API reference

Base URL: `EXPO_PUBLIC_API_URL` (default `http://localhost:18080`). JSON everywhere (camelCase) except photo upload (`multipart/form-data`).
Authenticated routes need `Authorization: Bearer <accessToken>` (15 min). Refresh with `POST /auth/refresh`.

Errors: HTTP status + `{"error": "human readable message", "code": "machine_code"}`.
Common codes: `invalid_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `rate_limited`, `profile_required`,
`profile_not_found`, `invalid_credentials`, `email_taken`, `underage`, `photo_limit`, `invalid_image`, `blocked`.

List endpoints take `limit` (default 20, max 50) and, when paginated, an opaque `cursor`; responses carry `nextCursor` (string or `null`).

## Shared shapes

```ts
Interest      { id: number; slug: string; label: string }
Photo         { id: string; position: number; isPrimary: boolean; url: string }   // url is relative to the API base URL, signed + expiring, no auth header needed
PublicProfile { userId: string; firstName: string; age: number; gender: Gender; bio: string; city: string;
                interests: Interest[]; photos: Photo[]; distanceKm: number | null; likedYou?: boolean }
Gender        "man" | "woman" | "non_binary"
Message       { id: string; conversationId: string; senderId: string; body: string; createdAt: string }
Conversation  { id: string; matchId: string; user: { userId: string; firstName: string; photoUrl: string | null };
                lastMessage: { body: string; senderId: string; createdAt: string } | null;
                unreadCount: number; createdAt: string; lastActivityAt: string }
Notification  { id: string; type: "new_like" | "new_match"; title: string; body: string; data: Record<string, string>;
                isRead: boolean; createdAt: string }
```

`distanceKm` is approximate (rounded, never exact) and `null` when unknown or hidden by the other user.

## Auth

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/auth/register` | `{email, password}` (8–72 chars) | 201 `{accessToken, refreshToken, user}` |
| POST | `/auth/login` | `{email, password}` | 200 same |
| POST | `/auth/refresh` | `{refreshToken}` | 200 same (token rotated) |
| POST | `/auth/logout` | `{refreshToken}` | 204 |
| POST | `/auth/forgot-password` | `{email}` | 202 always (no account enumeration) |
| POST | `/auth/reset-password` | `{token, password}` | 204 (revokes all sessions) |
| GET | `/me` | – | `{id, email, createdAt, profileComplete}` |
| POST | `/me/password` | `{currentPassword, newPassword}` | 204 |
| DELETE | `/me` | `{password}` | 204 – deletes the account and all associated data |

`user` = `{id, email, createdAt, profileComplete}`.

## Profile, preferences, privacy

| Method | Path | Notes |
|---|---|---|
| GET | `/interests` | `{interests: Interest[]}` (catalogue) |
| GET | `/me/profile` | `OwnProfile` or 404 `profile_not_found` |
| PUT | `/me/profile` | create/update: `{firstName, birthDate:"YYYY-MM-DD", gender, bio, city, interestIds:number[]}`. 18+ only (`underage`). `birthDate` can't change once set. Max 10 interests, bio ≤ 500, firstName 1–40. |
| PUT | `/me/location` | `{latitude, longitude}` → 204. Stored rounded to ~1 km. |
| DELETE | `/me/location` | 204 – forget location |
| GET/PUT | `/me/preferences` | `{interestedIn: Gender[], minAge, maxAge, maxDistanceKm: number \| null}` (18 ≤ minAge ≤ maxAge ≤ 99; distance 1–500 or null = anywhere). Defaults are created with the profile. |
| GET/PUT | `/me/privacy` | `{discoverable: boolean, showDistance: boolean}` |
| GET | `/profiles/:userId` | `PublicProfile`; 404 if blocked / hidden / unknown |

`OwnProfile` = `PublicProfile` fields (without distance) + `birthDate`, `discoverable`, `showDistance`, `hasLocation`, `preferences`.

## Photos (max 6, JPEG or PNG ≤ 8 MB; server re-encodes to JPEG ≤ 1280 px and strips metadata)

| Method | Path | Notes |
|---|---|---|
| POST | `/me/photos` | multipart field `file` → 201 `Photo` (`photo_limit` at 6) |
| PUT | `/me/photos/:id` | multipart field `file`, replaces the image, keeps position |
| PUT | `/me/photos/order` | `{photoIds: string[]}` (must be the exact set) → `{photos: Photo[]}`. First = primary. |
| DELETE | `/me/photos/:id` | 204 |
| GET | `/photos/:id/content?exp&sig` | image bytes (signed URL from `Photo.url`) |

A profile needs ≥ 1 photo to appear in discovery (`profileComplete` in `/me`).

## Discovery, swipes, matches

| Method | Path | Notes |
|---|---|---|
| GET | `/discover?limit=10` | `{profiles: PublicProfile[]}` – never contains already-swiped, blocked, hidden or non-compatible people. Call again after swiping to get the next batch. |
| POST | `/swipes` | `{userId, action:"like"\|"pass"}` → `{action, matched, conversation: Conversation \| null}`. Idempotent: repeating a like never duplicates. A pass can be upgraded to a like later. |
| GET | `/likes/received` | `{profiles: PublicProfile[]}` people who liked you and you haven't answered (free, unlimited) |
| DELETE | `/matches/:matchId` | unmatch: 204, deletes the conversation for both |

## Chat

| Method | Path | Notes |
|---|---|---|
| GET | `/conversations?cursor` | `{conversations: Conversation[], nextCursor}` ordered by `lastActivityAt` desc; hidden ones excluded |
| DELETE | `/conversations/:id` | hide locally (reappears on new message) → 204 |
| GET | `/conversations/:id/messages?before=<messageId>&limit=30` | `{messages: Message[] (newest first), nextCursor, otherLastReadAt: string \| null}` |
| POST | `/conversations/:id/messages` | `{body}` (1–2000) → 201 `Message` |
| POST | `/conversations/:id/read` | 204 |

Non-participants get 404.

## Notifications

`GET /notifications?cursor` → `{notifications, unreadCount, nextCursor}` · `POST /notifications/:id/read` 204 · `POST /notifications/read-all` 204

## Safety

| Method | Path | Notes |
|---|---|---|
| POST | `/blocks` | `{userId}` → 204; removes the match/conversation, hides both users from each other |
| DELETE | `/blocks/:userId` | 204 |
| GET | `/blocks` | `{blocks: {userId, firstName, createdAt}[]}` |
| POST | `/reports` | `{userId, reason, details?}` → 201 `{id}`. reasons: `spam fake_profile inappropriate_content harassment underage scam other` |

## Realtime

1. `POST /ws/ticket` (auth) → `{ticket, expiresInSeconds: 30}` (single use)
2. Open `GET /ws?ticket=<ticket>` as a WebSocket.

Server → client frames `{ "type": string, "data": object }`:
`message.new` (Message) · `message.read` `{conversationId, readerId, readAt}` · `match.new` (Conversation) ·
`notification.new` (Notification) · `conversation.removed` `{conversationId}`.
The client may send `{"type":"ping"}` (answered with `pong`). On reconnect, refetch conversations.

## Health

`GET /health` → `{status:"ok"}`
