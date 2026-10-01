# Aurore Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-10-01

### Added
- Aurore, a 100% free dating app built on the former `go-react-saas` boilerplate.
- API features: `profiles` (18+ validation, preferences, interests, approximate location), `photos`
  (content-sniffed, re-encoded, access-controlled), `discovery` (eligibility query, swipes, race-safe
  matches, recommendation ordering), `chat` (match-scoped conversations, read state, local deletion,
  WebSocket delivery), `notifications`, `moderation` (blocks, reports), account recovery and deletion.
- Versioned migrations (`schema_migrations`), rate limiting, demo seed (`cmd/seed`).
- `apps/aurore`: Expo app (iOS/Android/web) with onboarding, swipe discovery, chat, notifications,
  settings, light/dark themes; unit tests and a real-browser end-to-end test.

### Changed
- CI now runs API tests against PostgreSQL, app typecheck/lint/tests/web build, Docker health and E2E.
- Docker image runs as non-root, ships the seed binary and persists uploads in a volume.

### Removed
- Stripe/billing, posts, comments, organizations (multi-tenant) and the generic users/files modules.
- Tamagui (unused) from the app dependencies.

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
