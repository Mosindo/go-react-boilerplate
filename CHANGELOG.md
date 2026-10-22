# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Aurore dating application: accounts with password recovery, profiles (18+), preferences, up to six validated and re-encoded photos, swipe discovery, atomic reciprocal matches, real-time chat (WebSocket) with read receipts, notifications, blocking, reporting, account deletion.
- Coarse location handling (coordinates rounded to ~1 km, approximate distances only).
- Rate limiting, per-request session validation, versioned migrations (`schema_migrations`).
- Demo seed (`cmd/seed`, `@demo.invalid` accounts, refuses production), API smoke test, Playwright browser end-to-end test, Jest, ESLint, Prettier.
- Web target for the Expo app.

### Changed
- Product pivot from the generic `go-react-saas` boilerplate to a free dating app (see README).
- Database schema replaced by `014_dating_schema.sql`.
- Mobile UI rebuilt on the existing design-system components with a new "Aurore" palette; Docker image runs as non-root with an uploads volume.

### Removed
- Stripe billing, subscriptions and organizations are no longer wired (the product is 100 % free). Their source files are still present pending validation of the deletion: `services/api/internal/features/{billing,posts,comments,files,users}`, `apps/mobile/tamagui.config.ts`, `apps/mobile/src/theme`, `.codex/config.toml`, `docs/agents/payments-agent.md`.

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
