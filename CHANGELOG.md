# Lueur Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-09-30

### Added
- Lueur, application de rencontre gratuite : inscription, récupération de compte par code email, onboarding guidé, profils, photos (upload sécurisé, réordonnancement, URLs signées), localisation approximative, découverte avec swipe, matching réciproque, chat temps réel (WebSocket) avec accusés de lecture, notifications in-app, blocage, signalement, suppression de compte.
- Migrations versionnées (`schema_migrations`) avec verrou consultatif ; schéma de rencontre (`014`) et catalogue d'intérêts (`015`).
- Commande `cmd/seed` de données de démonstration (hors production, domaine `@demo.invalid`, drapeau `is_demo`).
- Tests : unitaires Go et TypeScript, intégration Go sur PostgreSQL (dont likes simultanés et WebSocket), smoke API, E2E web Playwright, flux Maestro ; ESLint côté mobile ; CI réécrite.

### Changed
- JWT : claim `typ` (accès vs ticket temps réel), vérification de session active à chaque requête.
- Proxies de confiance explicites pour l'IP client ; rate limiting anti-abus.
- Image Docker non-root avec volume pour les photos.

### Removed (du routage et du schéma)
- Billing Stripe, multi-tenant `organizations`, posts, commentaires, votes, module `files`, annuaire `/users`.

### Fichiers hérités à supprimer
La suppression de fichiers n'a pas été effectuée sans validation explicite. Ces fichiers ne sont plus référencés par l'application ni par l'API (ils compilent toujours) :

- `services/api/internal/features/{billing,comments,files,posts,users}/`
- `apps/mobile/src/screens/`, `apps/mobile/src/shared/`, `apps/mobile/src/theme/`, `apps/mobile/src/hooks/useAuth.tsx`, `apps/mobile/src/store/tokenStore.ts`, `apps/mobile/src/api/{auth,client,endpoints,platform,types}.ts`, `apps/mobile/tamagui.config.ts`
- `docs/agents/payments-agent.md`, `.codex/config.toml`
- Dépendances associées : `tamagui`, `@tamagui/babel-plugin`, `@types/react-native`

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
