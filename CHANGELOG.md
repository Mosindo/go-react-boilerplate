# Seren Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-11

The boilerplate becomes Seren, a free dating application.

### Added
- Dating domain API: profiles, preferences, interests, approximate location, photos (secure processing and signed URLs), discovery with reciprocal filters and a pluggable ranker, likes/passes, race-safe matches, match-gated realtime chat with read receipts, blocks, reports, in-app notifications.
- Password reset by e-mailed code, account deletion with full data cleanup, session checks on every request, rate limiting.
- Tracked, lock-protected database migrations and a demo data seed (`cmd/seed`).
- New Expo app with original design system (light/dark), onboarding, swipe deck, chat, notifications, privacy and account screens.
- Tests: Go unit and HTTP integration tests, frontend unit tests, API smoke test, Playwright web journey; CI runs all of them plus the Docker health check.

### Changed
- React Navigation upgraded to v7; ESLint and Prettier added to the mobile app.
- Docker image runs as non-root with a persistent uploads volume; single `docker-compose.yml`.

### Removed
- Stripe billing, organizations/tenancy, posts, comments, votes, metadata-only files, open user directory and unrestricted direct messages.
- Tamagui, Maestro and PowerShell QA scripts (replaced by the new E2E suite).

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
