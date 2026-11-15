# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-15

### Added
- Lueur, a free dating application built on the boilerplate.
- API features: profiles, preferences, privacy, interests, photos (validated, re-encoded, signed URLs),
  mutual-preference discovery with a pluggable ranker, swipes and race-safe matches, likes received,
  match-gated realtime chat with read receipts, notifications and badges, blocks and reports,
  password reset by e-mail code, password change and full account deletion.
- Platform: tracked and locked migrations, typed errors, rate limiting, WebSocket hub, local storage,
  SMTP mailer, demo seed command (`cmd/seed`).
- Mobile app: onboarding, swipe deck, profile detail, likes, conversations, chat, notifications,
  profile editing, preferences, settings; dark and light themes.
- Tooling: ESLint, Prettier, Jest, web export, API smoke and Playwright E2E scripts, unified CI.

### Changed
- Access tokens are checked against their session on every request.
- Docker image runs as a non-root user and persists photos in a volume.

### Removed
- Posts, comments, votes, organizations and Stripe billing (the product is free).
- Metadata-only `files` module (replaced by real photo uploads).
- Tamagui, PowerShell QA scripts and the Maestro flow tied to the old screens.

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
