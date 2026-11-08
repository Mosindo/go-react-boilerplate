# Ember Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Ember: a free dating application built on the boilerplate (profiles, photos, discovery, likes/matches, real-time chat, notifications, blocks/reports, password recovery, account deletion).
- Versioned migrations with `schema_migrations`, WebSocket hub, in-memory rate limiting, demo seed (`cmd/seed`).
- Mobile app rewritten (Expo/React Native, iOS/Android/web) with a light/dark design system, lint, Prettier and unit tests.
- API smoke test and Playwright browser journey (`npm run e2e:api`, `npm run e2e:web`).

### Removed
- Organizations/multi-tenancy, Stripe billing, posts, comments and generic files modules; Tamagui; PowerShell QA scripts; duplicated `infra/` compose files.

### Changed
- CI runs Go checks against PostgreSQL, mobile typecheck/lint/tests/build, the end-to-end journeys and the Docker health check.

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
