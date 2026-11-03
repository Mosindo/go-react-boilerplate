# Braise Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-11-03

### Added
- Free dating app on top of the former boilerplate: profiles, interests, photos, discovery, likes, matches,
  realtime chat (WebSocket), notifications, blocking, reporting, account deletion, password recovery.
- Tracked SQL migrations (`schema_migrations`) with an advisory lock; dating schema with constraints and indexes.
- Photo pipeline: content sniffing, size/dimension limits, re-encoding (strips EXIF), private authenticated serving.
- Per-IP / per-user rate limiting and a trusted-proxy setting.
- Demo seed (`cmd/seed`, `demo.invalid` accounts, refuses `APP_ENV=production`).
- Expo client rewritten: swipe deck, onboarding, likes, chat, settings, dark mode, French UI.
- Go integration tests (auth, profile, discovery, likes, concurrent match, chat permissions, blocks, deletion),
  mobile unit tests, ESLint + Prettier, browser E2E script.

### Removed
- Stripe billing, organizations/tenants, posts, comments, generic files/chat/notifications modules,
  Tamagui, PowerShell QA scripts. The product has no paid tier by design.

### Changed
- E-mail is trimmed/normalised before validation; login equalises timing for unknown accounts.
- Docker Compose is now `infra/docker-compose.yml` only (adds an uploads volume).
