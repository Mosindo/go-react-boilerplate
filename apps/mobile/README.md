# Lumo mobile (Expo 54 / React Native 0.81 / web)

Free dating app front-end. Talks to the Go API described in `docs/API.md`.

## Setup

```bash
cp .env.example .env     # EXPO_PUBLIC_API_URL: localhost:18080 (simulator/web), 10.0.2.2:18080 (Android emulator), http://<LAN_IP>:18080 (phone, never localhost)
npm ci
npm start                # or: npm run web / npm run android / npm run ios
```

The API must allow the web origin in `ALLOWED_ORIGINS` when running the web build.

## Quality gates

| Command                                  | What                                                                                                                                                                                                                                                                  |
| ---------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `npx tsc --noEmit` (`npm run typecheck`) | strict TypeScript                                                                                                                                                                                                                                                     |
| `npm run lint` / `npm run format`        | ESLint (flat) + typescript-eslint + react-hooks, Prettier check / fix                                                                                                                                                                                                 |
| `npm test`                               | Jest (jest-expo): API client + token refresh, validators, realtime client/cache updaters, components                                                                                                                                                                  |
| `npm run e2e:web`                        | exports the web build against `E2E_API_URL` (default `http://localhost:18080`) and runs the Playwright two-user journey (`e2e/lumo.spec.ts`). Needs a running API with `ALLOWED_ORIGINS=http://localhost:19081` (override the static server port with `E2E_WEB_PORT`) |

Layout: `src/api` (HTTP client with single-flight refresh, one module per area), `src/realtime` (WebSocket client + react-query cache updaters), `src/screens`, `src/shared/ui` (design system), `src/i18n/fr.ts` (all strings), `src/theme` (tokens, light/dark).
