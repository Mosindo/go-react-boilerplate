# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-22 — Lueur, free dating app

### Added
- Dating domain on the API: profiles, preferences, interests catalog, approximate location, photos (upload/replace/delete/reorder, signed URLs, EXIF stripping), mutual-preference discovery with a pluggable ranker, swipes, race-free matches, likes received, unmatch.
- Match-gated real-time chat (SSE), read receipts, unread counters, cursor pagination, local conversation clearing.
- Safety: blocks, reports with automatic review threshold, rate limiting, body limits, trusted proxies, immediate session revocation.
- Account recovery by e-mail code, password change, account deletion with full data erasure.
- Migration tracking (`schema_migrations` + advisory lock) and demo seed command (`cmd/seed`).
- New Expo app "Lueur": onboarding, discovery deck, likes, matches & chat, notifications, profile editing, preferences & privacy, settings; dark design system.
- Tests: Go integration suite for the dating flows, frontend unit tests, Playwright UI end-to-end, API smoke script; ESLint + Prettier; CI jobs for backend, mobile and e2e.

### Changed
- `/users` and chat payloads no longer expose e-mail addresses; `/users` lists matched contacts only.
- Chat endpoints require a match; message history is paginated with a `before` cursor.

### Removed
- Stripe billing module, Tamagui, Maestro flows and the legacy generic mobile screens.

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
