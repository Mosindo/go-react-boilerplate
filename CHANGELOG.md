# Alma Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

## [1.0.0] - 2026-10-21

### Added
- Dating product end to end: accounts with a server-enforced 18+ age gate, profiles, preferences, interests, photos (6 max, re-encoded, signed URLs), discovery with mutual eligibility, likes / passes, race-free matching, real-time chat with read receipts, notifications, blocking, reporting, privacy controls and full account deletion.
- Expo client in French with a light / dark design system, swipe deck with button fallbacks, onboarding wizard and web support.
- Embedded SQL migrations tracked in `schema_migrations`.
- Session revocation checked on every request; rate limiting; account recovery by e-mailed code.
- `cmd/seed` demo data (development only, marked `is_demo`), Playwright critical-flow test, CI with PostgreSQL.

### Changed
- Repository repositioned from the generic `go-react-saas` boilerplate to the Alma dating app (see README for the decisions).

### Removed
- Stripe billing, organizations / multi-tenancy, posts, comments, votes, generic files module, Tamagui, PowerShell smoke scripts and agent docs tied to the SaaS boilerplate.
