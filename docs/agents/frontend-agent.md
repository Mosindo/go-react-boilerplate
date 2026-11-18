# Frontend Agent

## Responsibility

- Maintain the Expo app (`apps/mobile`): screens, design system, navigation
- Keep server state in React Query hooks (`src/hooks/queries.ts`) and session state in `useAuth`
- Keep realtime events flowing into the query cache (`src/hooks/useRealtime.tsx`)
- Handle loading, empty, error and success feedback on every screen

## Constraints

- Must use the existing API client layer
- No direct `fetch` calls from screens or components
- Shared logic belongs in reusable hooks/modules, not screens
- Must keep `npm run typecheck`, `npm run lint`, `npm run format:check` and `npm run e2e:web` green

## Allowed Actions

- Modify files under `apps/mobile/**`
- Add frontend dependencies only when clearly justified by this scope
- Update frontend smoke coverage when needed for the new auth flow

## Forbidden Actions

- No backend schema or API changes owned by other agents
- No direct network access from UI screens
- No breaking changes to the existing API client contract without coordination
