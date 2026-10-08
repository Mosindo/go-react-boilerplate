# Changelog

## Unreleased — Lien (application de rencontre)

- Le boilerplate devient une application de rencontre gratuite : profils, photos, préférences, découverte, matchs, chat temps réel, notifications, blocage, signalement, suppression de compte.
- API : nouveau schéma (`internal/platform/db/schema`, migrations versionnées), modules `profiles`, `photos`, `discovery`, `safety`, WebSocket, rate limiting, récupération de mot de passe.
- App : refonte complète (design system clair/sombre, deck de swipe, chat, onboarding), build web, tests Jest et Playwright.
- Retirés du chemin d'exécution : Stripe/abonnements, organisations, posts, commentaires, fichiers génériques.

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Placeholder for upcoming unreleased changes.

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
