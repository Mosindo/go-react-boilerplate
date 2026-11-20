# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-20

### Added
- Lueur, a free dating application built on the boilerplate.
- API features: profiles and mutual preferences, coarse location, interests catalog, photos (validation, re-encoding, ordering, signed URLs), discovery with a pluggable ranker, swipes and atomic matching, free "who liked me", chat with read receipts, pagination and local deletion, grouped notifications, blocks, reports with review threshold, password recovery, password change, account deletion.
- WebSocket realtime channel with short-lived tickets.
- Tracked migrations (`schema_migrations`), demo seed command, rate limiting, server-side session checks.
- Mobile app: onboarding, swipe deck, profile pages, likes, matches and chat, notifications, settings, safety flows, light/dark design system.
- Tooling: integration tests on PostgreSQL, unit tests, ESLint, Prettier, API and Playwright web end-to-end tests, CI pipeline.

### Changed
- JWT validation now checks issuer and token type and verifies the session server-side on every request.
- Docker image runs as a non-root user with a persistent uploads volume.

### Removed
- Stripe billing, organizations/multi-tenancy, posts, comments, votes and the generic files module.
- Tamagui, Maestro and PowerShell QA scripts, duplicate `infra/` compose file.

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
