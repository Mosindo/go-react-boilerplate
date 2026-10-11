# Ember Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Breaking:** the repository is now Ember, a free dating app. Legacy SaaS tables are dropped by migration `014`; Stripe billing, organizations, posts, comments and files modules were removed.
- Migrations are tracked in `schema_migrations` and applied once, in transactions.

### Added
- Profiles, preferences, privacy, opt-in coarse location, photos with sanitisation and signed URLs.
- Discovery deck, likes/passes, atomic matching, unmatch, "likes received".
- Chat with unread counters, read receipts and WebSocket delivery; in-app notifications.
- Blocks, reports, password recovery, change password, account deletion, rate limiting.
- New Expo client (French UI, light/dark), demo seed command, smoke script, CI for API/mobile/Docker.

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
