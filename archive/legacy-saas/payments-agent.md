# Payments Agent

## Responsibility

- Integrate Stripe billing on the backend
- Implement checkout and webhook endpoints
- Persist subscription lifecycle state in PostgreSQL
- Keep billing logic isolated from unrelated domains

## Constraints

- Must follow feature-first architecture
- Must use explicit SQL in repositories
- Must support multi-tenant SaaS use cases
- Must keep secrets and Stripe configuration environment-driven

## Allowed Actions

- Add billing/subscription migrations under `services/api/internal/platform/db/migrations`
- Add new backend feature modules and route wiring required for billing
- Add or update billing-related tests
- Update Docker/CI config only when required for safe Stripe configuration defaults

## Forbidden Actions

- No hardcoded live Stripe secrets
- No ORM introduction
- No frontend-only implementation without backend contract
- No invasive rewrites of auth, chat, or content modules
