# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-27

### Added
- **Lueur**, a 100 % free dating application built on the boilerplate.
- API features: `profiles`, `photos`, `matching`, `chat` (match-scoped), `safety`, reworked `notifications`.
- Dating schema migration (`014_create_dating_schema.sql`) and tracked, locked migrations (`schema_migrations`).
- Password reset by e-mailed code, account deletion with photo file cleanup, per-request session validation.
- WebSocket realtime (`/ws`), in-memory rate limiting, signed expiring media URLs, SMTP mailer.
- Demo seed command (`cmd/seed`) with clearly flagged `.invalid` accounts and generated artwork.
- Expo app: onboarding wizard, swipe discovery, match celebration, realtime chat with read receipts,
  notifications, preferences, privacy, block/report, account deletion, light/dark themes.
- Tests: unit tests, HTTP integration tests (incl. concurrency and WebSocket), Node API smoke test,
  Playwright browser E2E of the critical journey; ESLint for the app; new CI pipeline.

### Changed
- Auth: stricter e-mail and password validation, timing-safe login, consistent `{error, code}` errors.
- Chat now requires a match; conversations are tied to matches.
- React Navigation 6 → 7; Docker image runs as non-root with a persistent `/app/data` volume.

### Removed
- Stripe billing, the `/users` directory (leaked e-mails), posts, comments, metadata-only files,
  unused Tamagui, PowerShell QA scripts, duplicate `infra/` Compose file, Maestro flow.

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
