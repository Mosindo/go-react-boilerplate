# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Lumen: a complete free dating app built on the boilerplate - profiles, photos, preferences,
  discovery, likes/passes, matches, real-time chat (WebSocket), notifications, blocks, reports,
  password recovery and account deletion.
- Tracked, transactional database migrations (`schema_migrations`) and a dating schema.
- `cmd/seed` demo data (`@demo.invalid` accounts, refuses to run in production).
- API contract in `docs/API.md`, Go integration tests (journey, permissions, security, WebSocket)
  and `scripts/smoke-api.js`.

### Changed
- Product pivot from a generic SaaS boilerplate to a dating app; the sessions/JWT foundation was kept.
- Docker Compose consolidated at the repository root with an uploads volume and a non-root API image.
- CI now runs Postgres-backed Go tests, mobile lint/format/typecheck/tests and a Docker smoke journey.

### Removed
- Stripe billing, posts, comments, organizations and the old generic chat/files/users modules were moved to `archive/legacy-saas/` (not built). The app has no payments of any kind.

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
