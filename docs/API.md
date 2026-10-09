# Lumo API reference

Base URL: `EXPO_PUBLIC_API_URL` (default `http://localhost:18080`). All bodies are JSON (`Content-Type: application/json`) except photo upload (`multipart/form-data`).
Authenticated endpoints need `Authorization: Bearer <accessToken>`. Access tokens live 15 minutes; renew them with `POST /auth/refresh` (refresh tokens rotate on every use).

## Conventions

- Errors: `{"error": "human readable message", "code": "machine_code"}` with a matching HTTP status.
  Common codes: `invalid_request` (400), `invalid_code` (400, bad/expired password-reset code), `photo_limit` (409), `unauthorized` (401), `forbidden` (403), `not_found` (404), `conflict` (409), `rate_limited` (429, with `Retry-After` header), `profile_incomplete` (403), `internal` (500).
- Ids are UUID strings. Timestamps are RFC 3339 UTC. Dates of birth are `YYYY-MM-DD`.
- Lists use `limit` (default 20, max 50) and `offset` unless stated otherwise.
- The API never returns e-mail addresses, birth dates, or coordinates of other users. Distances are approximate and rounded.

## Shared shapes

```ts
type Gender = "man" | "woman" | "nonbinary";

type Photo = { id: string; position: number; url: string };         // url is a path like "/photos/<id>/file" (needs Authorization header)

type PublicProfile = {
  id: string;                  // user id
  firstName: string;
  age: number;
  gender: Gender;
  bio: string;
  city: string;
  distanceKm: number | null;   // approximate (1, 2, 5, 10, 20, 50, 100, 200...), null if the user hides distance
  interests: Interest[];
  photos: Photo[];             // ordered, first is the main photo
};

type ProfileSummary = {        // used in lists (matches, conversations, likes)
  id: string; firstName: string; age: number; mainPhotoUrl: string | null;
};

type Interest = { slug: string; label: string };

type OnboardingState = { profile: boolean; photos: boolean; preferences: boolean; complete: boolean };

type MyProfile = {
  firstName: string; birthDate: string; age: number; gender: Gender; bio: string;
  city: string; latitude: number; longitude: number;   // coordinates are rounded to 0.01° server-side
  interests: Interest[]; photos: Photo[];
  discoverable: boolean; showDistance: boolean;
};

type Preferences = { interestedIn: Gender[]; minAge: number; maxAge: number; maxDistanceKm: number };
```

## Auth

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| POST | `/auth/register` | `{email, password}` (password 8-72 characters; the 72-byte cap is bcrypt's limit) | 201 `AuthResponse` |
| POST | `/auth/login` | `{email, password}` | 200 `AuthResponse` |
| POST | `/auth/refresh` | `{refreshToken}` | 200 `AuthResponse` |
| POST | `/auth/logout` | `{refreshToken}` | 204 |
| POST | `/auth/password/forgot` | `{email}` | 202 `{}` always (no account enumeration). A one-time 8-character code is e-mailed |
| POST | `/auth/password/reset` | `{email, code, newPassword}` | 204. Revokes every session. 400 `invalid_code` on a bad/expired code |
| GET | `/me` | | 200 `{id, email, createdAt, onboarding: OnboardingState}` |
| POST | `/me/password` | `{currentPassword, newPassword}` | 204 |
| DELETE | `/me` | `{password}` | 204. Permanently deletes the account, photos, matches, messages |

`AuthResponse = {accessToken, refreshToken, user: {id, email, createdAt}}`

## Profile, preferences, interests

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/interests` | | `{interests: Interest[]}` (curated catalogue) |
| GET | `/me/profile` | | 200 `MyProfile`, or 404 `not_found` before onboarding |
| PUT | `/me/profile` | `{firstName, birthDate, gender, bio?, city?, latitude, longitude, interests: string[] (slugs, max 10), discoverable?, showDistance?}` | 200 `MyProfile`. Validation: age 18-99, firstName 1-40 letters, bio ≤ 500 chars |
| GET | `/me/preferences` | | 200 `Preferences`; before the first save it returns `interestedIn: []`, ages 18-60 and 50 km (the user must pick `interestedIn`) |
| PUT | `/me/preferences` | `Preferences` (minAge ≥ 18, maxAge ≤ 99, minAge ≤ maxAge, maxDistanceKm 1-500, interestedIn non-empty) | 200 `Preferences` |

## Photos

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/me/photos` | | `{photos: Photo[]}` |
| POST | `/me/photos` | multipart field `file` (JPEG/PNG/WebP, ≤ 8 MB) | 201 `Photo`. Max 6 photos. 409 `photo_limit` when full |
| PUT | `/me/photos/:id` | multipart field `file` | 200 `Photo` (replaces image, keeps position) |
| PUT | `/me/photos/order` | `{ids: string[]}` (every photo id exactly once; first becomes the main photo) | 200 `{photos: Photo[]}` |
| POST | `/me/photos/:id/main` | | 200 `{photos: Photo[]}` |
| DELETE | `/me/photos/:id` | | 204 |
| GET | `/photos/:id/file` | | image bytes (re-encoded JPEG). Requires auth; 404 if the viewer may not see it |

## Discovery and matching

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/discover?limit=10` | | `{profiles: PublicProfile[]}` – profiles not yet swiped on, compatible both ways (gender, age, distance), never blocked. Call again after swiping to get the next batch. 403 `profile_incomplete` until onboarding is complete |
| GET | `/profiles/:userId` | | `PublicProfile` (404 if blocked/unknown/hidden) |
| POST | `/swipes` | `{userId, action: "like" \| "pass"}` | 200 `{matched: boolean, alreadySwiped: boolean, match: null \| {id, conversationId}}`. Idempotent: repeating a swipe never duplicates |
| GET | `/likes/received?limit=&offset=` | | `{profiles: PublicProfile[]}` – people who liked you and you have not swiped on yet |
| GET | `/matches?limit=&offset=` | | `{matches: [{id, createdAt, conversationId, user: ProfileSummary}]}` |
| DELETE | `/matches/:id` | | 204 (unmatch: removes the conversation for both, profiles never reappear) |

## Chat

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/conversations?limit=&offset=` | | `{conversations: Conversation[]}` ordered by last activity |
| GET | `/conversations/:id/messages?before=<messageId>&limit=30` | | `{messages: Message[], hasMore: boolean}` – oldest first inside the page; use `before` = id of the oldest loaded message to go back in time |
| POST | `/conversations/:id/messages` | `{body}` (1-2000 chars, trimmed) | 201 `Message` |
| POST | `/conversations/:id/read` | | 204 – marks everything received as read |
| DELETE | `/conversations/:id` | | 204 – hides the conversation and clears history for *you* only; it reappears when a new message arrives |

```ts
type Message = { id: string; conversationId: string; senderId: string; body: string; createdAt: string; readAt: string | null };
type Conversation = {
  id: string; matchId: string; user: ProfileSummary;
  lastMessage: { id: string; body: string; senderId: string; createdAt: string } | null;
  unreadCount: number; updatedAt: string;
};
```
Non-participants receive 404. Blocked pairs cannot send (403 `forbidden`).

## Safety

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/blocks` | | `{blocks: [{user: ProfileSummary, createdAt}]}` |
| POST | `/blocks` | `{userId}` | 204. Also unmatches and hides each other everywhere |
| DELETE | `/blocks/:userId` | | 204 |
| POST | `/reports` | `{userId, reason: "spam" \| "fake" \| "harassment" \| "inappropriate_photos" \| "underage" \| "other", details?: string (≤ 1000)}` | 201 `{id}` |

## Notifications

| Method | Path | Response |
| --- | --- | --- |
| GET | `/notifications?limit=&offset=` | `{notifications: Notification[], unreadCount: number}` |
| POST | `/notifications/:id/read` | 204 |
| POST | `/notifications/read-all` | 204 |

`Notification = {id, type: "match" | "message", title, body, data: {matchId?, conversationId?, userId?}, isRead, readAt, createdAt}`

## Real time (WebSocket)

1. `POST /ws/ticket` (authenticated) → `{ticket}` – single-use, valid 30 s.
2. Open `GET /ws?ticket=<ticket>` as a WebSocket (`ws://` / `wss://` + same host as the API).
3. Server pushes JSON frames `{type, data}`:
   - `message.new` – `data: Message` (also delivered to the sender's other devices)
   - `message.read` – `data: {conversationId, readerId, readAt}`
   - `match.new` – `data: {matchId, conversationId, user: ProfileSummary}`
   - `match.removed` – `data: {matchId, conversationId}`
   - `notification.new` – `data: Notification`
   The client sends nothing (frames from the client are ignored; the server pings every 25 s). Reconnect with a fresh ticket on close and refetch lists to catch up.

## Health

`GET /health` → `{"status":"ok"}` (503 if the database is unreachable).
