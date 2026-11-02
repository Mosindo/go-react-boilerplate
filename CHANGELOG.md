# Tandem Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.0.0]

### Added
- Dating domain: profiles, preferences, interests, photos (validated, resized, re-encoded), discovery deck with mutual preference filters, like/pass, race-free matching, match-based 1:1 chat with read receipts, notifications, block/report, account deletion.
- Realtime events over WebSocket with single-use tickets.
- Password recovery by e-mail code, session revocation checks, rate limiting, body-size limits.
- Migration tracking (`schema_migrations`), demo seed (`cmd/seed`).
- Expo app with onboarding, swipe deck, chat, notifications, profile management, dark mode; unit tests, ESLint/Prettier, browser E2E (`scripts/e2e-web.sh`).

### Removed
- Generic SaaS modules: Stripe billing, organisations/tenancy, posts, comments, files, generic chat, Tamagui.
