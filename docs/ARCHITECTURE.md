# Architecture

Lueur is a monorepo: a Go API (`services/api`), a PostgreSQL database, and an Expo / React Native client
(`apps/mobile`) that targets iOS, Android and the web. This document explains the data model, the core
algorithms, the realtime protocol and the security model. The README covers installation and operations.

## Backend layering

```text
HTTP (Gin) ──► handler ──► service ──► repository ──► PostgreSQL (pgxpool)
                 │            │
         validation +     business rules,
         status codes     orchestration, events
```

- **handler** — binds input, maps domain errors to HTTP status codes and stable error `code`s, nothing else.
- **service** — all business rules (age gate, preference checks, limits, normalisation, events).
- **repository** — explicit SQL only; transactions and locks live here. No ORM, no implicit magic.
- **platform** — shared concerns: `config`, `db` (pool + migrations), `middleware` (auth, rate limit, CORS,
  security headers, request logging), `httpx` (response helpers), `realtime` (WebSocket hub), `mailer`,
  `signing` (signed media URLs), `logger`.
- Features depend on each other only through small interfaces (for example `matching.ProfileReader`).
  `cmd/api/app.go` is the single place where everything is wired.

## Data model

```text
users 1─1 profiles 1─1 preferences          profiles.gender ∈ woman|man|non_binary
  │            └─ n─n interests (user_interests)
  │ 1─n photos (position 0..5, UNIQUE(user_id, position) DEFERRABLE)
  │ 1─n sessions (hashed refresh token)      1─n password_reset_tokens (hashed)
  ├─ swipes(swiper_id, target_id, action like|pass)   PRIMARY KEY (swiper_id, target_id)
  ├─ matches(user_a < user_b)   UNIQUE(user_a, user_b)
  │     └─ 1─1 conversations(match_id) ─ n messages(sender, recipient, content, read_at)
  │                          └─ conversation_participants(conversation_id, user_id, hidden_at)
  ├─ blocks(blocker_id, blocked_id)
  ├─ reports(reporter_id, reported_id → NULL on deletion, reason, status)
  └─ notifications(user_id, type, title, body, data jsonb, is_read)
```

- Every user-owned table references `users(id) ON DELETE CASCADE`: deleting an account removes the profile,
  photos (files are removed from storage by the application), swipes, matches, conversations, messages and
  notifications in one statement. `reports` keep their rows with the user ids set to `NULL`.
- Required indexes: `users.email` (unique), `swipes(target_id, action)`, `matches(user_a|user_b, created_at)`,
  `messages(conversation_id, created_at, id)`, partial index on unread messages, `notifications(user_id, created_at)`,
  `photos(user_id, position)`, discovery index `profiles(gender, latitude, longitude) WHERE is_discoverable`.
- Constraints enforce invariants even if application code is wrong: age ≥ 18 in preferences, ordered match pairs,
  ranges on coordinates, text length limits, enumerations for gender/action/report reason.
- `interests` is seeded in a migration; labels are French, slugs are stable identifiers.
- The legacy tables of the original boilerplate (`posts`, `comments`, `votes`, `files`, `organizations`,
  `subscriptions`) still exist because early migrations create them; nothing reads or writes them.

### Migrations

`schema_migrations(version)` records applied files. At boot the API takes an advisory lock, applies every
unapplied file in lexical order inside its own transaction, and releases the lock. Add a file named
`NNN_description.sql` to evolve the schema; never rewrite an applied migration.

## Discovery and matching

`matching.Repository.Discover` returns the next candidates in one query. A candidate `c` is proposed to viewer
`me` only if **both** sides are satisfied (this is the shared `candidateFilter` SQL fragment):

1. `c` is discoverable, has a location and at least one photo;
2. `c.gender ∈ me.interested_in` **and** `me.gender ∈ c.interested_in`;
3. `c.age ∈ [me.min_age, me.max_age]` **and** `me.age ∈ [c.min_age, c.max_age]`;
4. distance ≤ `min(me.max_distance_km, c.max_distance_km)` (bounding-box prefilter on latitude, then haversine);
5. no block in either direction; and `me` has not already liked or passed `c`.

Ordering (the only place to change the recommendation algorithm):
people who already liked `me` → number of shared interests → distance → recent activity.

`POST /swipes` re-applies the same filter in SQL (plus the exception "someone who already liked you stays
likeable"), so a forged request cannot reach a profile outside the viewer's criteria. In one transaction it
takes a per-pair advisory lock, inserts the swipe (`PRIMARY KEY` ⇒ duplicates return `409 already_swiped`), and if
the other person's like exists creates the match, the conversation, both participants and both notifications.
The lock makes two simultaneous reciprocal likes see each other, so exactly one match is created (covered by a
concurrency test).

## Chat

A conversation exists only for a match. Every operation resolves membership with the same query (`accessSQL`):
the user must be a participant **and** no block may exist; otherwise the API answers `404` (it never confirms that a
conversation exists). Messages are stored with `recipient_user_id` so unread counts are a single indexed query.
Opening a conversation does not mark it read; the client calls `POST …/read` while it is visible.
"Delete conversation" sets `hidden_at` for the caller only: older messages are hidden for them, the other person is
unaffected, and a new message makes the conversation reappear without the cleared history.

## Realtime

`GET /ws` upgrades to a WebSocket. The first client frame must be `{"type":"auth","token":"<access token>"}`
within 5 seconds (tokens never appear in URLs or logs). The server answers `{"type":"ready"}`, delivers events only
to the authenticated user's sockets, and closes the socket when the access token expires or the account is deleted.
Events: `match.new`, `match.removed`, `message.new`, `messages.read`, `notification.new`.

The client (`apps/mobile/src/realtime/client.ts`) reconnects with exponential backoff and jitter (1 s → 30 s), refreshes
the token on close code `4401`, and after every reconnect invalidates its query cache to catch up on missed events.
The hub is in-process; see the README for how to scale out.

## Photos

Upload (`multipart/form-data`, field `photo`, ≤ 8 MB) → content sniffing (JPEG/PNG only) → pixel limits checked
before decoding → decode → apply EXIF orientation → downscale to ≤ 1440 px → flatten transparency → re-encode as
JPEG (drops all metadata, including GPS, and any appended payload) → store under a random key
(`Storage` interface; `LocalStorage` writes mode 0600 files). Images are served at
`/photos/:id/image?exp&sig` where `sig = HMAC(key, id|exp)`; the API puts such URLs in profile payloads only for
viewers it authorised, they last 1–2 hours and are stable within the hour so client image caches work.

## Security model

- **Authentication**: access JWT (HS256, 15 min, `uid` + `sid` claims, algorithm pinned, `exp` mandatory) plus an
  opaque refresh token (256-bit, stored hashed, single use, rotated; a rotation only succeeds if the presented hash is
  still current). Every authenticated request verifies that its session is active, so logout, password change/reset
  and account deletion revoke access immediately.
- **Passwords**: bcrypt, 8–72 bytes, dummy hash comparison for unknown emails (no timing oracle). Recovery tokens are
  random, stored hashed, expire in 1 hour, are single use, and the endpoint answers `202` whether or not the email exists.
- **Authorisation**: derived from the token only (`userID` from claims); no endpoint accepts a user id as the actor.
- **Input**: JSON bodies capped at 1 MiB, strict binding + service-level normalisation (trim, control characters,
  lengths, enumerations, UUID path params), bounded pagination.
- **Rate limits** (abuse guards, per instance): global per IP, credentials per IP, swipes/messages/uploads/reports per
  user. The client IP comes from the TCP peer unless `TRUSTED_PROXIES` lists the reverse proxy.
- **Privacy**: coordinates rounded to ≈1 km at write time and never serialised; distances in 5 km steps; birth date
  never exposed (age only); email only in the owner's `/me`.
- **Headers/CORS**: `nosniff`, frame denial, no-referrer, HSTS on HTTPS, CORS allow-list from configuration.
- **Logs** are structured and redact bearer tokens, JWTs, passwords and refresh tokens.

## Client architecture (apps/mobile)

- `src/api` — `ApiClient` (timeouts, single-flight refresh, typed errors) and `platform.ts`, the only module screens
  use to talk to the API.
- State: TanStack Query for server state (keys in `realtime/cache.ts`), React context for auth, a tiny external
  store for toasts. No global mutable app state otherwise.
- Navigation: signed out → auth stack; signed in without a complete profile → onboarding; otherwise tabs
  (Découvrir, Messages, Profil) with detail screens on a native stack.
- Design system: `theme/` (tokens, light/dark following the system) and `shared/ui` components; contrast ratios are
  asserted in `theme/theme.test.ts`.
- Performance: cached images (`expo-image`), paginated lists, deck refilled in batches of 10, optimistic swipes,
  realtime updates patch the query cache instead of refetching whole lists.
- Accessibility: roles/labels on every control, button alternatives to the swipe gesture, reduced-motion support,
  live regions for errors and toasts, minimum 44 px targets.
