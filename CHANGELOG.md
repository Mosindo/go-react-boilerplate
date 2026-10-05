# Lueur Changelog

All notable changes to this project are documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- The dating product: profiles with photos, interests and preferences; discovery with mutual age / gender /
  distance filtering; likes, passes and atomic matching; match-gated real-time chat with read receipts;
  notifications; blocking, reporting, profile pause; password recovery; account deletion.
- WebSocket hub, signed photo URLs, image processing pipeline, rate limiting, tracked SQL migrations.
- Seed tool for clearly labelled demo accounts (`go run ./cmd/seed`, `--purge`).
- Mobile: onboarding wizard, swipe deck, matches, chat, settings; light/dark "Lueur" design system.
- Tests: Go unit and PostgreSQL integration tests, Vitest unit tests, Playwright browser E2E, Maestro flow, API smoke.
- CI for backend, mobile, browser E2E and Docker; ESLint and Prettier for the mobile app.

### Changed
- Product repositioned from the generic `go-react-saas` boilerplate to a free dating app.
- Migrations are now applied once and recorded in `schema_migrations` (they used to re-run on every boot).
- Notifications can no longer be created by clients; they are emitted by the server.
- Access tokens no longer carry an organisation claim; sessions are checked on every request.
- Replaced Tamagui with the in-repo design system.

### Removed
- Stripe billing, organisations, posts, comments, votes and the generic user directory were unwired from the
  product (source files pending deletion, see README "Legacy code").
