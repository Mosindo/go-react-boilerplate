# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Alba, a free dating application built on the boilerplate: profiles, preferences, interests, coarse geolocation, photo upload pipeline, discovery feed, likes/passes, mutual matches, match-bound chat with read receipts and WebSocket delivery, notifications, blocks, reports, password recovery and account deletion.
- Expo client rewritten around the new product (onboarding, swipe deck, chat, settings) with a light/dark design system.
- Demo seed (`go run ./cmd/seed`), cross-platform API smoke test (`scripts/smoke.mjs`) and a Playwright browser end-to-end suite.
- Tracked, transactional database migrations guarded by an advisory lock.
- Per-process rate limiting, trusted-proxy configuration and graceful shutdown.

### Changed
- Docker Compose consolidated into a single root `docker-compose.yml`; `infra/` removed.
- CI now runs formatting, vet, race-enabled tests against PostgreSQL, mobile typecheck/lint/tests, the API smoke test and the browser E2E suite.

### Removed
- Stripe billing and subscriptions, organizations/multi-tenancy, posts, comments, votes, the user directory and the generic files module.
- Tamagui and the PowerShell QA scripts.

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
