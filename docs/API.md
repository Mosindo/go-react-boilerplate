# API reference

REST + JSON (camelCase) over HTTP, plus one WebSocket for server-to-client events.
Base URL: `EXPO_PUBLIC_API_URL` on the client (`http://<LAN_IP>:18080` with the default Docker Compose).

All endpoints except `/health`, `/auth/*` require `Authorization: Bearer <accessToken>`.
Access tokens live 15 minutes; use `/auth/refresh` (refresh tokens rotate on every use).

## Errors

Every non-2xx response is `{ "error": "human readable message", "code": "machine_code" }`.

| Status | Meaning |
| --- | --- |
| 400 | invalid input (`code: invalid_request`, `validation_failed`) |
| 401 | missing/invalid/expired token or wrong credentials |
| 403 | forbidden (blocked, suspended account, not a member of the match) |
| 404 | not found (also used instead of 403 when existence must not leak) |
| 409 | conflict (`email_taken`, `photo_limit`) |
| 413 | upload too large |
| 415 | unsupported media type |
| 429 | rate limited, `Retry-After` header set |

Validation errors may include `"fields": { "firstName": "required" }`.

## Shared shapes

```ts
type Gender = "man" | "woman" | "non_binary";
type Photo = { id: string; position: number; url: string; thumbUrl: string; width: number; height: number };
type Interest = { slug: string; label: string };
type Message = { id: string; matchId: string; senderId: string; content: string; createdAt: string; readAt: string | null };
type Card = {                       // what other people see
  userId: string; firstName: string; age: number; gender: Gender;
  bio: string; city: string;
  distanceKm: number | null;        // bucketed (5, 10, 15 ...) – null when hidden or unknown
  interests: (Interest & { shared: boolean })[];
  photos: Photo[];
};
type MatchSummary = {
  id: string; createdAt: string; lastActivityAt: string; unreadCount: number;
  user: { userId: string; firstName: string; thumbUrl: string | null };
  lastMessage: { content: string; senderId: string; createdAt: string } | null;
};
type Notification = {
  id: string; type: "match" | "message"; title: string; body: string;
  data: { matchId?: string; userId?: string }; readAt: string | null; createdAt: string;
};
```

`url` / `thumbUrl` are relative paths (`/photos/{id}`, `/photos/{id}?size=thumb`). Photos are **not public**: load them
with the `Authorization` header (`<Image source={{ uri: API_BASE_URL + url, headers: { Authorization } }} />`).

## Auth

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| POST | `/auth/register` | `{ email, password }` | 201 `AuthResponse` |
| POST | `/auth/login` | `{ email, password }` | 200 `AuthResponse` |
| POST | `/auth/refresh` | `{ refreshToken }` | 200 `AuthResponse` |
| POST | `/auth/logout` | `{ refreshToken }` | 204 |
| POST | `/auth/forgot-password` | `{ email }` | 202 always (no account enumeration) |
| POST | `/auth/reset-password` | `{ email, code, password }` | 204, revokes all sessions |

```ts
type AuthResponse = { accessToken: string; refreshToken: string; user: { id: string; email: string; createdAt: string } };
```

Password: 8–72 characters with at least one letter and one digit.
The reset `code` is emailed (SMTP) or, when SMTP is not configured, printed in the API log (development only).

## Account

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/me` | – | `{ id, email, createdAt, onboarding: { hasProfile, photoCount, completed } }` |
| DELETE | `/me` | `{ password }` | 204, permanently deletes the account and every associated row |

`onboarding.completed` is true when the profile exists and at least one photo is uploaded. Only completed
profiles appear in discovery.

## Profile, preferences, interests

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/interests` | – | `{ interests: Interest[] }` |
| GET | `/me/profile` | – | `Profile` (404 `profile_not_found` before onboarding) |
| PUT | `/me/profile` | `ProfileInput` | `Profile` (create or update) |
| GET | `/me/preferences` | – | `Preferences` |
| PUT | `/me/preferences` | `Preferences` | `Preferences` |

```ts
type ProfileInput = {
  firstName: string;            // 1-40 chars
  birthDate: string;            // YYYY-MM-DD, must be 18+ (server enforced)
  gender: Gender;
  bio: string;                  // <= 500 chars, may be empty
  city: string;                 // 1-80 chars
  latitude?: number | null;     // optional, stored rounded to ~2km
  longitude?: number | null;
  interests: string[];          // interest slugs, <= 10
};
type Profile = { userId: string; firstName: string; birthDate: string; age: number; gender: Gender; bio: string; city: string;
                 hasLocation: boolean; interests: Interest[]; photos: Photo[] };
type Preferences = {
  interestedIn: Gender[];       // non-empty
  minAge: number;               // 18..99
  maxAge: number;               // >= minAge, <= 99
  maxDistanceKm: number;        // 1..500
  discoverable: boolean;        // false = hidden from discovery (matches still chat)
  showDistance: boolean;        // false = others never see your distance
};
```

## Photos

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/me/photos` | `{ photos: Photo[] }` ordered, `position 0` is the main photo |
| POST | `/me/photos` | `multipart/form-data`, field `photo` (JPEG/PNG/WebP, ≤ 8 MB). 201 `Photo`. Max 6 photos (409 `photo_limit`) |
| PUT | `/me/photos/{id}` | same multipart body, replaces the image in place, keeps position |
| DELETE | `/me/photos/{id}` | 204, remaining photos are re-numbered |
| PUT | `/me/photos/order` | `{ photoIds: string[] }` (every id exactly once, first = main) → `{ photos }` |
| GET | `/photos/{id}?size=thumb\|full` | image bytes (`image/jpeg`), private cache headers. 404 if blocked/unknown |

Images are decoded, stripped of metadata (EXIF/GPS), resized (full ≤ 1080 px, thumb ≤ 320 px) and re-encoded as JPEG server-side.

## Discovery and matching

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/discover?limit=10` | – | `{ profiles: Card[] }` (1–20, default 10). Excludes yourself, already-swiped, blocked, hidden, suspended, and anyone whose preferences don't fit yours (mutual). Returns the same profiles until you swipe them. |
| GET | `/profiles/{userId}` | – | `Card` |
| POST | `/swipes` | `{ targetUserId, action: "like" \| "pass" }` | `{ action, alreadyProcessed, matched, match: MatchSummary \| null }` |
| GET | `/matches?limit=20&cursor=` | – | `{ matches: MatchSummary[], nextCursor: string \| null }` ordered by last activity |
| DELETE | `/matches/{id}` | – | 204 unmatch (both sides lose the conversation) |
| POST | `/matches/{id}/hide` | – | 204 removes the conversation from *your* list and history; it reappears when a new message arrives |

A swipe decision is final and idempotent: repeating it returns `alreadyProcessed: true` without side effects.

## Chat

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/matches/{id}/messages?limit=30&cursor=` | – | `{ messages: Message[], nextCursor: string \| null }` newest first |
| POST | `/matches/{id}/messages` | `{ content }` (1–2000 chars, trimmed) | 201 `Message` |
| POST | `/matches/{id}/read` | – | 204 marks the other person's messages as read |

Only the two members can use these endpoints (404 otherwise).

## Moderation

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/blocks` | – | `{ blocks: { userId, firstName, createdAt }[] }` |
| POST | `/blocks` | `{ userId }` | 204, also unmatches |
| DELETE | `/blocks/{userId}` | – | 204 (does not restore the match) |
| POST | `/reports` | `{ userId, reason, details?, matchId?, block? }` | 201 |

`reason`: `spam`, `fake_profile`, `harassment`, `inappropriate_content`, `underage`, `other`. `details` ≤ 1000 chars.
Accounts reported by 5 distinct users are automatically suspended pending review.

## Notifications

| Method | Path | Response |
| --- | --- | --- |
| GET | `/notifications?limit=20&cursor=` | `{ notifications, unreadCount, nextCursor }` |
| POST | `/notifications/{id}/read` | 204 |
| POST | `/notifications/read-all` | 204 |

## Realtime (WebSocket)

`GET /ws` upgrades to a WebSocket (`ws://` / `wss://`). The first frame sent by the client must be
`{"type":"auth","token":"<accessToken>"}` within 5 seconds (tokens are never put in URLs). The server answers `{"type":"ready"}`.
Clients may send `{"type":"ping"}` (answered with `{"type":"pong"}`); reconnect with backoff when the socket closes.
After a reconnect refetch `/matches` and `/notifications`.

Server events: `{ "type": string, "data": object }`

| type | data |
| --- | --- |
| `message.new` | `Message` |
| `message.read` | `{ matchId, readerId, readAt }` |
| `match.new` | `MatchSummary` |
| `match.removed` | `{ matchId }` |
| `notification.new` | `Notification` |
