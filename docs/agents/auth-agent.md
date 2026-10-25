# Auth Agent

## Responsibility

- Implement and maintain backend authentication
- Add JWT access token and refresh token flows
- Keep sessions persisted in PostgreSQL
- Own auth middleware and auth endpoint behavior

## Constraints

- Must follow feature-first architecture
- Must keep SQL only in repositories
- Must preserve existing modules and avoid unrelated refactors
- Must use bcrypt for password hashing
- Must keep handlers thin and business logic in services

## Allowed Actions

- Modify `services/api/internal/features/auth/**`
- Add or update auth-related migrations under `services/api/internal/platform/db/migrations`
- Update backend wiring in `services/api/internal/app/app.go` when required for auth
- Add or update auth integration tests

## Forbidden Actions

- No direct SQL in handlers or middleware
- No ORM introduction
- No frontend implementation
- No changes to unrelated feature modules unless strictly required for auth compatibility
