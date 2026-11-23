# Hearth Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-23

### Added
- Hearth, a free dating app on the go-react-saas foundation.
- API features: profiles, preferences, privacy settings, approximate location, interests,
  photos (validated, re-encoded, signed URLs), mutual discovery, swipes, race-free matching,
  match-scoped realtime chat with read receipts, blocks, reports with auto-suspension,
  notifications, password recovery by email, password change, account deletion.
- Platform: tracked migrations, rate limiting, local object storage, SMTP mailer, WebSocket hub,
  session revocation checks, uniform error payloads.
- Mobile app: original dark design system, onboarding, discovery deck, matches, chat, activity,
  profile and settings screens.
- Quality: integration suite against PostgreSQL, ESLint/Prettier, API smoke, Playwright web E2E,
  consolidated CI, `scripts/qa.js`, demo seed command.

### Removed
- Stripe billing, posts, comments, votes, metadata-only files, `/users` directory, Tamagui,
  PowerShell QA scripts and Maestro flow tied to the removed modules.

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
