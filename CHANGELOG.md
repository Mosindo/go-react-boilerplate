# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-21

### Added
- Lueur, a free dating application: profiles, photos, discovery, likes and passes,
  mutual matches, realtime chat, notifications, preferences, privacy settings,
  blocking, reporting, password recovery and account deletion.
- API features `profiles`, `photos`, `discovery`, `matching`, `safety`; tracked
  migrations; WebSocket realtime hub; signed media URLs; rate limiting; demo seed.
- Expo client rebuilt with a light/dark design system, onboarding and all dating screens.
- Unit, integration, API smoke and Playwright web end-to-end tests; unified CI.

### Changed
- Chat now requires an active match; `/users` no longer exposes emails; community
  modules (users, posts, comments, files) are disabled unless `ENABLE_COMMUNITY_MODULES=true`.

### Removed
- Stripe billing (the product is entirely free), PowerShell QA scripts and Maestro flow
  (replaced by cross-platform smoke and browser E2E tests), duplicate `infra/` compose file.

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
