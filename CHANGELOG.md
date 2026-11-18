# Lueur changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-18

The boilerplate becomes **Lueur**, a free dating app.

### Added
- API: profiles, preferences, interests, privacy controls and approximate location.
- API: photo uploads (content validation, re-encoding without EXIF, signed URLs, ordering, replacement).
- API: discovery with mutual filters and ranking, swipes, atomic match creation, unmatch.
- API: match-only chat with cursor pagination, read receipts and WebSocket realtime events.
- API: blocks, reports with automatic moderation threshold, account deletion.
- API: password reset by e-mailed code, password change, session check on every request, rate limiting.
- API: tracked migrations with advisory locking, demo seed command (`cmd/seed`).
- Mobile: complete Expo app (iOS, Android, web) with onboarding, swipe deck, matches, chat, activity, settings, light/dark themes.
- Quality: integration test suite on Postgres, API smoke test, Playwright web E2E of the full journey, ESLint and Prettier, `scripts/qa.sh`.

### Changed
- CI runs Go tests against Postgres with the race detector, mobile checks, both E2E suites and a Docker health check.
- Docker image runs as a non-root user with a media volume.

### Removed
- Stripe billing and subscriptions (the app is free), posts/comments/votes, the metadata-only files module and the `/users` directory endpoints.
- Tamagui, the Maestro flow and the PowerShell smoke scripts targeting removed modules.

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
