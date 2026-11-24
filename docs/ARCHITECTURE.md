# Architecture de Lueur

Ce document décrit l'organisation du code, les flux principaux et les choix de sécurité. Le [README](../README.md) couvre l'installation et l'exploitation.

## Vue d'ensemble

```text
App Expo (iOS / Android / web)
  │  REST JSON (Bearer access token)          WebSocket (ticket 60 s)
  ▼                                            ▼
API Go (Gin, un seul serveur HTTP) ──── hub temps réel en mémoire
  │  pgxpool (pool unique)                     │
  ▼                                            ▼
PostgreSQL 16                         stockage photos (disque, interface Store)
```

## Backend (`services/api`)

### Couches

| Dossier | Rôle |
| --- | --- |
| `cmd/api` | Composition : config, pool, migrations, stockage, mailer, hub, routeur. Arrêt propre sur SIGTERM. |
| `cmd/seed` | Profils de démonstration (refusé en production). |
| `internal/platform/config` | Lecture et validation des variables d'environnement. |
| `internal/platform/db` | Connexion avec retry, migrations embarquées suivies dans `schema_migrations`. |
| `internal/platform/errors` | Erreurs applicatives typées (`*Error{Status, Code, Message}`). |
| `internal/platform/httpx` | Helpers Gin : réponses d'erreur uniformes, binding, UUID, pagination. |
| `internal/platform/middleware` | Auth (JWT + session active), CORS, en-têtes de sécurité, request id, logs, métriques, rate limiting, taille de corps. |
| `internal/platform/tokens` | Signature / validation JWT HS256 (émetteur, audience, expiration obligatoire). |
| `internal/platform/media` | URLs signées HMAC expirantes pour les photos. |
| `internal/platform/storage` | Interface `Store` + implémentation disque (écriture atomique, protection path traversal). |
| `internal/platform/mailer` | SMTP, ou journalisation en développement. |
| `internal/platform/realtime` | Hub de connexions par utilisateur (`Publisher`). |
| `internal/platform/ratelimit` | Token bucket en mémoire. |
| `internal/features/*` | Domaines métier (voir ci-dessous). |

Chaque feature contient `model.go`, `repository.go` (SQL uniquement ici), `service.go` (règles métier, retourne des erreurs typées), `handler.go` (binding + appel service + réponse) et `routes.go`.

### Features

| Feature | Responsabilités | Dépend de |
| --- | --- | --- |
| `auth` | Inscription, connexion, refresh rotatif, logout, codes de récupération, suppression de compte, vérification de session pour le middleware | `photos` (purge des fichiers), hub (déconnexion) |
| `photos` | Upload, traitement d'image, ordre, remplacement, suppression, URLs signées, service des fichiers | storage, media |
| `profiles` | Profil, localisation arrondie, confidentialité, préférences, centres d'intérêt, cartes publiques (requêtes groupées) | `photos` |
| `matching` | Découverte (pool SQL + `Ranker`), swipes, création atomique match + conversation, annulation | `profiles`, `notifications`, hub |
| `chat` | Conversations, messages, lus/non lus, masquage local | `photos`, hub |
| `safety` | Blocages, signalements (avec blocage) | hub |
| `notifications` | Notifications serveur, compteurs, lecture | hub |
| `realtime` | Ticket WebSocket et connexion | `auth` (session), hub |

### Flux clés

**Like → match**
1. `POST /swipes` : le service vérifie profil complet de l'auteur, cible visible et non bloquée.
2. Transaction : verrou consultatif sur la paire triée, `INSERT … ON CONFLICT DO NOTHING` du swipe (doublon → 409), recherche du like réciproque.
3. Si réciproque : `INSERT` du match (paire ordonnée, unique), de la conversation et des deux participants.
4. Après commit : événement `match.created` et notification pour les deux membres.

**Découverte**
1. CTE `me` : profil + préférences du demandeur, boîte englobante calculée depuis la distance max.
2. Candidats : visibles, géolocalisés, avec photo, compatibles dans les deux sens (genre, âge, distance), non swipés, non bloqués, non mis à l'écart par la modération, hors démo en production.
3. Pool de 200 trié par activité, puis classement Go (`WeightedRanker`) et découpe à `limit`.
4. Cartes construites en 3 requêtes groupées (profils + distance, intérêts partagés, photos), quel que soit `limit` : pas de N+1.

**Message**
1. Vérification d'appartenance à la conversation (sinon 404 pour ne pas révéler l'existence).
2. Insertion + `last_message_at` + `last_read_at` de l'expéditeur dans une transaction.
3. Événement `message.created` aux deux participants ; `POST /read` publie `conversation.read` à l'autre.

### Sécurité

- **Authentification** : bcrypt (coût 12), access token 15 min (`aud=lueur-app`), refresh token aléatoire 256 bits stocké haché, rotation à usage unique ; comparaison à temps constant et hachage factice pour ne pas révéler l'existence d'un e-mail.
- **Autorisation** : chaque requête vérifie la session en base ; toutes les requêtes SQL filtrent par l'utilisateur authentifié (photos, conversations, notifications, matchs).
- **Entrées** : binding Gin + validation métier (e-mail normalisé, prénom, âge ≥ 18, énumérations, longueurs en runes, coordonnées), contraintes `CHECK` en base en seconde ligne, corps JSON limités à 64 Kio.
- **Fichiers** : signature binaire, bornes de dimensions et de poids (10 Mo), réencodage JPEG sans métadonnées, clés de stockage aléatoires générées serveur, service via URL signée + `Content-Security-Policy: default-src 'none'`.
- **Abus** : rate limiting par IP (auth, récupération, WebSocket) et par utilisateur (swipes, messages, uploads, signalements).
- **Vie privée** : position arrondie avant stockage, distance arrondie à l'affichage, e-mail et date de naissance jamais exposés aux autres membres, suppression de compte en cascade.
- **En-têtes** : `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, HSTS derrière HTTPS, `Cache-Control: no-store` par défaut.

## Frontend (`apps/mobile`)

| Dossier | Rôle |
| --- | --- |
| `src/api` | `client.ts` (timeout, erreurs `ApiError{status, code}`, refresh unique partagé entre requêtes concurrentes), `endpoints.ts` (seul endroit qui connaît les routes), `types.ts` (DTO). |
| `src/auth` | Stockage sécurisé des tokens (SecureStore ; `sessionStorage` sur le web de dev), `AuthProvider` (états `booting / signedOut / signedIn`). |
| `src/realtime` | Connexion WebSocket avec reconnexion exponentielle, intégration des événements dans le cache React Query, resynchronisation REST à la reconnexion. |
| `src/hooks` | Requêtes et mutations ; polling de secours uniquement si le temps réel est coupé. |
| `src/navigation` | Pile racine conditionnelle : non connecté → onboarding (tant que le profil est incomplet) → onglets + écrans empilés. |
| `src/screens` | Écrans par domaine. Aucun appel réseau direct : tout passe par `src/api`. |
| `src/components` | Kit UI (Button, TextField, Chip, Sheet, Toast, Stepper, Section…) et composants métier (SwipeCard, PhotoGrid, ReportSheet, LocationCard, formulaires de profil). |
| `src/theme/tokens.ts` | Design system unique. |

Gestion d'état : React Query pour l'état serveur (cache, pagination infinie, mises à jour optimistes du deck), état local React pour les formulaires, petits stores `useSyncExternalStore` pour les toasts et la célébration de match.

## Tests

| Niveau | Où |
| --- | --- |
| Unitaires Go | `*_test.go` à côté du code (validation, ranking, JWT, images, stockage, URLs signées, rate limit, hub). |
| Intégration Go | `cmd/api/main_integration_test.go` (vraie base, routeur complet, WebSocket réel). |
| Unitaires front | `apps/mobile/src/__tests__` (Jest). |
| Smoke API | `apps/mobile/scripts/e2e-smoke.js`. |
| E2E UI | `apps/mobile/e2e/web/critical-flow.mjs` (Playwright sur la version web). |
