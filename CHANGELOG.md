# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-10

The boilerplate becomes **Lueur**, a complete and 100% free dating application.

### Added
- Dating data model with tracked migrations: profiles, preferences, interests, photos, swipes,
  matches (conversations), messages, blocks, reports, notifications, password resets.
- API modules `profiles`, `photos`, `discovery`, `matches`, `safety`, `realtime`; password reset by
  emailed code, password change and account deletion in `auth`.
- Secure photo pipeline (content sniffing, decompression-bomb guard, re-encoding that strips EXIF/GPS)
  and signed, expiring media URLs.
- Discovery with mutual-preference eligibility, coarse geolocation, pluggable ranking and race-free
  matching; realtime chat over WebSocket with read receipts.
- Rate limiting, request body limits, graceful shutdown, non-root Docker image with healthcheck.
- Demo seed command clearly separated from real data.
- Expo app: Lueur design system (light/dark), onboarding, discovery deck, match celebration, chat,
  notifications, preferences, privacy, blocked users, account management.
- HTTP integration tests on Postgres, Playwright E2E of the critical journey, ESLint and Prettier,
  `scripts/smoke.sh`, rewritten CI.

### Removed
- Stripe billing, organizations/multi-tenancy, posts, comments, votes, public user listing,
  metadata-only files module, open user-to-user chat.
- Unused Tamagui dependency, PowerShell QA scripts, Maestro flow, duplicated `infra/` compose file.

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
