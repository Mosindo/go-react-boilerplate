# go-react-saas Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Alma, application de rencontre 100 % gratuite : comptes, profils, photos, découverte (swipe), matching, chat
  temps réel (WebSocket), notifications, blocage, signalement, suppression de compte, récupération de mot de passe.
- Migrations suivies (`schema_migrations`), commande de seed de démonstration, tests d'intégration API,
  tests unitaires mobile, script `e2e:smoke`, CI (Postgres, lint, typecheck, bundle Metro, smoke Docker).

### Removed
- Modules posts, comments, files, users, billing (Stripe) et organisations multi-tenant ; thème Tamagui ;
  scripts PowerShell et Maestro obsolètes.

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
