# Alma Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Dating domain on the API: profiles, preferences, interests, coarse location, photos (validated, re-encoded, signed URLs),
  discovery with mutual criteria, likes/passes, matches, match-bound chat with read state and WebSocket delivery,
  notifications, blocks, reports, password recovery, password change and account deletion.
- Migrations `014`-`019`, tracked in `schema_migrations` behind an advisory lock.
- `cmd/seed` demo data (reserved `.invalid` addresses, refused in production).
- Expo app "Alma": onboarding, swipe deck, matches, chat, notifications, profile management, settings, light/dark theme.
- Integration test suite on a real PostgreSQL, API smoke script, Maestro UI flow, Vitest unit tests, ESLint and Prettier.
- CI: PostgreSQL service for integration tests, mobile lint/format/test/bundle, full journey on the Docker stack.

### Changed
- JWT claims no longer carry an organization; access tokens are rejected as soon as their session is revoked.
- Docker image runs as a non-root user and ships a health check; compose gains an uploads volume.
- Documentation rewritten for the product.

### Removed
- Stripe/billing configuration and routes (the product is free), generic posts/comments/files/users routes, `tamagui`.

### Pending approval
- Deleting the now-unmounted legacy code could not be done in the automated run: `services/api/internal/features/{billing,posts,comments}`,
  the matching tables created by migrations `002`, `003`, `010`, `013`, `apps/mobile/tamagui.config.ts`, `docs/agents/payments-agent.md`
  and `.codex/config.toml` (Stripe MCP). They are inert (not routed, not imported by the app).

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
