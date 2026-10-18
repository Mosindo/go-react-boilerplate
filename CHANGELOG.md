# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Tendre: a free dating application built on the boilerplate (profiles, preferences, interests, photos, discovery,
  swipes, matches, real-time chat, notifications, blocking, reporting, account deletion, password recovery).
- Versioned database migrations (`schema_migrations`) and migration `014_dating_schema.sql`.
- Rate limiting, session revocation on every request, signed expiring photo URLs, image re-encoding.
- `cmd/seed` demo data (`@demo.invalid` accounts only).
- Integration tests on throwaway databases, Playwright browser journey, API smoke script, ESLint and Prettier.

### Changed
- Mobile app rewritten around the dating journey with a new visual identity; API client refreshes tokens itself.
- Infra: non-root image, uploads volume, CI runs formatting, vet, race tests, lint, typecheck and the browser journey.

### Removed
- Stripe billing and subscriptions, multi-tenant organizations, posts/comments/votes, metadata-only files module and the
  global user directory (their tables are dropped by migration 014).

### Pending manual cleanup
The environment used to produce this change refused file deletions, so the following are **unreferenced** but still in
the tree and should be deleted with `git rm -r`:
- `services/api/internal/features/{users,posts,comments,billing,files}`
- `docs/agents/payments-agent.md`, `.codex/config.toml` (Stripe MCP reference)
- `apps/mobile/src/theme`, `apps/mobile/tamagui.config.ts` and the `tamagui` / `@tamagui/babel-plugin` dependencies
- unused design-system leftovers: `apps/mobile/src/shared/ui/{Avatar,Badge,ListItem,MessageItem,NotificationItem}.tsx`
  and `apps/mobile/src/shared/layout/{Header,Section,ScreenContainer}.tsx` (remove their lines from the `index.ts` files)

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
