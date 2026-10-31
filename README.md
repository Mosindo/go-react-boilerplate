# Alma — application de rencontre 100 % gratuite

Alma permet à deux personnes de se rencontrer : créer un compte et un profil, ajouter des photos, découvrir des profils compatibles, liker ou passer, obtenir un **match** quand l'intérêt est réciproque, puis discuter en **temps réel**.

**Gratuit pour de vrai** : aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de likes, aucune publicité. (Seul un limiteur anti-abus technique existe, ex. 120 swipes/minute.)

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription (18 ans minimum vérifié côté serveur), connexion, sessions avec refresh token rotatif, récupération de mot de passe par code e-mail, déconnexion, suppression définitive du compte |
| Profil | Prénom, âge (calculé, la date de naissance n'est jamais exposée), genre, bio, ville, centres d'intérêt, préférences (genres, âge, distance), visibilité (apparaître ou non, afficher la distance ou non) |
| Photos | Jusqu'à 6 photos, upload, remplacement, suppression, ordre et photo principale ; validation par décodage réel, redimensionnement (1280 px), ré-encodage JPEG (supprime EXIF/GPS) ; accès protégé par authentification |
| Découverte | Flux paginé par lots, filtré par préférences **mutuelles** (genre, âge, distance), sans profils déjà traités ni bloqués ; swipe ou boutons ; profil complet |
| Matching | Like / pass uniques (idempotents), match créé exactement une fois même en cas de likes simultanés, « ils vous ont liké » |
| Chat | Conversation créée au match, messages texte, temps réel (WebSocket), horodatage, non lus / lu, liste des conversations, suppression locale, annulation du match |
| Sécurité | Blocage, signalement (avec blocage optionnel), contrôle d'accès serveur partout, rate limiting, validation stricte |
| Notifications | Match, message, like (liste in-app, compteur, marquer comme lu) |
| Géolocalisation | Seule une zone d'environ 1 km est stockée ; les autres voient une distance approximative (paliers de 1 km puis 5 km) |

## Stack

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256, bcrypt, SQL explicite (pas d'ORM), `gorilla/websocket`
- **Mobile** : Expo / React Native, TypeScript strict, React Navigation, TanStack Query, `expo-image`, `expo-image-picker`, `expo-location`
- **Infra** : Docker, Docker Compose, GitHub Actions

## Architecture

```text
.
├─ apps/mobile/                  # client Expo
│  └─ src/
│     ├─ api/                    # client HTTP (refresh auto du token), endpoints typés
│     ├─ auth/  realtime/        # session, WebSocket + mise à jour du cache
│     ├─ components/ screens/    # UI réutilisable, écrans
│     ├─ navigation/ lib/ store/ # navigation, validation/format, stockage sécurisé des tokens
│     └─ shared/                 # design system (tokens, Text, Button, Input…)
├─ services/api/
│  ├─ cmd/api/                   # serveur
│  ├─ cmd/seed/                  # données de démonstration (dev uniquement)
│  └─ internal/
│     ├─ app/                    # composition du routeur + tests d'intégration
│     ├─ platform/               # config, db+migrations, middleware, logger, realtime, mail, validate, httpx
│     └─ features/               # auth, profiles, photos, matching, chat, safety, notifications
│        └─ <feature>/           # handler → service → repository, model, routes
├─ docker-compose.yml            # postgres + api
└─ docs/ARCHITECTURE.md
```

Règles : handlers fins, logique dans les services, SQL uniquement dans les repositories, un seul serveur HTTP (Gin), aucun appel réseau direct depuis l'UI (tout passe par `apps/mobile/src/api`).

## Prérequis

- Go ≥ 1.24, Node ≥ 20 (22 recommandé), Docker (ou un PostgreSQL 14+ local)
- Un téléphone avec Expo Go, ou un émulateur Android / simulateur iOS

## Installation et lancement local

```bash
git clone <repo> && cd <repo>
cp .env.example .env
# éditez .env : JWT_SECRET=$(openssl rand -hex 32)

# 1. API + base de données (migrations appliquées automatiquement au démarrage)
docker compose --env-file .env up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# 2. Données de démonstration (facultatif, jamais en production)
docker compose --env-file .env exec api ./seed           # 12 profils fictifs @demo.invalid
# après avoir créé VOTRE compte dans l'app :
docker compose --env-file .env exec api ./seed -like-me vous@example.com   # ils vous likent → likez-les pour matcher

# 3. Application mobile
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<IP_LAN_DE_VOTRE_PC>:18080 npx expo start
```

> **Téléphone physique : n'utilisez jamais `localhost`.** Utilisez l'IP locale de votre machine. Émulateur Android : `http://10.0.2.2:18080`. Simulateur iOS : `http://localhost:18080`. Vérifiez qu'aucun autre service n'occupe le port (`docker compose ps`, `curl http://<IP>:18080/health`) ; en cas de conflit, changez le port hôte dans `docker-compose.yml` et `EXPO_PUBLIC_API_URL`.

Sans Docker : démarrez PostgreSQL, puis `cd services/api && DATABASE_URL=… JWT_SECRET=… go run ./cmd/api` (et `go run ./cmd/seed`).

Comptes de démonstration : `camille@demo.invalid`, `julien@demo.invalid`, … mot de passe `demo-password-123` (ou `DEMO_PASSWORD`). Supprimer toutes les données de démo : `./seed -reset`.

## Variables d'environnement

Voir [.env.example](.env.example). Obligatoires : `JWT_SECRET` (≥ 32 caractères aléatoires, les valeurs « placeholder » sont refusées) et `DATABASE_URL`. `PORT` (8080 par défaut). `APP_ENV=production` exige `SMTP_HOST`. Aucun secret ne doit se trouver dans l'application mobile : seule `EXPO_PUBLIC_API_URL` (publique) y est lue.

## Base de données et migrations

Les migrations sont embarquées (`services/api/internal/platform/db/migrations/*.sql`) et appliquées dans l'ordre au démarrage de l'API ; la table `schema_migrations` garantit qu'une migration ne s'exécute qu'une fois (verrou consultatif pour les démarrages concurrents). Pour ajouter une évolution : créer `00N_nom.sql`, ne jamais modifier une migration déjà publiée.

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `likes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Contraintes d'unicité, clés étrangères `ON DELETE CASCADE` (la suppression d'un compte efface tout ; les signalements émis restent avec `reporter_id` à `NULL`) et index sur les chemins chauds (`messages(conversation_id, created_at)`, `notifications(user_id, created_at)`, `likes(target_id, …)`…).

> Ce dépôt est une refonte : une base créée avec l'ancien boilerplate (posts/billing) doit être recréée (`docker compose down -v`).

## Tests, qualité, build

```bash
# API (les tests d'intégration ont besoin de PostgreSQL ; sinon ils sont ignorés)
cd services/api
export DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable
gofmt -l .            # doit n'afficher aucun fichier
go vet ./...
go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci
npm run typecheck && npm run lint && npm run format:check && npm test
npx expo export --platform android      # vérifie que le bundle se construit
```

Les tests d'intégration (`services/api/internal/app/app_test.go`) couvrent le parcours *inscription → profil → photos → découverte → like → match → conversation*, les permissions (un tiers ne peut ni lire ni écrire dans une conversation), blocage, signalement, suppression de compte, concurrence des likes réciproques, validation des uploads, filtres âge/distance/visibilité, WebSocket. Il n'y a pas de test E2E d'interface automatisé (voir « Limites »).

## Déploiement

1. Provisionner PostgreSQL géré et un secret `JWT_SECRET` (`openssl rand -hex 32`).
2. Construire l'image : `docker build -t alma-api services/api` (utilisateur non-root, healthcheck `/health`).
3. Variables : `APP_ENV=production`, `DATABASE_URL`, `JWT_SECRET`, `SMTP_*`, `TRUSTED_PROXIES` (IP de votre reverse proxy), éventuellement `ALLOWED_ORIGINS`.
4. Placer l'API derrière un reverse proxy **HTTPS** (TLS obligatoire pour les tokens et les photos) qui autorise les WebSockets sur `/ws`.
5. Mobile : `EXPO_PUBLIC_API_URL=https://api.example.com` puis build via EAS (`eas build`).

## Décisions techniques

- **Pas de réécriture inutile** : conservation de Gin, pgx, JWT HS256 strict (expiration obligatoire, claims `uid`/`sid`), sessions à refresh rotatif et de l'architecture `handler → service → repository`. Supprimés car contraires au produit ou inutiles : billing/Stripe (aucun paywall), organisations multi-tenant, posts/commentaires/fichiers génériques, Tamagui (non utilisé).
- **Photos en base (`bytea`)** : zéro dépendance de stockage objet, suppression de compte atomique (cascade), accès contrôlé par la même logique de visibilité que les profils. Pour de gros volumes, remplacer le repository par un stockage objet signé sans toucher au reste.
- **Temps réel** : WebSocket authentifié par une première trame (jamais de token dans l'URL) ; l'envoi passe par REST (fiable, limité en débit) et le serveur pousse `message`/`match`/`read`. Le hub est en mémoire (une instance) ; pour plusieurs instances, brancher Postgres `LISTEN/NOTIFY` ou Redis derrière l'interface `realtime.Publisher`.
- **Likes** : la première décision est définitive (un pass ne devient pas un like) ; un verrou consultatif par paire garantit un seul match même en cas de likes simultanés.
- **Visibilité** : un profil masqué n'est visible que par les personnes déjà matchées ou qui l'ont liké ; un blocage coupe tout (profil, photos, chat) dans les deux sens.
- **Localisation** : coordonnées arrondies à ~1 km avant stockage, distance affichée en paliers.
- **Anti-énumération** : « mot de passe oublié » répond toujours 202, la connexion exécute un bcrypt factice si l'e-mail est inconnu, une conversation inaccessible répond toujours 404.
- **Thème** : un seul thème clair chaleureux (corail / crème) ; le mode sombre n'est pas livré.

## Limites connues

- Pas de notifications push hors application (uniquement in-app + temps réel) : cela demande des identifiants FCM/APNs propres à votre compte.
- Pas de modération back-office : les signalements sont enregistrés (`reports`, statut `open`) et à traiter par SQL ou un futur outil d'administration.
- Les jetons d'accès restent valides 15 minutes après un changement de mot de passe ou une suppression de compte (les refresh tokens sont, eux, révoqués immédiatement).
- Les tests d'interface sont des tests unitaires/composants Jest ; aucun E2E sur appareil n'est automatisé, et le build Docker est vérifié par la CI (pas exécutable dans tous les environnements).
- Le filtre de distance utilise la formule de Haversine en SQL sans index spatial : suffisant jusqu'à quelques dizaines de milliers de profils, puis envisager PostGIS.
