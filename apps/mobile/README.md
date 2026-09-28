# Lumen mobile app

Expo SDK 54 / React Native 0.81 / React 19 / TypeScript (strict) client for the Lumen dating API. Lumen is free: no
ads, no paywall, no subscriptions. The HTTP contract is [`docs/API.md`](../../docs/API.md).

## Configure the API address

Copy `.env.example` to `.env` and set `EXPO_PUBLIC_API_URL`.

> **Never use `localhost` on a physical phone.** It points at the phone itself. Use your computer's LAN IP and the host
> port published by Docker Compose (default `18080`): `EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080`.

Emulators only, when the variable is unset in development: Android `http://10.0.2.2:18080`, iOS simulator
`http://localhost:18080`. `EXPO_PUBLIC_*` values are embedded in the bundle, so never put secrets there.

```bash
npm ci
npm run start:phone   # detects the LAN IP, warns if /health is unreachable, starts Expo
# or: EXPO_PUBLIC_API_URL=http://192.168.1.42:18080 npx expo start
```

`start:phone` reads `API_PORT` (host port, default 18080) or a full `EXPO_PUBLIC_API_URL`.

Before calling the app ready, verify no other service hijacks the API port:
`docker compose -f ../../infra/docker-compose.yml ps` and `curl http://<LAN_IP>:18080/health`. If there is a conflict,
change the host port in `infra/docker-compose.yml` and align `src/api/config.ts`, `scripts/start-phone.js`, this file.

## Scripts

| Script                                    | What it does                                                                        |
| ----------------------------------------- | ----------------------------------------------------------------------------------- |
| `npm run typecheck`                       | `tsc --noEmit`                                                                      |
| `npm run lint`                            | ESLint (flat config, typescript-eslint, react-hooks), zero warnings allowed         |
| `npm test`                                | Jest (`jest-expo`) unit and component tests                                         |
| `npm run format:check` / `npm run format` | Prettier                                                                            |
| `npm run e2e:smoke`                       | Runs the real critical flow against a live API (`API_URL` or `EXPO_PUBLIC_API_URL`) |

The smoke script registers two users, completes profiles (location, preferences, generated JPEG), discovers, likes both
ways, matches, chats, reads, checks notifications and realtime, blocks, reports, unmatches and deletes both accounts.
It exits non-zero on any failure.

## Structure

- `src/theme/index.ts` - product name (`BRAND`), palettes (light/dark, follows the system), spacing, typography.
- `src/api/` - the only place that talks to the network: `client.ts` (typed client, error mapping, single-flight token
  refresh), `http.ts` (configured instance), one module per API area, `realtime.ts` (WebSocket client).
- `src/domain/` - pure logic with unit tests (age validation, distance labels, chat grouping, swipe deck state machine,
  realtime cache reducers).
- `src/hooks/` - react-query hooks, auth session, realtime bridge.
- `src/navigation/`, `src/screens/`, `src/components/`, `src/shared/` - UI.

Tokens are stored with `expo-secure-store`. Access tokens (15 min) are refreshed transparently on 401 or before the
WebSocket connects, and concurrent requests share one refresh so the rotating refresh token is never reused.

## Onboarding and location

A profile is complete once it has a profile row, a location and at least one photo. Location is required for
discovery: the coordinates always come from the device (permission prompt). The city label is display-only; if reverse
geocoding fails the user types it. If permission is denied the app explains why it is needed and offers a retry or the
system settings.
