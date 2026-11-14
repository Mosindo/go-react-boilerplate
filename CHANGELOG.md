# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-14

### Added
- Lueur, a free dating application built on the boilerplate.
- API features: profiles (18+ validation, preferences, coarse location, privacy, interests), photos
  (content-validated uploads, EXIF orientation, resize + metadata-stripping re-encode, signed URLs,
  ordering/replace/delete), matching (mutual-preference discovery ranking, like/pass, race-safe
  matches, unmatch), chat (match-scoped conversations, read receipts, local clear, keyset
  pagination), notifications, safety (block, report, auto-suspension), password reset by e-mailed
  code, account deletion, WebSocket realtime hub, rate limiting, versioned migrations, demo seed.
- Expo app: welcome/auth, onboarding wizard, swipe deck, full profiles, match celebration, chat,
  activity, profile/photo/preferences/privacy settings, blocked members, report and delete flows,
  light/dark design system.
- HTTP + WebSocket integration test suite, Playwright web end-to-end critical flow, CI pipeline.

### Removed
- Stripe billing, organizations/multi-tenancy, posts, comments, votes, the `/users` directory and the
  metadata-only files module (incompatible with a free dating product or with member privacy).
- Tamagui (unused) and PowerShell/Maestro scripts targeting the removed endpoints.

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
