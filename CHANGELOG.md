# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-11-16

### Added
- Free dating product "Lueur": accounts (18+), password reset by emailed code, password change,
  account deletion with full data and photo purge.
- Profiles, preferences (mutual gender/age/distance), interests, coarse location (~1 km) and
  approximate distances.
- Photo pipeline: content sniffing, decompression-bomb guard, JPEG re-encoding (EXIF/GPS stripped),
  ordering, main photo, replace, delete, signed expiring media URLs.
- Discovery with SQL hard filters and a pluggable ranker; swipes and race-free matching.
- Realtime chat over WebSocket (single-use tickets), read receipts, idempotent sends, history paging.
- Blocks, reports, unmatch, activity notifications, per-IP/per-user rate limiting.
- Expo app (iOS/Android/web) with an original dark design system, onboarding, swipe deck,
  matches, conversation, activity, profile and privacy settings.
- Demo seed (`cmd/seed`), Postgres integration tests, API smoke and Playwright web E2E, new CI.

### Changed
- Migrations are now tracked in `schema_migrations`, transactional and advisory-locked; the schema
  was reset (legacy boilerplate databases must be recreated).
- Single `docker-compose.yml` at the repository root; non-root API image with an uploads volume.

### Removed
- Stripe billing, posts, comments, votes, organizations/multi-tenancy, public user directory,
  metadata-only files module, unused Tamagui, PowerShell QA scripts.

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
