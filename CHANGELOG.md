# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-24

### Added
- Lueur, a free dating app: onboarding, profiles, photos, discovery with swipe, likes, matches, realtime chat, notifications, preferences, privacy settings, blocks, reports and account deletion.
- API features `profiles`, `photos`, `matching`, `safety`, `realtime`; password recovery codes; signed media URLs; rate limiting; tracked migrations; demo seed command.
- ESLint, Jest, API smoke test, Playwright web E2E and a CI pipeline running integration tests against Postgres.

### Changed
- Fresh database schema (recreate databases created by the boilerplate).
- Mobile app rebuilt on React Navigation 7 with a new dark design system.

### Removed
- Stripe billing, organizations/tenancy, posts, comments, votes, generic files and the public user directory.
- Tamagui, Maestro flow and PowerShell smoke scripts superseded by Node/Playwright tests.

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
