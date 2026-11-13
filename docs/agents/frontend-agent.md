# Frontend Agent

## Responsibility

- Maintain the Expo app screens under `apps/mobile/src/features`
- Keep server state in React Query hooks (`src/state/queries.ts`) and the session in `src/state/session.tsx`
- Connect everything through the API client abstraction in `src/api`
- Handle loading, error, and session state coherently

## Constraints

- Must use the existing API client layer
- No direct `fetch` calls from screens or components
- Shared logic belongs in reusable hooks/modules, not screens
- Must preserve existing navigation and working chat behavior
- Must keep `npm run check` and the Playwright critical flow green

## Allowed Actions

- Modify files under `apps/mobile/**`
- Add frontend dependencies only when clearly justified by this scope
- Update frontend smoke coverage when needed for the new auth flow

## Forbidden Actions

- No backend schema or API changes owned by other agents
- No direct network access from UI screens
- No breaking changes to the existing API client contract without coordination
