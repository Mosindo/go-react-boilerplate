# Liane Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Dating domain: profiles, preferences, interests, photos (signed URLs, sanitised uploads), discovery, swipes, matches.
- Real-time chat over WebSocket with unread counters and read receipts; notifications.
- Blocking, reporting, password recovery by emailed code, account deletion, rate limiting.
- Tracked SQL migrations, demo seed (`cmd/seed`), Go integration tests, `scripts/smoke-api.sh`.
- New Expo client "Liane" (see apps/mobile).

### Removed
- Generic SaaS modules (posts, comments, files, users directory, Stripe billing, organizations) and the unused Tamagui setup.

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
