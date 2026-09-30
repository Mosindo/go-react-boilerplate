# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre mobile (iOS, Android, web) : on crée son profil, on découvre des personnes proches, on like ou on passe, et on discute en temps réel dès que l'intérêt est réciproque.

Elle est **entièrement gratuite** : aucun abonnement, aucun paywall, aucun « boost », aucun like limité, aucune publicité. Les seules limites existantes sont des protections anti-abus (rate limiting) dimensionnées pour ne jamais gêner un usage humain.

## Fonctionnalités

| Domaine | Ce qui est implémenté |
| --- | --- |
| Compte | Inscription (confirmation 18+), connexion, session persistante avec renouvellement transparent du jeton, récupération de compte par code à 6 chiffres envoyé par email, déconnexion, suppression définitive du compte et des données |
| Profil | Prénom, date de naissance (âge seul affiché, 18 ans minimum, non modifiable ensuite), genre, bio, métier, objectif relationnel, ville, centres d'intérêt (36, max. 10), 1 à 6 photos |
| Photos | Upload, remplacement, suppression, réordonnancement, photo principale ; validation par contenu réel, redimensionnement, ré-encodage (supprime les métadonnées EXIF/GPS), correction d'orientation, stockage privé, URLs signées à durée limitée |
| Localisation | Demandée uniquement sur action de l'utilisateur ; coordonnées arrondies à ~1 km avant envoi et avant stockage ; seule une distance arrondie (≥ 2 km, paliers de 5 km au-delà de 10 km) est montrée, désactivable |
| Découverte | Deck de cartes avec swipe et boutons accessibles, profil complet, filtres mutuels (genres, âges, distances), exclusion des profils déjà traités, bloqués, signalés par ≥ 3 personnes ou inactifs depuis 180 jours, classement pluggable |
| Matching | Like / passer définitifs, match créé atomiquement quand l'intérêt est réciproque (verrou par paire : jamais de doublon, même en cas de likes simultanés), annulation de match |
| Chat | Conversation créée au match, messages texte, temps réel (WebSocket), horodatage, lu/non lu et accusé de lecture, pagination, suppression locale d'une conversation, envoi optimiste avec reprise |
| Notifications | Notifications in-app (nouveau match, nouveau message dédupliqué par conversation), badges, poussées en temps réel |
| Confidentialité & sécurité | Mettre son profil en pause, masquer sa distance, bloquer, débloquer, signaler (bloque aussi), rate limiting, contrôle d'accès côté serveur partout |

Décisions produit et limites connues : voir [docs/DECISIONS.md](docs/DECISIONS.md).

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16 via `pgxpool`, SQL explicite (pas d'ORM), JWT HS256 + refresh tokens rotatifs, bcrypt, WebSocket (`gorilla/websocket`) avec diffusion via PostgreSQL `LISTEN/NOTIFY`, traitement d'images `golang.org/x/image`.
- **Mobile** : Expo SDK 54 (React Native 0.81, React 19), TypeScript strict, React Navigation 6 (stack + onglets), TanStack Query, `expo-image`, `expo-image-picker`, `expo-location`, `expo-secure-store`. Fonctionne aussi sur le web (`react-native-web`).
- **Qualité** : `go test` (unitaires + intégration PostgreSQL), `go vet`, `gofmt`, `tsc`, ESLint (`eslint-config-expo`), Jest (`jest-expo`), Playwright (E2E web), Maestro (E2E natif), GitHub Actions.
- **Infra** : Docker (image non-root), Docker Compose.

## Architecture

```text
.
├─ apps/mobile/                 Application Expo
│  ├─ App.tsx                   Providers (thème, React Query, session, toasts)
│  ├─ src/design/               Design system « Lueur » (tokens clair/sombre, composants)
│  ├─ src/lib/                  Client API, session, temps réel, formatage, cache
│  ├─ src/features/             auth, onboarding, discover, chat, notifications, profile, safety
│  ├─ src/navigation/           Navigateur racine et types de routes
│  ├─ e2e/web/                  Tests Playwright (parcours critique) + fixtures
│  ├─ e2e/maestro/              Parcours natif Maestro
│  └─ scripts/                  Smoke API, fixtures E2E, serveur statique web
├─ services/api/                API Go
│  ├─ cmd/api/                  Serveur HTTP (+ tests d'intégration HTTP de bout en bout)
│  ├─ cmd/seed/                 Données de démonstration (hors production)
│  └─ internal/
│     ├─ app/                   Composition root (câblage des features)
│     ├─ platform/              config, db (migrations), middleware, authtoken, ratelimit,
│     │                         realtime, storage, media (URLs signées), geo, mailer, httpx, errors
│     └─ features/              auth, profiles, photos, discovery, matching, chat,
│                               notifications, safety  (handler → service → repository)
├─ docker-compose.yml           PostgreSQL + API
└─ docs/                        ARCHITECTURE.md, DECISIONS.md
```

Chaque feature backend suit `handler.go` (HTTP fin) → `service.go` (règles métier) → `repository.go` (SQL). Détails : [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation

Prérequis : Node.js 20+, npm, Go 1.24+, Docker (ou PostgreSQL 16 local).

```bash
git clone <repo> && cd go-react-boilerplate
cp .env.example .env
# Renseigner JWT_SECRET, par exemple :
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env
```

### Variables d'environnement

Toutes sont documentées dans [.env.example](.env.example).

| Variable | Obligatoire | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256 (≥ 32 caractères, les valeurs d'exemple sont refusées). Sert aussi à dériver la clé de signature des URLs de photos. |
| `DATABASE_URL` | oui | Chaîne de connexion PostgreSQL. |
| `APP_ENV` | non (`development`) | `development`, `production` ou `test` (`test` désactive le rate limiting). |
| `PORT` | non (`8080`) | Port HTTP de l'API. |
| `UPLOAD_DIR` | non (`./data/uploads`) | Dossier privé des photos traitées. |
| `ALLOWED_ORIGINS` | pour le web | Origines CORS autorisées (client web). Inutile pour les apps natives. |
| `TRUSTED_PROXIES` | derrière un proxy | Proxies de confiance pour `X-Forwarded-For` (sinon l'IP réelle du pair est utilisée). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | en production | Envoi des codes de récupération. Sans SMTP : codes écrits dans les logs en développement, récupération désactivée en production. |
| `EXPO_PUBLIC_API_URL` | app mobile | URL publique de l'API vue par l'app (IP LAN sur téléphone physique, jamais `localhost`). |

Aucun secret n'est présent dans l'app mobile : seules des variables `EXPO_PUBLIC_*` publiques y sont lues.

## Base de données et migrations

- Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées automatiquement au démarrage, chacune **une seule fois** dans une transaction, avec suivi dans `schema_migrations` et un verrou consultatif (sûr avec plusieurs instances qui démarrent en même temps).
- `014_dating_schema.sql` crée le schéma de rencontre (profils, préférences, photos, intérêts, swipes, matches, conversations, messages, blocages, signalements, codes de réinitialisation) et retire les modules du boilerplate d'origine. `015_seed_interests.sql` insère le catalogue d'intérêts.
- Contraintes : clés étrangères avec suppression en cascade, `CHECK` sur toutes les valeurs énumérées et longueurs, unicité des swipes par paire, des matches par paire canonique, des positions de photos par utilisateur.

## Données de démonstration

```bash
cd services/api
go run ./cmd/seed                              # 16 membres de démo autour de Paris
go run ./cmd/seed -like-email vous@exemple.com # ils likent votre compte : likez-les pour matcher
go run ./cmd/seed -purge                       # supprime tous les membres de démo
```

Les comptes de démo utilisent le domaine réservé `@demo.invalid`, sont marqués `users.is_demo = true`, ont des photos abstraites générées (aucun visage réel), et la commande refuse de s'exécuter si `APP_ENV=production`. Mot de passe commun : `DemoPassword1`.

## Lancement local

### Avec Docker

```bash
export JWT_SECRET=$(openssl rand -base64 48)
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
docker compose exec api ./seed            # optionnel : données de démo
```

(Dans le conteneur, `APP_ENV` vaut `development` par défaut via Compose ; passez `APP_ENV=production` en production.)

### Sans Docker

```bash
cd services/api
export DATABASE_URL='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET=$(openssl rand -base64 48)
export ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api
```

### Application mobile

```bash
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://localhost:18080 npm start     # puis i / a / w
```

- Émulateur Android sans variable : l'app utilise `http://10.0.2.2:18080`.
- Téléphone physique : `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080` (jamais `localhost`), et vérifiez que le port n'est pas déjà utilisé.
- Web : `npm run web` (ajoutez l'origine à `ALLOWED_ORIGINS` côté API).

## Tests

```bash
# Backend : formatage, analyse statique, tests unitaires
cd services/api
gofmt -l . && go vet ./... && go test ./...

# Backend : + tests d'intégration (auth, profils, photos, likes, matchs dont likes
# simultanés, chat, permissions, blocage, signalement, suppression de compte, WebSocket)
DATABASE_URL_TEST='postgresql://postgres:postgres@localhost:5432/app_test?sslmode=disable' go test ./...

# Mobile : types, lint, tests unitaires
cd apps/mobile
npx tsc --noEmit && npx eslint . && npx jest

# Parcours complet via l'API (API démarrée)
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:api

# E2E UI web (Playwright) : inscription → profil → photo → découverte → like → match → chat temps réel
#   API démarrée avec APP_ENV=test et ALLOWED_ORIGINS=http://localhost:8099
EXPO_PUBLIC_API_URL=http://localhost:18080 npx expo export --platform web --output-dir dist
E2E_API_URL=http://localhost:18080 npx playwright test

# E2E natif (Maestro, appareil/émulateur + Expo Go)
npm run e2e:ui:run
```

Sans `DATABASE_URL_TEST`, les tests d'intégration sont ignorés (skip) et seuls les tests unitaires s'exécutent. La CI ([.github/workflows/ci.yml](.github/workflows/ci.yml)) lance tout, y compris Playwright et un démarrage de l'image Docker.

## Build

- **API** : `docker build -t lueur-api ./services/api` (image Alpine, utilisateur non-root, volume `/app/data/uploads`, binaires `api` et `seed`).
- **Web** : `EXPO_PUBLIC_API_URL=https://api.exemple.com npm run build:web` → `apps/mobile/dist/` (fichiers statiques).
- **iOS / Android** : via EAS Build (`npx eas build`), après avoir renseigné `EXPO_PUBLIC_API_URL` pour le profil de build. Les identifiants `app.lueur.mobile` et les textes d'autorisation (localisation, photos) sont définis dans `app.json`.

## Déploiement (recommandations)

1. PostgreSQL 16 managé, sauvegardes activées.
2. API en conteneur derrière un reverse proxy HTTPS : `APP_ENV=production`, `JWT_SECRET` fort (gestionnaire de secrets), `TRUSTED_PROXIES` = proxy, `ALLOWED_ORIGINS` = domaine web, SMTP configuré.
3. Volume persistant pour `UPLOAD_DIR` (ou implémentation `storage.Store` vers un stockage objet — voir DECISIONS).
4. Le proxy doit laisser passer les WebSockets sur `/realtime`.
5. Plusieurs instances : le temps réel fonctionne tel quel (LISTEN/NOTIFY) ; le rate limiting est par instance, ajoutez une limite au niveau du proxy.

## API (résumé)

| Méthode | Route | Description |
| --- | --- | --- |
| POST | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Authentification |
| POST | `/auth/password/forgot`, `/auth/password/reset` | Récupération de compte |
| GET / DELETE | `/me` | Compte courant / suppression (mot de passe requis) |
| GET / PATCH | `/profile` | Profil propriétaire |
| PUT | `/profile/preferences`, `/profile/interests`, `/profile/location` | Préférences, intérêts, position approximative (`DELETE` pour l'effacer) |
| GET / POST / PUT / DELETE | `/profile/photos`, `/profile/photos/order`, `/profile/photos/:id` | Photos |
| GET | `/media/photos/:id?exp&sig` | Photo via URL signée |
| GET | `/interests`, `/profiles/:userId` | Catalogue d'intérêts, profil public (si autorisé) |
| GET | `/discovery` | Prochains profils |
| POST / DELETE | `/swipes`, `/matches/:id` | Like/passer, annuler un match |
| GET / DELETE | `/conversations`, `/conversations/:id` | Conversations, suppression locale |
| GET / POST | `/conversations/:id/messages`, `/conversations/:id/read` | Messages (pagination par curseur), lecture |
| GET / POST | `/notifications`, `/notifications/:id/read`, `/notifications/read-all` | Notifications |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId`, `/reports` | Blocages et signalements |
| POST / GET | `/realtime/ticket`, `/realtime` | Ticket (60 s) puis WebSocket |

Erreurs : `{"error": "message", "code": "code_stable"}` avec le statut HTTP approprié.

## Principales décisions techniques

Résumé (détail dans [docs/DECISIONS.md](docs/DECISIONS.md)) :

- Conserver la base existante (Go/Gin/pgx, Expo) et son architecture en couches ; retirer ce qui contredisait le produit (billing Stripe, multi-tenant, posts/commentaires, annuaire d'emails).
- Temps réel par WebSocket + `LISTEN/NOTIFY` PostgreSQL : pas d'infrastructure supplémentaire, compatible multi-instances ; repli en polling côté client si la connexion tombe.
- Photos jamais servies publiquement : ré-encodage systématique et URLs signées HMAC.
- Localisation minimisée : grille ~1 km, distances arrondies.
- Classement de la découverte isolé derrière une interface `Scorer` pour faire évoluer l'algorithme sans toucher à la requête.

## Documentation complémentaire

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [docs/DECISIONS.md](docs/DECISIONS.md)
- [CONTRIBUTING.md](CONTRIBUTING.md) · [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) · [CHANGELOG.md](CHANGELOG.md)
