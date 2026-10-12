# Changelog

## Unreleased — dating application

### Added
- Dating product: profiles, preferences, photos, discovery, swipes and matches, match-scoped realtime chat, notifications, blocking and reporting, password recovery, account deletion.
- Demo data command (`cmd/seed`) using the reserved `demo.invalid` domain.
- Tracked database migrations with an advisory lock; migrations 014-017.
- Mobile design system with light/dark themes, Playwright browser E2E, API smoke test, Jest tests, ESLint and Prettier.
- `node scripts/qa.js` cross-platform quality gate; new CI workflow with PostgreSQL service and E2E job.

### Changed
- Sessions are validated against the database on every request (logout, reset, suspension and deletion take effect immediately).
- Passwords are limited to 72 bytes (bcrypt limit); login timing no longer reveals unknown accounts.
- Gin only trusts proxies listed in `TRUSTED_PROXIES`.
- Docker image runs as a non-root user and stores photos on a volume.

### Removed
- Stripe billing and the organization (tenant) concept from the routes, configuration and mobile app.
- Tamagui and the generic boilerplate screens / UI kit; Maestro and PowerShell QA scripts (replaced by Playwright and Node scripts).

### Pending
- Legacy Go modules (`billing`, `chat`, `comments`, `files`, `notifications`, `posts`, `users`) and migrations 002-006 / 008-013 are unreferenced and await a validated deletion.
