# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Dating app domain: profiles, preferences, interests, photos (authenticated, re-encoded), discovery feed with mutual preferences,
  swipes with atomic match creation, matches, real-time chat (WebSocket), read receipts, notifications, blocks, reports, account deletion.
- Password reset by e-mailed one-time code, change password, session-checked access tokens, rate limiting, request size limits.
- `cmd/seed` demo accounts (`is_demo`), `docs/API.md` contract, Makefile, migration tracking (`schema_migrations`).
- New Expo client "Lueur" (French UI, light/dark theme, onboarding, swipe deck, chat, settings), ESLint, Prettier, Jest.

### Changed
- Product repositioned from the generic `go-react-saas` boilerplate to a 100 % free dating app.
- Single canonical Compose file (`infra/docker-compose.yml`), non-root API image with healthcheck, merged CI workflow.

### Removed
- Stripe billing, subscriptions, organizations/tenants, posts, comments, generic files module, Tamagui, PowerShell QA scripts.
- Breaking: the database schema was rewritten (`001_init.sql`); databases created by the old boilerplate must be recreated.

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
