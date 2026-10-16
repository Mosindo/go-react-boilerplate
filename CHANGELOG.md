# Lien Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-10-16

Lien: the boilerplate becomes a complete, 100% free dating application.

### Added
- Profiles with immutable, server-verified age (18+), preferences, interests and privacy settings.
- Photo management (validation by content, re-encoding that strips metadata, ordering, main photo, replacement) served through signed URLs.
- Discovery with mutual preference filtering and an isolated ranking, swipes, race-safe matching and "who liked me" for everyone.
- Match-bound chat with real-time delivery over WebSocket, read receipts, unread counters, per-user conversation clearing.
- Notifications, blocking, reporting, password recovery by email code, full account deletion.
- Rate limiting, tracked SQL migrations (`schema_migrations`), `cmd/seed` demo data and `cmd/moderate` operator CLI.
- Expo app rebuilt around these flows (French UI), Vitest unit tests, ESLint/Prettier, Node smoke journey, Maestro flow and Playwright browser journey.

### Changed
- Auth: access tokens no longer carry tenant data; email is trimmed/lower-cased before validation; passwords limited to 72 bytes (bcrypt).
- CI runs Go tests against PostgreSQL with the race detector, mobile lint/format/tests, a bundle build and a browser journey.

### Removed
- Stripe billing, subscriptions, organizations/multi-tenancy, posts, comments, files and direct-message modules (unwired; see the repository notes for files pending deletion).
- Tamagui.

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
