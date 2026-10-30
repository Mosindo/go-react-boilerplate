# API

JSON over HTTP, authentication by `Authorization: Bearer <accessToken>` unless stated. Errors: `{"error": "message", "field": "optional"}`.
Pagination is explicit (`limit`, and `before=<RFC3339 timestamp>` cursors; the response carries `nextBefore` when more data exists).

| Method | Path | Auth | Purpose |
| --- | --- | --- | --- |
| GET | `/health` | – | Liveness + DB check |
| POST | `/auth/register` | – | Create account (`email`, `password` 10–72 chars). Rate limited |
| POST | `/auth/login` | – | Returns `accessToken`, `refreshToken`, `user`. Rate limited |
| POST | `/auth/refresh` | – | Rotates the refresh token |
| POST | `/auth/logout` | – | Revokes the refresh token |
| POST | `/auth/forgot` | – | Sends a reset code by email (always 202) |
| POST | `/auth/reset` | – | `token` + new `password`; revokes all sessions |
| GET | `/me` | ✔ | Current account |
| PUT | `/me/password` | ✔ | Change password; revokes all sessions |
| DELETE | `/me` | ✔ | Delete account (`password`) and all its data |
| GET / PUT | `/profile` | ✔ | Own profile (404 until created). `PUT` is a full update; coordinates optional after creation |
| GET / PUT | `/preferences` | ✔ | `interestedIn[]`, `minAge`, `maxAge`, `maxDistanceKm` |
| GET | `/interests` | ✔ | Interest catalog |
| GET | `/profiles/:id` | ✔ | Another member's profile (only if visible to the caller) |
| GET | `/photos` | ✔ | Own photos |
| POST | `/photos` | ✔ | Multipart field `file` (JPEG/PNG, ≤ 8 MB, max 6 photos) |
| PUT | `/photos/:id` | ✔ | Replace a photo (multipart `file`) |
| DELETE | `/photos/:id` | ✔ | Delete a photo |
| PUT | `/photos/order` | ✔ | `photoIds[]`, first = main photo |
| GET | `/photos/:id/file?exp&sig` | signed URL | Image bytes |
| GET | `/discover?limit=` | ✔ | Next candidates (409 if profile incomplete) |
| POST | `/swipes` | ✔ | `targetUserId`, `action` = `like` \| `pass`; returns `matched`, `duplicate`, `match` |
| DELETE | `/matches/:id` | ✔ | Unmatch |
| GET | `/conversations` | ✔ | Conversations with last message and unread count |
| GET / POST | `/conversations/:id/messages` | ✔ | History (chronological page) / send (max 2000 chars) |
| POST | `/conversations/:id/read` | ✔ | Mark as read |
| DELETE | `/conversations/:id` | ✔ | Hide for the caller only |
| GET | `/events?after=` | ✔ | Long poll (≤ 25 s) for realtime signals |
| GET | `/notifications` | ✔ | List + `unreadCount` |
| POST | `/notifications/:id/read`, `/notifications/read-all` | ✔ | Mark as read |
| GET / POST | `/blocks` | ✔ | List / block (`userId`) |
| DELETE | `/blocks/:userId` | ✔ | Unblock |
| POST | `/reports` | ✔ | `userId`, `reason`, `details`, `block` |
