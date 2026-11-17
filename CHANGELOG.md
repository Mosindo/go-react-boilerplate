# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-17

### Added
- Lueur, a free dating app: profiles with photos and interests, mutual-preference discovery with
  a swipe deck, race-safe matching, realtime chat with read receipts, notifications, privacy
  settings, blocking, reporting, password reset and permanent account deletion.
- Tracked SQL migrations (`schema_migrations`) with an advisory lock and a legacy-schema guard.
- Photo pipeline (magic-byte validation, decompression-bomb guard, resize, metadata stripping,
  signed expiring URLs) and pluggable storage.
- WebSocket realtime hub, rate limiting, trusted proxies, strict JWT validation.
- Demo seed command with flagged, non-routable demo accounts.
- Go HTTP integration tests, node unit tests, Playwright end-to-end scenario, CI for all of them.

### Changed
- Rebuilt the Expo app around the dating journey, with a new visual identity and React Navigation 7.
- Docker image runs as non-root, ships the seed binary and a healthcheck.

### Removed
- Stripe billing, posts, comments, votes, organisations, the public user list, Tamagui, PowerShell
  QA scripts and Maestro flow from the former boilerplate.

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
