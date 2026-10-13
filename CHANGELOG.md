# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Lueur, a free dating application built on the boilerplate: profiles, photos (validated and re-encoded), preferences, geographic discovery with approximate distance, atomic likes and matches, realtime chat with read receipts, notifications, blocking, reporting, account recovery and account deletion.
- Tracked, embedded SQL migrations (`schema_migrations`) with legacy-database baselining; migrations `014` (dating schema) and `015` (SaaS cleanup).
- Rate limiting, strict JWT claims (issuer, mandatory expiry, HS256 only), refresh-token rotation with replay protection, body size limits and trusted-proxy configuration.
- `cmd/seed` demo data (`@seed.invalid` accounts, refused in production).
- French-language Expo client (iOS, Android, web) with light/dark themes, swipe deck, onboarding and settings.
- ESLint, Prettier, Jest/Testing Library UI tests, Playwright browser E2E and a restructured CI pipeline (Go with PostgreSQL, mobile, E2E, Docker).

### Changed
- Product is no longer a generic SaaS boilerplate: README and architecture documentation describe Lueur.
- Docker image runs as a non-root user and ships the `seed` binary; PostgreSQL 16 is bound to localhost in Compose.

### Removed
- Stripe billing, organizations, posts, comments, votes, files, the public user directory, and the unused Tamagui dependency.

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
