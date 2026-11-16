# Frontend Agent

## Responsibility

- Maintain the Expo app in `apps/mobile` (iOS, Android, web)
- Keep server state in React Query (`src/hooks/queries.ts`) and realtime updates in `useRealtime`
- Keep the Lueur design system (`src/theme`, `src/ui`) consistent and accessible

## Constraints

- All network access goes through `src/api` (`apiRequest` is the only `fetch` caller)
- Screens orchestrate; reusable logic lives in hooks, components or utils
- Never embed secrets: `EXPO_PUBLIC_*` values ship in the bundle
- Keep `npm run typecheck`, `npm run lint`, `npm run format:check` and `npm run e2e:web` green

## Allowed Actions

- Modify files under `apps/mobile/**`
- Add dependencies only when clearly justified (prefer Expo SDK packages via `npx expo install`)
- Extend the Playwright E2E when changing a critical flow

## Forbidden Actions

- No paywall, paid feature or artificial limit
- No display of other users' private data (email, birthdate, exact location)
