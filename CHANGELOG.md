# Aimant Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-13

### Added
- Free dating product "Aimant": profiles, preferences, interests, location (rounded), photos with server-side sanitizing and signed URLs.
- Discovery with mutual filters and a pluggable ranker; swipes; race-free mutual-like matches; unmatch.
- Match-gated realtime chat (WebSocket) with read receipts, pagination and local clearing.
- In-app notifications pushed in realtime; blocks, reports and automatic moderation hiding.
- Password recovery by e-mailed code, account deletion, session-checked access tokens, rate limiting, tracked migrations.
- New Expo app (French UI, light/dark themes), onboarding wizard, swipe deck, chat, settings.
- Demo seed command, Go unit/integration tests, API smoke test, Playwright E2E, CI pipeline.

### Security
- Refresh tokens are single-use (atomic rotation) and replaying a rotated token revokes the session; logout closes realtime sockets.
- Password reset attempts are counted atomically; at most 3 codes per account per hour.
- Paused or moderation-hidden members cannot browse, swipe or chat; one report alone can no longer hide a profile.
- JSON photo routes keep the global body limit; image decoding capped at 16 MP with bounded concurrency; replacing a photo invalidates old signed links.

### Removed
- Stripe billing, posts, comments, votes, metadata-only files and the `/users` directory (exposed e-mails), plus the unused Tamagui setup and PowerShell QA scripts.

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
