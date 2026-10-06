# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Lien**: the boilerplate becomes a free dating application. New backend features `profiles`, `photos`, `matching`, `safety`, `account`; chat rebuilt around matches; notifications created server-side; realtime WebSocket hub.
- Migrations `014`–`016` (profiles, preferences, interests, photos, swipes, matches, blocks, reports, password resets) and tracked migrations (`schema_migrations`, advisory lock).
- Password recovery by one-time emailed code; session-validating auth middleware; rate limiting and request body limits.
- `cmd/seed` (demo data, refuses production) and `cmd/moderation` (report review, suspension).
- Mobile app rewritten: onboarding, swipe discovery, likes received, matches and realtime chat, notifications, profile/photos/preferences/privacy settings, block/report, account deletion, dark mode.
- Integration tests against real PostgreSQL, mobile unit tests, API smoke script, Maestro flow, `scripts/check.sh`.

### Changed
- Docker image runs as non-root with an uploads volume and healthcheck; compose files now use PostgreSQL 16 and no payment variables.
- CI runs lint/format/unit tests for the mobile app and integration tests with a PostgreSQL service.
- `/me` and auth responses unchanged; `GET /users` and the `/chats` API are replaced by `/matches` (a user directory would expose every account).

### Removed
- Billing/Stripe, posts, comments, generic files and user-listing routes are no longer wired into the router (the application is free and has no such concepts). Their source packages and the legacy migrations `001`–`013` remain in the repository pending an explicit deletion decision.

## [0.1.0] - 2026-03-17

### Added
- Introduced a standardized documentation set with `README.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, and `docs/ARCHITECTURE.md`.
- Added a generic changelog for future platform releases.

### Changed
- Converted the repository from a single-product application into `go-react-saas`, a reusable fullstack boilerplate.
- Repositioned the repository as `go-react-saas`, a reusable fullstack boilerplate for social, forum, SaaS, marketplace, and community products.
- Updated project descriptions, examples, and documentation to use generic module language.
- Aligned the documentation with the feature-first backend and generic frontend shell.

### Removed
- Removed legacy status and daily log documents tied to earlier single-product tracking.
- Removed remaining product-specific naming from repository-level documentation.
