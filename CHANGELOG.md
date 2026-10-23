# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Free dating application on top of the boilerplate: profiles, preferences, private photos (validated, metadata-stripped, resized),
  discovery with mutual-preference filtering and ranking, atomic likes/matches, realtime chat over WebSocket, notifications,
  blocking, reporting with automatic suspension, password recovery, account deletion.
- Migration runner with `schema_migrations` tracking and advisory locking; per-IP/per-user rate limiting; strict JWT validation
  (HS256, `exp`, `iss`) with live session checks; `cmd/seed` demo data; `docs/API.md` contract; mobile app "Cléo" (Expo, French, light/dark).

### Changed
- Schema reset: migrations now live in `internal/platform/db/schema/`. Existing databases of the previous boilerplate must be recreated (`docker compose down -v`).
- Integration tests run against PostgreSQL in CI; mobile CI now runs typecheck, lint, format check and unit tests.

### Removed
- Stripe billing/subscriptions, organizations (multi-tenancy), posts, comments and generic files are no longer wired into the API.
  Their source directories still exist and are listed for deletion in the README ("Legacy code pending removal").

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
