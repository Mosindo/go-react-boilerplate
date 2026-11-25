# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-25

### Added
- Lueur, a 100% free dating app: onboarding, profiles, interests, preferences, coarse location, photos (up to 6),
  discovery with mutual filters, like/pass, mutual matching, realtime chat with read receipts, in-app notifications,
  privacy controls, blocks, reports, password recovery and account deletion.
- WebSocket realtime channel with single-use tickets (`github.com/coder/websocket`).
- Server-side image sanitising (re-encode, EXIF/GPS removal, orientation, downscale) and signed photo URLs.
- Tracked, transactional database migrations and a clearly flagged demo seed (`cmd/seed`).
- Rate limiting, session revocation checks on every request, strict JWT claims, trusted proxies configuration.
- HTTP integration tests, Jest unit tests, API smoke test, Playwright web E2E, ESLint and Prettier, `scripts/qa.sh`.

### Changed
- Mobile app rebuilt with a dark "Lueur" design system, native-stack navigation and React Query data hooks.
- Docker image runs as non-root with a persistent uploads volume; Compose and CI updated.

### Removed
- Stripe billing, organizations/subscriptions, posts, comments, votes, the `/users` listing (it exposed every email),
  client-created notifications, Tamagui, the Maestro flow and the PowerShell smoke scripts targeting removed endpoints.

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
