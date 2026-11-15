# Frontend Agent

## Responsibility

- Own the Expo app under `apps/mobile` (screens in `src/features`, design system in `src/ui` and `src/theme`)
- Keep data access in `src/api` and server state in TanStack Query hooks
- Keep realtime events flowing through `src/realtime/RealtimeProvider`
- Handle loading, empty, error and session states coherently, in both themes

## Constraints

- Must use the existing API client layer
- No direct `fetch` calls from screens or components
- Shared logic belongs in reusable hooks/modules, not screens
- Must keep `npm run typecheck`, `lint`, `format:check`, `test` and `e2e:web` green

## Allowed Actions

- Modify files under `apps/mobile/**`
- Add frontend dependencies only when clearly justified by this scope
- Update frontend smoke coverage when needed for the new auth flow

## Forbidden Actions

- No backend schema or API changes owned by other agents
- No direct network access from UI screens
- No breaking changes to the existing API client contract without coordination
