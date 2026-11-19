# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-19

### Added
- Lueur, a complete and entirely free dating application (API + Expo app for iOS, Android and web).
- Accounts with session revocation, password recovery by emailed code and full account deletion.
- Profiles, preferences, interests, approximate location (~1 km) and privacy controls.
- Photo management (1-6 photos, re-encoding without metadata, signed expiring URLs).
- Discovery with mutual filters and pluggable ranking, likes/passes, atomic matching, free "who liked me".
- Realtime chat over WebSocket with read receipts, local deletion and unmatch.
- Blocking, reporting, notifications.
- Tracked transactional migrations, demo seed, integration tests, Jest tests, Playwright web E2E, CI.

### Removed
- Stripe billing, organizations/subscriptions, posts, comments, votes and generic files modules.
- Unused Tamagui dependency and PowerShell/Maestro QA scripts (replaced by cross-platform CI and E2E).

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
