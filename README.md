# Ember — une application de rencontre 100 % gratuite

Ember est une application de rencontre complète (mobile + web) : créer un profil, ajouter des photos,
découvrir des profils, liker ou passer, matcher, discuter en temps réel, bloquer et signaler.

**Gratuit pour de vrai** : aucun abonnement, aucun paywall, aucun « boost », aucun like limité,
aucune publicité. Rien n'est verrouillé dans le code (il n'existe aucun modèle de facturation).

Le dépôt est issu du boilerplate `go-react-saas`; l'API et le client ont été remplacés par un domaine de
rencontre en gardant la plateforme (auth, middleware, migrations, Docker, CI).

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | inscription, connexion, jetons d'accès courts + refresh rotatif, récupération de mot de passe (e-mail), changement de mot de passe, déconnexion, suppression de compte (cascade complète) |
| Profil | prénom, date de naissance (18+ vérifié côté serveur), genre, bio, ville, 10 centres d'intérêt max, position approximative |
| Préférences | genres recherchés, tranche d'âge, distance max (ou « partout ») |
| Photos | 6 max, ordre, photo principale, remplacement, suppression ; JPEG/PNG vérifiés par contenu, redimensionnés et ré-encodés (EXIF/GPS supprimés), servies uniquement à des utilisateurs authentifiés autorisés |
| Découverte | deck à swiper (geste **ou** boutons accessibles), parcours des photos, profil détaillé, filtres mutuels (genre, âge, distance), jamais deux fois le même profil |
| Matching | like réciproque = match ; atomique, sans doublon, sans match incohérent ; annulation de match |
| Chat | temps réel (WebSocket), heure, lu / non lu, accusés de lecture, pagination, suppression locale de conversation |
| Notifications | matchs et messages (in-app + temps réel + badges) |
| Sécurité | blocage (bidirectionnel, met fin au match), signalement (6 motifs), rate limiting, autorisations serveur partout |
| Confidentialité | pause du profil, masquer la distance, distance floutée, coordonnées jamais exposées |

## Stack

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgx/v5` (pool partagé, SQL explicite, sans ORM), JWT HS256, bcrypt, `gorilla/websocket`, `golang.org/x/image`
- **Client** : Expo SDK 54 / React Native 0.81 (iOS, Android **et web**), TypeScript strict, React Navigation, TanStack Query
- **Infra** : Docker, Docker Compose, GitHub Actions

## Architecture

```text
services/api/
├─ cmd/api/            point d'entrée + composition (events.go relie chat/matchs → notifications + temps réel)
├─ cmd/seed/           comptes de démonstration (explicite, purgeable)
└─ internal/
   ├─ platform/        config, db (+ migrations), middleware, ratelimit, mailer, httpx, logger
   └─ features/        handler → service → repository (+ model, routes) par domaine :
      auth · profiles · photos · discovery (feed, swipes, matchs) · chat · safety · notifications · realtime
apps/mobile/
├─ App.tsx
└─ src/  api/ (client + endpoints typés) · auth/ · realtime/ · navigation/ · screens/ · features/ · ui/ · theme/ · lib/
docs/ARCHITECTURE.md   détails, modèle de données, décisions
```

Règles : handlers minces, logique dans les services, SQL uniquement dans les repositories, un seul serveur HTTP (Gin),
aucun accès réseau direct depuis l'UI (tout passe par `src/api`). Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Démarrage rapide

Prérequis : Docker, Node 20+ (22 recommandé).

```bash
cp .env.example .env
# renseignez JWT_SECRET :  openssl rand -hex 32
docker compose up -d --build        # PostgreSQL + API sur http://localhost:18080

# (optionnel) comptes de démonstration  — mot de passe : demo-password-1
docker compose exec api ./seed -confirm      # lea@demo.invalid, thomas@demo.invalid, …
docker compose exec api ./seed -purge        # les supprimer

cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run web     # client web sur http://localhost:8081
```

> Les migrations s'appliquent automatiquement au démarrage de l'API (table `schema_migrations`, verrou consultatif).

### Sur un téléphone physique

Ne jamais utiliser `localhost` : utilisez l'IP LAN de votre machine.

```bash
cd apps/mobile
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npx expo start
# scannez le QR code avec Expo Go
```

Avant de déclarer l'app « prête », vérifiez : `docker compose ps`, `curl http://<IP_LAN>:18080/health`, puis
déroulez le parcours : inscription → profil → photo → découverte → like → match → message.
Si le port 18080 est pris, changez `API_HOST_PORT` dans `.env` et `EXPO_PUBLIC_API_URL`.

### Sans Docker (développement)

```bash
createdb app
cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export ALLOWED_ORIGINS=http://localhost:8081      # uniquement pour le client web
PORT=18080 go run ./cmd/api
go run ./cmd/seed -confirm                         # optionnel
```

## Variables d'environnement

Voir [.env.example](.env.example) (aucun secret n'est committé).

| Variable | Requis | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | ≥ 32 caractères aléatoires ; les valeurs « placeholder » sont refusées |
| `DATABASE_URL` | oui (hors compose) | URL PostgreSQL |
| `PORT` | non | port d'écoute (8080) |
| `ALLOWED_ORIGINS` | web | origines navigateur autorisées (CORS), séparées par des virgules |
| `APP_BASE_URL` | non | URL publique du client web (liens e-mail) |
| `TRUSTED_PROXIES` | derrière un proxy | IP/CIDR autorisés à fournir `X-Forwarded-For` (sinon ignoré) |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | non | envoi des e-mails de récupération ; **sans `SMTP_HOST`, le code est écrit dans les logs de l'API (dev uniquement)** |
| `APP_ENV` | non | `production` empêche `cmd/seed` de s'exécuter |
| `EXPO_PUBLIC_API_URL` | client | URL de l'API, injectée au build (ne contient aucun secret) |

## Base de données et migrations

- `services/api/internal/platform/db/migrations/*.sql` sont embarquées dans le binaire et appliquées dans l'ordre,
  chacune dans une transaction, une seule fois (`schema_migrations`).
- Pour faire évoluer le schéma : ajoutez `002_xxx.sql`. Ne modifiez jamais une migration déjà déployée.
- Tables : `users, sessions, password_resets, profiles, preferences, interests, user_interests, photos, swipes,
  matches, conversations, conversation_participants, messages, blocks, reports, notifications`
  (contraintes, clés étrangères `ON DELETE CASCADE`, unicités et index décrits dans `docs/ARCHITECTURE.md`).
- Les 20 centres d'intérêt sont des données de référence insérées par la migration (pas de données de démo).

## Tests et qualité

```bash
# API  (les tests d'intégration ont besoin d'une base JETABLE : ils suppriment le schéma public)
createdb app_test
cd services/api
gofmt -l . ; go vet ./...
DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable' go test ./...
go build ./cmd/api ./cmd/seed

# Client
cd apps/mobile
npm ci && npx tsc --noEmit && npm run lint && npm run format:check && npm test

# E2E navigateur (web build + API lancés, voir l'en-tête de apps/mobile/e2e/web-smoke.js)
npx playwright install chromium && npm run e2e:web
```

Sans `DATABASE_URL_TEST`, les tests d'intégration sont ignorés (les tests unitaires tournent toujours).
La CI (`.github/workflows/ci.yml`) exécute tout cela avec un service PostgreSQL, build le bundle web et démarre la stack Docker.

Couverture : validation des entrées, âge 18+, règles de découverte (genre/âge/distance/blocage/pass), absence de doublons et
concurrence des likes simultanés, permissions de chat, accusés de lecture, blocage/signalement, notifications, WebSocket,
uploads dangereux, rate limiting, suppression de compte en cascade.

## Déploiement

1. Construire l'image : `docker build -t ember-api services/api` (binaire statique, utilisateur non-root, `/health`).
2. Fournir `DATABASE_URL`, `JWT_SECRET`, `ALLOWED_ORIGINS`, `SMTP_*`, `TRUSTED_PROXIES`, `APP_ENV=production`.
3. Mettre l'API derrière un reverse proxy **TLS** (le WebSocket `/ws` doit être proxifié avec l'en-tête `Upgrade`).
4. Web : `EXPO_PUBLIC_API_URL=https://api.example.com npx expo export --platform web` puis servir `dist/` (fallback SPA).
5. Mobile : `eas build` avec `EXPO_PUBLIC_API_URL` défini. Les identifiants `app.ember.dating` sont à adapter.
6. Plusieurs instances : le hub temps réel et le rate limiter sont en mémoire — voir « Limites connues ».

## Décisions techniques principales

- **On garde Go/Gin/pgx et Expo** : base saine, rien à remplacer. Supprimés : Stripe/billing, organisations multi-tenant,
  posts, commentaires, fichiers génériques, Tamagui (inutilisé), anciens scripts PowerShell. Ces suppressions étaient
  explicitement autorisées pour cette transformation.
- **Schéma réinitialisé** (`001_init.sql`) : le boilerplate n'avait pas de production à préserver.
- **Photos en base (`bytea`)** : pas de volume à gérer, suppression en cascade garantie, contrôle d'accès dans la même
  requête SQL. Chaque photo est limitée à ~1280 px. Pour de gros volumes, déplacez les octets vers un stockage objet
  privé en conservant la table `photos` et la route `/photos/:id`.
- **Ré-encodage systématique en JPEG** : supprime EXIF/GPS et charges polyglottes ; remplacer une photo émet un nouvel `id`
  (caches client invalidés sans logique supplémentaire).
- **Géolocalisation minimale** : l'appareil envoie sa position, le serveur n'en garde qu'une version arrondie (~1 km),
  qui n'est jamais renvoyée ; les distances affichées sont floutées (≤ 10 km : arrondi au km, au-delà : paliers de 5 km).
  Position refusée → pas de filtre de distance (« partout »).
- **Swipes définitifs et idempotents** : un like/pass ne peut pas être changé ; le doublon renvoie l'état existant.
  Un verrou consultatif par paire rend le match atomique même si les deux likent simultanément.
- **Recommandation** : un score SQL unique (`rankScore` dans `discovery/repository.go`) — centres d'intérêt communs, déjà
  liké, activité récente, proximité — isolé pour être remplacé sans toucher aux handlers.
- **Temps réel** : WebSocket avec ticket à usage unique (30 s) pour ne jamais mettre le JWT dans une URL. La base reste la
  source de vérité : le client re-synchronise à chaque (re)connexion.
- **Sessions vérifiées côté serveur** : chaque requête contrôle que la session existe ; déconnexion, changement de mot de passe et
  suppression de compte prennent effet immédiatement (pas d'attente de l'expiration du jeton de 15 min).
- **Chat = 1 conversation par match** ; la fin d'un match (annulation/blocage) coupe l'accès pour les deux ; l'API répond 404 (jamais 403)
  pour ne rien révéler sur les conversations d'autrui.
- **Notifications** : in-app + temps réel. Pas de notifications push OS (nécessiterait un compte APNs/FCM) — voir « Limites ».
- **Modération** : les signalements sont stockés (`reports`, statut `open`). Pas de back-office : traitez-les en SQL ou branchez un outil
  (`SELECT * FROM reports WHERE status='open'`). Supprimer un compte supprime aussi les signalements qu'il a émis ou reçus.
- **Interface en français**, sans framework i18n (les chaînes sont dans les écrans) ; thème clair/sombre automatique.
- **E-mail à l'inscription non vérifié** : l'adresse n'est utilisée que pour la connexion et la récupération.
  L'inscription répond `409` si l'e-mail existe (compromis d'ergonomie) ; la récupération, elle, ne révèle jamais l'existence d'un compte.

## Limites connues

- Hub WebSocket et rate limiter **en mémoire** : une seule instance d'API, ou ajoutez un fan-out partagé (LISTEN/NOTIFY, Redis).
- Pas de notifications push natives, pas de vérification d'e-mail ni de vérification d'identité/âge au-delà de la date déclarée.
- Pas de back-office de modération (voir ci-dessus).
- Les tests UI natifs (iOS/Android) ne sont pas automatisés ici ; le parcours critique est couvert dans un navigateur via le build web.

## Licence

Voir le dépôt.
