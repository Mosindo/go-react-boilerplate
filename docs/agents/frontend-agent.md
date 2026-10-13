# Frontend Agent

## Responsibility

- Integrate React Query into the mobile frontend
- Implement auth hooks such as `useAuth` (sign in, sign up, sign out, delete account)
- Connect hooks through the existing API client abstraction
- Handle loading, error, and session state coherently

## Constraints

- Must use the existing API client layer
- No direct `fetch` calls from screens or components
- Shared logic belongs in reusable hooks/modules, not screens
- Must preserve existing navigation and working chat behavior

## Allowed Actions

- Modify files under `apps/mobile/**`
- Add frontend dependencies only when clearly justified by this scope
- Update frontend smoke coverage when needed for the new auth flow

## Forbidden Actions

- No backend schema or API changes owned by other agents
- No direct network access from UI screens
- No breaking changes to the existing API client contract without coordination
