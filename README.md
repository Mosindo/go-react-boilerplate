# Ember — une application de rencontre 100 % gratuite

Ember est une application de rencontre complète : créer un compte, composer un profil avec photos, découvrir des personnes près de soi, liker ou passer, matcher quand l'intérêt est réciproque, discuter en temps réel, gérer sa confidentialité, bloquer / signaler, et supprimer son compte.

**Gratuit pour de vrai** : aucun abonnement, aucun paywall, aucun boost payant, aucune limite artificielle de likes, aucune publicité. Il n'existe aucun code de facturation dans ce dépôt (les modules Stripe/organisations du boilerplate d'origine ont été retirés).

> Le dépôt est parti du boilerplate `go-react-saas` (Go + Gin + PostgreSQL + Expo). L'architecture `handler → service → repository` et la pile technique ont été conservées ; le domaine a été remplacé.

## Sommaire
- [Fonctionnalités](#fonctionnalités) · [Stack](#stack) · [Architecture](#architecture) · [Démarrage rapide](#démarrage-rapide)
- [Variables d'environnement](#variables-denvironnement) · [Base de données](#base-de-données-migrations-et-seed) · [Lancer en local](#lancer-en-local-sans-docker)
- [Tests et qualité](#tests-et-qualité) · [Build et déploiement](#build-et-déploiement) · [API](#api) · [Décisions techniques](#décisions-techniques) · [Limites connues](#limites-connues)

## Fonctionnalités

| Domaine | Ce qui est livré |
|---|---|
| Compte | Inscription (prénom, date de naissance, **18 ans minimum vérifiés côté serveur**), connexion, refresh avec rotation, déconnexion, récupération par code e-mail, changement de mot de passe, suppression de compte |
| Profil | Genre, bio, ville, centres d'intérêt (24 au choix, 8 max), préférences (genres recherchés, âge, distance), visibilité (pause du profil, masquer la distance) |
| Photos | Jusqu'à 6 ; JPEG/PNG détectés par le contenu ; recompression en JPEG (EXIF/GPS supprimés), redimensionnement, ordre, photo principale, remplacement, suppression ; fichiers jamais publics |
| Découverte | Filtrage 100 % serveur : orientation et âge **réciproques**, distance respectée **des deux côtés**, profils bloqués / déjà traités exclus ; les personnes qui t'ont déjà liké passent en premier |
| Matching | `like` / `pass` idempotents, un seul match par paire (contrainte + verrou de paire contre les likes simultanés), conversation créée avec le match, désappariement |
| Chat | Messages texte, temps réel (WebSocket), heure, lu / non lu, pagination, suppression locale d'historique, accès réservé aux deux participants |
| Notifications | Match et nouveau message (regroupés), en temps réel + liste + marquer comme lu |
| Sécurité | Blocage (symétrique, réversible), signalement (6 motifs), rate limiting, sessions révocables immédiatement |
| Géolocalisation | Coordonnées arrondies à ~1 km **avant stockage** ; seuls la ville et une distance approximative sont exposées |

## Stack

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256, bcrypt, `gorilla/websocket` (seule dépendance ajoutée), aucun ORM
- **Mobile** : Expo SDK 54 / React Native, TypeScript strict, React Navigation, TanStack Query, `expo-image`, `expo-image-picker`, `expo-location` (iOS, Android et **web**)
- **Infra** : Docker, Docker Compose, GitHub Actions

## Architecture

```text
.
├─ docker-compose.yml          # postgres + api (+ volume de photos)
├─ .env.example
├─ services/api/
│  ├─ cmd/api/                 # composition root (main.go, router.go) + tests d'intégration
│  ├─ cmd/seed/                # comptes de démonstration (jamais en production)
│  └─ internal/
│     ├─ platform/             # config, db (+ migrations/*.sql), middleware, realtime, storage, mailer, validate, httpx, errors, logger
│     └─ features/             # auth, profiles, photos, discovery, matches, chat, notifications, safety
│                              # chaque feature : handler.go → service.go → repository.go (+ model.go, routes.go)
├─ apps/mobile/
│  ├─ App.tsx
│  ├─ src/{api,auth,realtime,hooks,lib,theme,ui,navigation,features}
│  └─ scripts/                 # e2e-api.js, e2e-web.js, start-phone.js
└─ docs/                       # ARCHITECTURE.md, décisions, guides agents
```

Règles : handlers fins, pas de SQL hors des repositories, un seul serveur HTTP (Gin), un seul pool PostgreSQL, aucun accès réseau direct depuis les écrans (tout passe par `src/api`). Détails dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Démarrage rapide

Avec Docker (API + base) :

```bash
cp .env.example .env
# éditer .env : JWT_SECRET=$(openssl rand -hex 32)
docker compose up --build -d
curl http://localhost:18080/health            # {"status":"ok"}
docker compose exec api ./seed                # (optionnel) comptes de démonstration
```

Application mobile :

```bash
cd apps/mobile
npm ci
# Simulateur iOS / web : EXPO_PUBLIC_API_URL=http://localhost:18080 npx expo start
# Téléphone physique   : API_PORT=18080 npm run start:phone      # détecte l'IP du réseau local
```

> **Téléphone physique : n'utilise jamais `localhost`.** Utilise l'IP de ton ordinateur (`http://192.168.x.x:18080`). Vérifie avant : `curl http://<IP>:18080/health`, et qu'aucun autre service n'occupe le port (`docker compose ps`, `lsof -i :18080` / `netstat -ano | findstr :18080`). L'émulateur Android joint l'hôte via `10.0.2.2`.

Version web : `EXPO_PUBLIC_API_URL=http://localhost:18080 npm run web` (ajouter l'origine web, ex. `http://localhost:8081`, à `ALLOWED_ORIGINS` côté API).

Comptes de démonstration (après `seed`) : `camille@demo.invalid`, `louis@demo.invalid`, … mot de passe `demo-password-1`. Domaine `.invalid` (jamais livrable), marqués `users.is_demo`, photos générées (dégradés, pas de vraies personnes). Suppression : `docker compose exec api ./seed -purge`. Le seed refuse de tourner avec `APP_ENV=production`.

## Variables d'environnement

Toutes documentées dans [`.env.example`](.env.example). Aucune valeur secrète n'est versionnée.

| Variable | Requis | Rôle |
|---|---|---|
| `JWT_SECRET` | oui | Secret HS256, ≥ 32 caractères, les valeurs-exemples sont refusées |
| `DATABASE_URL` | oui (hors compose) | Chaîne de connexion PostgreSQL |
| `PORT` | non (8080) | Port d'écoute |
| `APP_ENV` | non | `production` exige SMTP et désactive le seed |
| `UPLOAD_DIR` | non | Dossier des photos (volume Docker `uploads`) |
| `ALLOWED_ORIGINS` | non | Origines web autorisées (CORS + WebSocket) ; vide = aucune |
| `TRUSTED_PROXIES` | non | Proxies dont `X-Forwarded-For` est cru (rate limiting par IP) ; vide = IP de la socket |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | prod | E-mails de récupération ; sans hôte, le code est écrit dans le journal de l'API (dev) |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API visible du client (**public**, jamais de secret ici) |

## Base de données, migrations et seed

- Migrations SQL versionnées dans `services/api/internal/platform/db/migrations/` (`001` comptes, `002` profils/photos/intérêts, `003` matching/chat/sécurité/notifications, `004` fonction de distance). Elles sont embarquées dans le binaire et **appliquées automatiquement au démarrage**, une fois chacune (table `schema_migrations`, verrou consultatif entre instances, une transaction par fichier).
- Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `photos`, `interests`, `user_interests`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`.
- Contraintes : clés étrangères `ON DELETE CASCADE` (la suppression d'un compte efface tout), `CHECK` (âge ≥ 18, genres, longueurs, `user_a < user_b` pour un match unique), unicités (`email`, `(swiper, target)`, `(user, position)` des photos…).
- Index : `users.email`, `messages(conversation_id, created_at)`, `notifications(user_id, created_at)`, `conversation_participants(user_id, conversation_id)`, géo `(latitude, longitude)` partiel sur les profils découvrables, etc.
- Seed : `go run ./cmd/seed` (ou `./seed` dans le conteneur), `-purge` pour tout retirer.

## Lancer en local (sans Docker)

```bash
docker compose up -d postgres            # ou un PostgreSQL 14+ existant
cd services/api
export DATABASE_URL=postgres://ember:ember@localhost:5432/ember?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
go run ./cmd/api                          # migrations appliquées au démarrage
```

## Tests et qualité

```bash
# API (les tests d'intégration ont besoin d'un PostgreSQL vide dédié)
cd services/api
export DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable
gofmt -l . && go vet ./... && go test ./... && go build ./cmd/api

# Mobile
cd apps/mobile
npm ci && npx tsc --noEmit && npm run lint && npm run format:check && npm test && npm run build:web

# Bout en bout (API démarrée + build web servi)
npm run e2e:api       # parcours API : inscription → profil → découverte → like → match → chat → notifications
npm run e2e:web       # parcours navigateur (Playwright) : inscription → onboarding → découverte → like → match → chat temps réel → suppression
```

La CI (`.github/workflows/ci.yml`) exécute tout cela, plus le build Docker et un contrôle `/health`. Les tests d'intégration sont ignorés (skip explicite) si `DATABASE_URL_TEST` n'est pas défini.

Couverture : règles métier (âge, e-mail, traitement d'images, rate limiter, config), intégration (auth + sessions + récupération, profil, photos, filtres de découverte, like/pass/match, **likes simultanés**, chat + permissions + lu/non lu, blocage, signalement, notifications, suppression de compte, rate limiting, WebSocket), logique mobile (date de naissance, formats, fusion de messages, geste de swipe).

## Build et déploiement

- Image API : `docker build -t ember-api services/api` (multi-étapes, utilisateur non-root, `HEALTHCHECK`, contient `api` et `seed`).
- Production : `APP_ENV=production`, un `JWT_SECRET` aléatoire, `POSTGRES_PASSWORD` fort, SMTP configuré, TLS terminé par un reverse proxy (renseigner `TRUSTED_PROXIES`), volume persistant sur `UPLOAD_DIR`, sauvegardes PostgreSQL.
- Mobile : `eas build` (Expo) avec `EXPO_PUBLIC_API_URL=https://api.votre-domaine`; web : `npm run build:web` produit un site statique dans `dist/`.

## API

JSON, erreurs `{"error": "message"}`, pagination par curseur `?limit=&before=<timestamp>`. Authentification `Authorization: Bearer <accessToken>` (15 min) + refresh token à rotation (30 j).

| Groupe | Endpoints |
|---|---|
| Auth | `POST /auth/register · login · refresh · logout · forgot-password · reset-password`, `PUT /auth/password`, `GET /me`, `DELETE /me` |
| Profil | `GET/PUT /profile`, `PUT /profile/location`, `GET /profiles/:userId`, `GET/PUT /preferences`, `GET /interests` |
| Photos | `GET/POST /photos`, `PUT /photos/order`, `PUT/DELETE /photos/:id`, `GET /photos/:id/file` (authentifié) |
| Découverte | `GET /discover?limit=`, `POST /swipes {userId, action: like\|pass}`, `GET /matches`, `DELETE /matches/:id` |
| Chat | `GET /conversations`, `GET /conversations/unread-count`, `GET/POST /conversations/:id/messages`, `POST /conversations/:id/read`, `DELETE /conversations/:id` |
| Sécurité | `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` |
| Notifications | `GET /notifications`, `POST /notifications/read-all`, `POST /notifications/:id/read` |
| Temps réel | `GET /ws` — première trame `{"type":"auth","token":"…"}` ; événements `message.new`, `match.new`, `notification.new`, `conversation.read` |
| Divers | `GET /health` |

## Décisions techniques

- **Garder la pile du boilerplate** (Go/Gin/pgx, Expo) ; retirer organisations, Stripe, posts, commentaires, fichiers génériques (hors sujet pour un produit gratuit). Schéma repris à zéro, sans données héritées.
- **Âge** : 18 ans minimum à l'inscription (403 sinon), 99 maximum ; la date de naissance n'est jamais exposée, seul l'âge l'est.
- **Localisation** : arrondi à 2 décimales (~1 km) avant stockage ; distance haversine en SQL (pas de PostGIS, donc pas de dépendance d'infrastructure) ; filtre par boîte englobante indexée puis distance exacte ; la distance minimale affichée est 1 km.
- **Mutualité** : orientation, âge et distance doivent convenir **aux deux** personnes pour apparaître en découverte.
- **Likes** : idempotents ; contredire une réponse déjà donnée → 409 ; pas de limite quotidienne. Les `pass` sont définitifs (le profil ne revient pas) — choix le plus simple et le plus sûr contre la lassitude.
- **Algorithme de recommandation** isolé dans `discovery.rankingOrder` (une clause `ORDER BY`) : remplaçable sans toucher à l'API. Actuel : a déjà liké → proximité → activité récente.
- **Photos** : stockage disque derrière l'interface `storage.Storage` (S3 plus tard sans toucher aux features) ; jamais de chemin statique : chaque lecture est autorisée (propriétaire, ou non bloqué + découvrable/apparié). URL versionnée → cache client immuable.
- **Temps réel** : WebSocket, jeton envoyé dans la première trame (pas dans l'URL, donc pas dans les journaux) ; le client se resynchronise par REST à chaque reconnexion. Hub en mémoire : une instance d'API (voir limites).
- **Sessions** : l'access token est vérifié **et** la session est contrôlée en base à chaque requête, donc déconnexion, changement de mot de passe oublié et suppression de compte prennent effet immédiatement.
- **Blocage** : masque tout (découverte, matchs, chats, notifications, profil, photos) dans les deux sens et reste réversible ; les données sont conservées pour permettre la modération d'un signalement.
- **Suppression locale d'une conversation** : un marqueur `hidden_at` par participant ; l'autre personne garde son historique.
- **Notifications** : in-app + temps réel. Pas de push natif (APNs/FCM exigent des comptes développeur propres au déploiement) : c'est le point d'extension prévu (`notifications.Service.Notify`).
- **Récupération de compte** : code à 10 caractères (≈ 50 bits), haché en base, valable 1 h, à usage unique, révoque toutes les sessions ; la réponse est identique que le compte existe ou non.

## Limites connues

- Le hub temps réel et le rate limiter sont en mémoire (par instance). Pour plusieurs instances : LISTEN/NOTIFY ou Redis pour le hub, limite complémentaire au proxy.
- Pas de notifications push natives, pas de tableau de bord de modération (les signalements sont stockés dans `reports`, statut `open`).
- Pas de vérification de l'e-mail à l'inscription ni de modération automatique des photos.
- WebP/HEIC non acceptés côté serveur (le sélecteur mobile fournit du JPEG).
- Les tests UI natifs (iOS/Android) n'ont pas été exécutés dans cet environnement ; le parcours est validé sur le build web et via l'API.
