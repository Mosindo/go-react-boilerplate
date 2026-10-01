# Lumen Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Dating domain: profiles, preferences, interests, photos (secure upload pipeline), discovery, swipes, matches.
- Match-gated chat with read receipts, per-user conversation clearing and WebSocket push (short-lived tickets).
- Safety: blocking, reporting, admin moderation (`promote-admin`), account deletion with file cleanup.
- Password recovery (e-mail code), password change, session-checked auth middleware, token-bucket rate limiting.
- `cmd/seed` demo data (development only, `@demo.invalid` accounts).
- Mobile app rewrite: onboarding, swipe deck, matches, conversations, notifications, settings, dark mode, Jest tests, ESLint/Prettier, Maestro flow.
- Node end-to-end smoke test of the whole dating journey (`npm run e2e:smoke`).

### Changed
- Product is now **Lumen**, a free dating app (no subscription, paywall, boost or like quota).
- Chat is restricted to active matches; `/users` (member listing) removed; notifications are server-generated only.
- Migrations are serialized with an advisory lock; `014`/`015` add the dating schema.
- Docker image runs as non-root with an uploads volume; Compose uses PostgreSQL 16.

### Removed
- `posts`, `comments`, `billing` (Stripe), `files` and organization-based routing from the HTTP API
  (source folders remain on disk pending explicit validation of their deletion).

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
