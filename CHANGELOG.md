# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased — Alma dating app

### Added
- Dating domain: profiles, preferences, interests, photos, discovery feed, likes, matches, realtime chat (WebSocket), blocks, reports, notifications.
- Password reset by e-mail code, account deletion, 18+ validation, rate limiting, tracked migrations (`schema_migrations`).
- Demo data command (`cmd/seed`), integration tests for the critical journey, mobile unit/component tests, ESLint and Prettier.

### Changed
- Mobile app rebuilt around the dating journey (onboarding, swipe deck, messages, chat, profile and privacy settings).
- Router composition moved to `internal/app`; graceful shutdown; non-root Docker image.

### Removed
- Stripe billing, organizations, posts, comments, generic files, Tamagui, legacy PowerShell smoke scripts.

## [Unreleased]

### Added
- Placeholder for upcoming unreleased changes.

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
