# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Alba, a free dating app: profiles, photos, discovery, likes, matches, realtime chat, notifications, blocks, reports and account deletion.
- Tracked SQL migrations (`schema_migrations`), rate limiting, signed private photo URLs, image re-encoding, password reset by email.
- Demo seed (`cmd/seed`), API smoke test and Playwright browser journey, CI with PostgreSQL, lint and formatting checks.

### Changed
- The Expo app was rebuilt (French UI, light/dark theme) on top of a single typed API client.
- Database schema rebuilt from a clean baseline; recreate existing databases.

### Removed
- Stripe billing, organizations/tenants, posts, comments, votes, generic files module, Tamagui, PowerShell QA scripts and Maestro flows.

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
