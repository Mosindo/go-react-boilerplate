# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Ember, a 100% free dating app: profiles with up to 6 photos, preferences and interests, discovery swipe deck, atomic mutual-like matching, realtime chat (WebSocket) with read receipts, notifications, blocking, reporting, password recovery and full account deletion.
- Authenticated photo storage with strict validation and re-encoding.
- Migration tracking (`schema_migrations`), rate limiting, demo seed command (`cmd/seed`).
- ESLint, Prettier, Jest (unit + UI), live-API smoke test (`npm run e2e:api`), CI for API, mobile, e2e and Docker build.

### Removed
- Stripe billing, posts, comments, generic file records, organizations/tenants and the unused Tamagui dependency (out of scope for a free dating product).

### Changed
- Repository renamed from the `go-react-saas` boilerplate to Ember; JWT claims no longer carry an organization id.
