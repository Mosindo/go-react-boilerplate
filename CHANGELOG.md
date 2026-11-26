# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-26

### Added
- Lueur, a free dating app: onboarding, profiles with up to 6 photos, preferences, interests, coarse location.
- Discovery deck with mutual filters and pluggable ranking; likes, passes, race-free matches, likes received.
- Match-gated realtime chat (WebSocket) with read receipts, unread counters and cursor pagination.
- Notifications (match, like, message) stored and pushed in real time.
- Blocking, reporting with review escalation, rate limiting, password reset by e-mailed code, password change,
  account deletion with full data and photo purge.
- Secure photo pipeline: magic-byte validation, decompression guard, EXIF-stripping re-encode, signed media URLs.
- Tracked SQL migrations, demo seed, HTTP integration tests, Jest tests, Playwright web E2E, ESLint and Prettier.
- New light/dark design system for the Expo app (iOS, Android, web).

### Changed
- Auth sessions are verified on every request (immediate logout/reset), tokens no longer carry organizations.
- Docker image runs as a non-root user and persists photos in a volume; CI runs backend, mobile and E2E jobs.

### Removed
- Stripe billing, organizations, posts, comments, votes, global user listing and the metadata-only files module.
- Tamagui, Maestro flow and PowerShell smoke scripts tied to the removed modules.

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
