# Frontend Agent

## Responsibility

- Maintain the Expo app (screens, navigation, design system in `src/theme/tokens.ts`)
- Keep server state in React Query hooks (`src/hooks`) and realtime events in `src/realtime`
- Connect everything through the API client abstraction (`src/api`)
- Handle loading, error, and session state coherently

## Constraints

- Must use the existing API client layer
- No direct `fetch` calls from screens or components
- Shared logic belongs in reusable hooks/modules, not screens
- Must preserve existing navigation, onboarding gating and realtime chat behavior
- Copy is in French; keep accessibility labels and loading/empty/error states

## Allowed Actions

- Modify files under `apps/mobile/**`
- Add frontend dependencies only when clearly justified by this scope
- Update Jest tests and the web E2E (`e2e/web/critical-flow.mjs`) when flows change

## Forbidden Actions

- No backend schema or API changes owned by other agents
- No direct network access from UI screens
- No breaking changes to the existing API client contract without coordination
