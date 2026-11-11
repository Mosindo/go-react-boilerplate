# Frontend Agent

## Responsibility

- Maintain the React Query layer (`src/hooks/queries.ts`) and realtime cache updates (`src/hooks/useRealtime.tsx`)
- Maintain session handling in `src/hooks/useAuth.tsx`
- Connect hooks through the existing API client abstraction
- Handle loading, error, and session state coherently

## Constraints

- Must use the existing API client layer
- No direct `fetch` calls from screens or components
- Shared logic belongs in reusable hooks/modules, not screens
- Must preserve existing navigation, the free (no paywall) product rules and working chat behavior

## Allowed Actions

- Modify files under `apps/mobile/**`
- Add frontend dependencies only when clearly justified by this scope
- Update frontend smoke coverage when needed for the new auth flow

## Forbidden Actions

- No backend schema or API changes owned by other agents
- No direct network access from UI screens
- No breaking changes to the existing API client contract without coordination
