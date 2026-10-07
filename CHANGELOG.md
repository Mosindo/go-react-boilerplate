# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

## Unreleased — Ember free dating app

- Turned the boilerplate into a free dating app: profiles, photos, preferences, discovery, matching, real-time chat, notifications, block/report, account deletion, password reset.
- New migrations 014–020 (retire generic schema, add dating schema); migration runner now records applied versions.
- Removed Stripe billing and organisations from the active API (tenant claims dropped from JWT).
- Mobile app rebuilt (Tamagui removed; React Navigation 7, TanStack Query, ESLint/Prettier/Jest).
- Legacy Go packages `billing`, `posts`, `comments`, `files`, `users` are no longer wired and are pending deletion approval.
