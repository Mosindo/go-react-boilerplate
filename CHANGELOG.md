# Amora Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Amora dating app built on the boilerplate: profiles, preferences, interests, photos, discovery, race-safe matching, match-bound chat with WebSocket delivery and read receipts, in-app notifications, blocking and reporting, password recovery, account deletion.
- Tracked, transactional SQL migrations (`schema_migrations`) with an advisory lock.
- In-memory rate limiting, request body limits, trusted-proxy configuration.
- `cmd/seed` demo data (`@demo.invalid`, development only).
- Mobile: design system re-skin, onboarding wizard, swipe deck, chat, activity, settings; unit tests (Vitest), ESLint, HTTP smoke test, Playwright browser journey, Maestro flow.
- Web build target (react-native-web) used for browser E2E.

### Changed
- Auth: email trimmed and validated server-side, 8–72 byte passwords, constant-time login for unknown emails, access-token parsing shared with the WebSocket handshake.
- Notifications can only be created by the server; the public create endpoint is gone.
- Docker image runs as a non-root user and ships the seeder; compose adds an uploads volume.

### Removed
- Stripe/billing configuration and routes, the generic user directory, posts, comments, files and the old username-keyed chat are no longer mounted. Their source is still in the tree pending owner validation to delete: `services/api/internal/features/{billing,posts,comments,users,files,chat}`, `apps/mobile/tamagui.config.ts`, `apps/mobile/src/theme/`, `docs/agents/payments-agent.md`, and migrations `001`–`013` tables that only those modules used.

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
