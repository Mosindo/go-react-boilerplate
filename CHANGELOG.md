# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Lumo, a free dating app: profiles, preferences, interests, photos (re-encoded, ordered, authorised serving), mutual-preference discovery, swipes, matches, "who liked me" (free), real-time chat (WebSocket) with read receipts, in-app notifications, blocking, reporting, password recovery and account deletion.
- Rate limiting, tracked and lock-protected migrations (`schema_migrations`), `cmd/seed` demo data, `scripts/smoke-api.mjs`, Go integration tests on PostgreSQL, Jest and Playwright (web) tests for the Expo client.

### Changed
- The Expo client was rewritten around React Navigation 7, TanStack Query and a French design system with light/dark themes.
- CI now runs formatting, vet, race tests on PostgreSQL, the mobile quality gates, the API smoke test and the web E2E test.

### Removed
- Multi-tenant organizations, Stripe billing/subscriptions, posts, comments, votes, generic files and the generic user directory.
- Windows-only PowerShell QA scripts and the Maestro flow for the old app.

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
