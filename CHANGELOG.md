# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-12

### Added
- Lueur, a 100% free dating app built on the boilerplate.
- API: profiles, preferences, curated interests, photo pipeline (sniffing, EXIF orientation, resize, metadata stripping, signed URLs), discovery with mutual filters and ranking, swipes and race-safe matches, match-gated chat with pagination and read receipts, WebSocket realtime, in-app notifications, blocks, reports, password reset by email code, password change, account deletion, rate limiting, tracked migrations, demo seed.
- Mobile: new design system (light/dark), auth and password recovery, 5-step onboarding, swipe deck with accessible buttons, match modal, conversations, notifications, profile/preferences/privacy/blocked/account screens.
- Quality: Go unit + integration tests (incl. concurrency and WebSocket), ESLint/Prettier, frontend unit tests, API smoke and Playwright browser E2E, Maestro native flow, CI with Postgres and E2E jobs, `scripts/qa.sh`.

### Changed
- `/users` no longer exposes emails and respects privacy and blocks.
- Chat is only available between matched members.
- Access tokens are revoked immediately on logout, password reset and account deletion.

### Removed
- Stripe billing (the product is free), unused Tamagui dependency, duplicate `infra/` compose file, obsolete PowerShell smoke scripts.

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
