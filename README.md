# Hearth

**Hearth** est une application de rencontre moderne, **100 % gratuite** : pas
d'abonnement, pas de paywall, pas de « boost », pas de likes limités, pas de
publicité. Deux personnes peuvent créer un compte, compléter leur profil,
ajouter leurs photos, découvrir des profils compatibles près de chez elles,
liker ou passer, obtenir un match quand l'intérêt est réciproque, discuter en
temps réel, gérer leur confidentialité, bloquer ou signaler, et supprimer leur
compte.

Le projet est construit sur le socle `go-react-saas` (API Go + client Expo).

---

## Sommaire

- [Fonctionnalités](#fonctionnalités)
- [Stack technique](#stack-technique)
- [Architecture](#architecture)
- [Installation](#installation)
- [Variables d'environnement](#variables-denvironnement)
- [Base de données et migrations](#base-de-données-et-migrations)
- [Données de démonstration](#données-de-démonstration)
- [Lancement local](#lancement-local)
- [Tests et qualité](#tests-et-qualité)
- [Build](#build)
- [Déploiement](#déploiement)
- [API](#api)
- [Principales décisions techniques](#principales-décisions-techniques)
- [Limites connues](#limites-connues)

---

## Fonctionnalités

| Domaine | Ce qui est implémenté |
|---|---|
| Compte | Inscription, connexion, session persistante (refresh token rotatif), déconnexion, **récupération par code email**, changement de mot de passe, **suppression du compte et des données**. |
| Onboarding | 5 étapes : prénom / date de naissance (18 ans minimum) / genre → qui rencontrer, tranche d'âge, distance → photos → bio et centres d'intérêt → localisation approximative. |
| Profil | Prénom, âge, genre, bio (500 car.), ville, 10 centres d'intérêt max, 6 photos (principale, ordre, remplacement, suppression), critères de rencontre. |
| Découverte | Pile de profils avec **swipe** et boutons accessibles, navigation entre photos, profil complet, distance approximative. Compatibilité **mutuelle** (genres, âges, distance). |
| Matching | Like / passer, match uniquement si réciproque, pas de doublon, pas de profil déjà traité réaffiché, écran de célébration. |
| Chat | Conversations après match, messages texte **temps réel** (WebSocket), heure d'envoi, **lu / envoyé**, non-lus, pagination, suppression du match. |
| Notifications | Fil d'activité (nouveaux matchs, messages regroupés par conversation), badges, toasts en direct. |
| Confidentialité | Mettre le profil en pause, masquer sa distance, mettre à jour / effacer sa position. |
| Sécurité | Bloquer, débloquer, signaler (motif + détails, blocage automatique), suspension automatique après 3 signalements distincts. |

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, SQL explicite (pas d'ORM),
  JWT HS256 + refresh tokens, bcrypt, WebSocket (`coder/websocket`),
  traitement d'image (`golang.org/x/image`).
- **Client** : Expo SDK 54 / React Native 0.81 / React 19, TypeScript strict,
  React Navigation, TanStack Query, expo-image, expo-image-picker,
  expo-image-manipulator, expo-location, expo-secure-store. Cible iOS,
  Android et web.
- **Infra** : Docker, Docker Compose (PostgreSQL, API, Mailpit), GitHub Actions.
- **Qualité** : tests Go unitaires + intégration (Postgres réel), ESLint,
  Prettier, smoke API, E2E navigateur Playwright.

## Architecture

```text
apps/mobile/            Client Expo (src/api, hooks, navigation, screens, shared/ui…)
services/api/
  cmd/api/              Point d'entrée + tests d'intégration
  cmd/seed/             Données de démo (développement uniquement)
  internal/platform/    config, db, middleware, httpx, ratelimit, storage,
                        media, mailer, realtime, logger
  internal/features/    auth, profiles, photos, matching, chat, safety,
                        notifications  (handler → service → repository)
docker-compose.yml      postgres + api + mailpit
scripts/qa.js           Contrôle qualité local (identique à la CI)
```

Le détail (modèle de données, flux, choix de sécurité, points d'extension)
est dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation

Prérequis : **Node.js 20+**, **Go 1.24+**, **Docker** (recommandé) ou un
PostgreSQL 14+ local.

```bash
git clone <repo> hearth && cd hearth
cp .env.example .env
# Génère un secret fort et colle-le dans JWT_SECRET :
openssl rand -base64 48
cd apps/mobile && npm ci && cd ../..
```

## Variables d'environnement

Toutes sont documentées dans [`.env.example`](.env.example). Aucun secret n'est
versionné.

| Variable | Obligatoire | Rôle |
|---|---|---|
| `JWT_SECRET` | oui | Secret HS256 (32+ caractères aléatoires). Sert aussi à dériver la clé de signature des URLs de photos. |
| `DATABASE_URL` | oui | Connexion PostgreSQL (fournie automatiquement par Docker Compose). |
| `APP_ENV` | non (`development`) | `development`, `test` ou `production`. En production, SMTP est exigé et le seed refuse de s'exécuter. |
| `PORT` | non (`8080`) | Port HTTP de l'API (exposé en `18080` par Docker). |
| `UPLOAD_DIR` | non (`./data/uploads`) | Stockage des photos (volume Docker `uploads`). |
| `ALLOWED_ORIGINS` | web uniquement | Origines CORS autorisées pour le client web (ex. `http://localhost:8081`). |
| `TRUSTED_PROXIES` | derrière un proxy | IP/CIDR des reverse proxies autorisés à fournir `X-Forwarded-For`. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | prod | Envoi des codes de récupération. Docker Compose utilise Mailpit par défaut. |
| `EXPO_PUBLIC_API_URL` | client | URL de l'API intégrée au bundle mobile (publique : jamais de secret). |
| `SEED_PASSWORD` | non | Mot de passe des comptes de démo. |

## Base de données et migrations

Les migrations SQL sont dans `services/api/internal/platform/db/migrations`,
embarquées dans le binaire et **appliquées automatiquement au démarrage** de
l'API (et des tests). Chaque fichier n'est appliqué qu'une fois (table
`schema_migrations`, verrou consultatif pour les déploiements multi-instances).

Pour ajouter une migration : créer `NNN_description.sql` avec le numéro suivant.
`014_dating_core.sql` contient le schéma de rencontre (profils, préférences,
photos, centres d'intérêt, swipes, matchs, blocages, signalements,
récupération de mot de passe) avec contraintes, clés étrangères, index et
suppressions en cascade.

## Données de démonstration

Le seed crée des profils **clairement identifiés** : emails en
`@demo.hearth.invalid` (domaine réservé, non routable), drapeau
`users.is_demo`, bio commençant par « Profil de démonstration », photos
abstraites (dégradés, aucune vraie personne). Il refuse de tourner avec
`APP_ENV=production`.

```bash
# Avec Docker (exécuté dans le conteneur de l'API) :
docker compose exec api ./seed -count 24 -lat 48.8566 -lon 2.3522
# Faire liker ton compte par tous les profils de démo (match immédiat au like retour) :
docker compose exec api ./seed -like-email toi@exemple.fr
# Tout supprimer :
docker compose exec api ./seed -purge

# Sans Docker (depuis services/api, mêmes variables que l'API) :
go run ./cmd/seed -count 24
```

## Lancement local

### 1. Backend avec Docker (recommandé)

```bash
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

- API : `http://localhost:18080`
- Emails de récupération (Mailpit) : `http://localhost:8025`

Vérifier qu'aucun autre service n'occupe le port avant de déclarer l'API prête
(`docker compose ps`, puis `lsof -i :18080` ou `netstat -ano | findstr :18080`
sous Windows). En cas de conflit, changer le port hôte (`18081:8080`) dans
`docker-compose.yml` et aligner `EXPO_PUBLIC_API_URL`.

### 1 bis. Backend sans Docker

```bash
cd services/api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
export JWT_SECRET="$(openssl rand -base64 48)"
export ALLOWED_ORIGINS=http://localhost:8081
PORT=18080 go run ./cmd/api
```

Sans `SMTP_HOST` en développement, les codes de récupération s'affichent dans
les logs de l'API.

### 2. Application mobile

```bash
cd apps/mobile
# Téléphone physique (Expo Go) : utilise l'IP LAN de ta machine, jamais localhost
npm run start:phone                 # détecte l'IP LAN et définit EXPO_PUBLIC_API_URL
# ou explicitement :
EXPO_PUBLIC_API_URL=http://192.168.1.42:18080 npx expo start
# Émulateur Android : http://10.0.2.2:18080 par défaut ; simulateur iOS : localhost
npm run android | npm run ios
# Navigateur :
npm run web                         # http://localhost:8081 (ALLOWED_ORIGINS doit l'inclure)
```

Parcours à vérifier avant de considérer le client prêt : inscription →
onboarding → découverte → like → match → message → lecture → notifications.

## Tests et qualité

```bash
# Tout en une commande (identique à la CI) :
DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable node scripts/qa.js
# + smoke API contre une API qui tourne :
node scripts/qa.js --smoke
```

| Niveau | Commande | Couverture |
|---|---|---|
| Unitaires Go | `go test ./...` | règles d'âge, normalisation, préférences, arrondis de position, mots de passe, codes de récupération, rate limiter, JWT (alg, `exp`, session révoquée), URLs signées, stockage (path traversal), traitement d'image (EXIF, bombes, SVG/HTML), hub temps réel, SMTP. |
| Intégration Go | `go test ./cmd/api` (Postgres requis) | auth (rotation, révocation, changement), récupération, onboarding/validation, photos (limites, ordre, remplacement, signature), découverte mutuelle (genre, âge, distance, pause, distance masquée), likes/doublons/passes/match, **likes simultanés → un seul match**, permissions du chat, pagination, lu/non-lu, notifications regroupées, WebSocket, blocage, unmatch, signalements + suspension, suppression de compte. |
| Typecheck / lint / format | `npm run typecheck`, `npm run lint`, `npm run format:check` | `apps/mobile` |
| Smoke API | `npm run e2e:smoke` | parcours critique complet contre une API réelle, temps réel inclus. |
| E2E navigateur | `npm run build:web && npm run e2e:web` | deux navigateurs : inscription → onboarding (upload photo, géolocalisation) → découverte → like → match → chat temps réel → notifications. Captures dans `apps/mobile/e2e/.artifacts`. |

Les tests d'intégration sont ignorés (skip) si aucune base n'est disponible ;
la CI fournit PostgreSQL pour qu'ils s'exécutent toujours.

## Build

```bash
# API
cd services/api && go build -o bin/api ./cmd/api
docker build -t hearth-api services/api          # image non-root, healthcheck

# Web
cd apps/mobile && EXPO_PUBLIC_API_URL=https://api.exemple.fr npm run build:web   # → dist/

# iOS / Android (EAS Build)
npx eas-cli build --platform all                 # nécessite un compte Expo
```

## Déploiement

1. **PostgreSQL managé** (sauvegardes activées) → `DATABASE_URL`.
2. **API** : image Docker `services/api`, avec `APP_ENV=production`,
   `JWT_SECRET` fort, SMTP configuré, `TRUSTED_PROXIES` = IP du load balancer,
   `ALLOWED_ORIGINS` = domaine du client web. Monter un volume persistant sur
   `UPLOAD_DIR` (ou implémenter `storage.Storage` pour S3). Servir en **HTTPS**
   (HSTS envoyé automatiquement), WebSocket autorisé sur `/realtime`.
3. Les migrations s'appliquent au démarrage. Healthcheck : `GET /health`.
4. **Client** : `dist/` sur un hébergement statique, binaires natifs via EAS,
   avec `EXPO_PUBLIC_API_URL` pointant vers l'API publique.
5. Plusieurs instances d'API : le hub temps réel et le rate limiting sont en
   mémoire ; ajouter un bus partagé (LISTEN/NOTIFY ou Redis) et un rate
   limiting au niveau du proxy (voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)).

**Modération** : les signalements sont dans la table `reports`
(`status = 'open'`). Un compte signalé par 3 personnes distinctes est masqué
(`users.suspended_at`) en attendant une revue. Exemple :

```sql
SELECT reported_id, reason, details, created_at FROM reports WHERE status = 'open' ORDER BY created_at;
UPDATE reports SET status = 'reviewed' WHERE reported_id = '<id>';
UPDATE users SET suspended_at = NULL WHERE id = '<id>';   -- lever la suspension
```

## API

Toutes les routes (sauf auth publique, `/health` et `/media`) exigent
`Authorization: Bearer <accessToken>`. Erreurs : `{"error": "...", "code": "..."}`.

| Méthode | Route | Description |
|---|---|---|
| GET | `/health` | Santé (DB incluse). |
| POST | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Session. |
| POST | `/auth/password/forgot`, `/auth/password/reset` | Récupération par code. |
| GET / DELETE | `/me` | Compte courant / suppression (mot de passe requis). |
| POST | `/me/password` | Changement de mot de passe. |
| GET / PATCH | `/profile` | Mon profil (+ complétude). |
| PUT | `/profile/preferences`, `/profile/settings` | Critères, confidentialité. |
| PUT / DELETE | `/profile/location` | Position approximative. |
| GET / POST | `/profile/photos` | Lister / ajouter (multipart `photo`). |
| PUT / DELETE | `/profile/photos/:id` | Remplacer / supprimer. |
| PUT | `/profile/photos/order` | Réordonner. |
| GET | `/media/photos/:id?exp&sig` | Photo (URL signée expirante). |
| GET | `/interests`, `/profiles/:userId` | Catalogue, profil public. |
| GET | `/discover?limit=` | Profils à découvrir. |
| POST | `/swipes` | `{targetUserId, action: like|pass}` → `{matched, match}`. |
| GET / DELETE | `/matches`, `/matches/:id` | Matchs / unmatch. |
| GET / POST | `/conversations/:id/messages` | Messages (`before=` pour paginer) / envoyer. |
| POST | `/conversations/:id/read` | Accusé de lecture. |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId` | Blocages. |
| POST | `/reports` | Signalement (bloque aussi). |
| GET / POST | `/notifications`, `/notifications/unread-count`, `/notifications/read-all`, `/notifications/:id/read` | Notifications. |
| GET (WS) | `/realtime` | Événements temps réel (1er message : `{"type":"auth","token":"…"}`). |

## Principales décisions techniques

- **Conserver le socle** (Go/Gin/pgx, Expo/React Query) et le découpage
  `handler → service → repository` ; retrait des modules hors produit
  (facturation Stripe, posts, commentaires, votes, fichiers génériques,
  annuaire `/users` qui exposait les emails, Tamagui inutilisé).
- **Gratuité réelle** : aucune logique de quota produit ; seuls des rate
  limits anti-abus (ex. 120 swipes/min) protègent contre les bots.
- **Compatibilité mutuelle** dans une seule requête SQL, classement simple et
  remplaçable (intérêts communs, proximité, activité récente).
- **Match sans course** : verrou consultatif par paire + contraintes d'unicité.
- **Chat limité aux matchs** : toute opération vérifie l'appartenance, le
  match actif et l'absence de blocage ; sinon `404` (aucune fuite d'existence).
- **Unmatch / blocage** suppriment la conversation pour les deux ; le blocage
  n'est jamais révélé (même événement qu'un unmatch).
- **Position** arrondie à ~1 km côté client *et* serveur, jamais renvoyée,
  distance affichée en km entiers ; aucune géolocalisation en arrière-plan.
- **Photos** ré-encodées (métadonnées supprimées) et servies par URLs signées
  expirantes, compatibles avec le cache d'images.
- **Sessions vérifiées à chaque requête** : déconnexion, récupération et
  suppression prennent effet immédiatement.
- **Temps réel** par WebSocket authentifié au premier message (pas de jeton
  dans les URLs), avec reconnexion et rafraîchissement du jeton côté client.
- **Thème sombre chaleureux unique** (contrastes AA), identité originale.
- **Âge minimum 18 ans** vérifié côté serveur.

## Limites connues

- Notifications **push** système (APNs/FCM) non branchées : les notifications
  sont in-app et temps réel. Point d'extension documenté.
- Hub temps réel et rate limiting **en mémoire** (une instance) — voir
  Déploiement pour le multi-instances.
- Sur **web**, la session est gardée en mémoire (pas de `localStorage`, par
  sécurité) : recharger la page demande de se reconnecter.
- Pas d'interface d'administration : la modération se fait en SQL.
