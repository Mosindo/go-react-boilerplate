# Changelog

Format : [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-10-26

### Added
- Lien, application de rencontre gratuite : profils, préférences, localisation approximative, photos, découverte par swipe, matchs, chat temps réel, notifications, blocage, signalement, suppression de compte, récupération de mot de passe.
- Migrations suivies (`schema_migrations`), données de démonstration (`cmd/seed`), tests d'intégration PostgreSQL, tests Jest, scénario Playwright, CI complète.

### Changed
- Client Expo reconstruit (React Navigation 7, TanStack Query, thème clair/sombre, build web).
- Docker Compose unique à la racine, image API non-root avec healthcheck.

### Removed
- Stripe/abonnements, organisations multi-tenant, posts/commentaires/votes, fichiers génériques, Tamagui, scripts PowerShell obsolètes.
