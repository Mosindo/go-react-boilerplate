# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Product pivot**: the repository is now Ember, a free dating app (API + Expo client). The generic SaaS domain was replaced.
- Single clean `001_init.sql` migration and a tracked migration runner (`schema_migrations`, advisory lock).
- JWT middleware now verifies the session server-side; claims no longer include an organization.
- Docker Compose consolidated at the repository root (the duplicate `infra/` stack was removed).
- CI now runs gofmt, vet, unit + integration tests against PostgreSQL, mobile typecheck/lint/format/tests, the web bundle and a Docker boot check.

### Added
- Profiles, preferences, interests, private photo management, discovery feed, swipes and matches, chat with read receipts,
  notifications, blocks and reports, WebSocket realtime, password recovery, account deletion, rate limiting, seed command.
- Expo client for iOS/Android/web with onboarding, swipe deck, chat, settings and a light/dark theme.
- Browser end-to-end test of the critical journey (`apps/mobile/e2e/web-smoke.js`).

### Removed
- Stripe billing/subscriptions, organizations, posts, comments, generic files, Tamagui, PowerShell smoke scripts.

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
