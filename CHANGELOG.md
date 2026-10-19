# Ember Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Ember, a 100% free dating app built on the boilerplate: profiles with 18+ validation, private sanitised photos, preferences, interests, discovery, mutual-like matching, realtime chat with read receipts, notifications, blocking, reporting, password recovery and account deletion.
- Mobile app rewritten around the dating flows (onboarding, swipe discovery, likes received, messages, settings) with light/dark themes, accessibility labels and loading/empty/error states.
- WebSocket hub, token-bucket rate limiting, migration tracking with advisory lock, demo seed (`cmd/seed`), HTTP integration test suite, mobile unit/UI tests, API smoke test and Maestro flow.

### Changed
- JWT no longer carries an organization claim; passwords are limited to 72 bytes (bcrypt).
- Single canonical `docker-compose.yml` (PostgreSQL 16, non-root API image with healthcheck and uploads volume).
- CI now runs gofmt, vet, race tests against PostgreSQL, mobile typecheck/lint/format/tests/bundle and a full-stack smoke test.

### Removed
- Generic social modules (posts, comments, files, users listing), multi-tenancy and the Stripe billing module; legacy PowerShell QA scripts and the Codex Stripe MCP config.

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
