# Aurore — application de rencontre 100 % gratuite

Aurore est une application de rencontre complète : créer un compte, composer son profil et ses photos, découvrir des personnes proches, liker ou passer, matcher en cas d'intérêt réciproque, discuter en temps réel, recevoir des notifications, bloquer, signaler et supprimer son compte.

**Gratuite pour de vrai** : aucun abonnement, aucun paywall, aucun « boost » payant, aucun like limité artificiellement, aucune publicité. Le code ne contient d'ailleurs aucune intégration de paiement.

> Ce dépôt est issu du boilerplate `go-react-saas`. Les modules génériques (billing Stripe, posts, commentaires, organisations multi-tenant) ont été remplacés par les modules de rencontre.

## Sommaire

1. [Stack](#stack)
2. [Architecture](#architecture)
3. [Démarrage rapide](#démarrage-rapide)
4. [Variables d'environnement](#variables-denvironnement)
5. [Base de données, migrations, seed](#base-de-données-migrations-seed)
6. [Lancer en local (sans Docker)](#lancer-en-local-sans-docker)
7. [Application mobile / web](#application-mobile--web)
8. [Tests et qualité](#tests-et-qualité)
9. [Déploiement](#déploiement)
10. [API](#api)
11. [Sécurité et vie privée](#sécurité-et-vie-privée)
12. [Décisions techniques](#décisions-techniques)
13. [Limites connues](#limites-connues)

## Stack

| Couche | Technologies |
| --- | --- |
| App | Expo 54 / React Native 0.81 / React 19, TypeScript strict, React Navigation 7, TanStack Query 5, fonctionne sur iOS, Android et web |
| API | Go 1.24, Gin, pgx/pgxpool (sans ORM), JWT HS256, bcrypt, WebSocket (gorilla/websocket) |
| Données | PostgreSQL 16, migrations SQL versionnées embarquées dans le binaire |
| Photos | Stockage disque privé (interface `Store` remplaçable), recompression JPEG via `golang.org/x/image` |
| Infra | Docker, Docker Compose, GitHub Actions |

## Architecture

```
services/api/
  cmd/api/            point d'entrée du serveur HTTP (un seul serveur, Gin)
  cmd/seed/           comptes de démonstration (désactivé en production)
  internal/
    app/              assemblage : config → services → routes (+ tests d'intégration)
    platform/         config, db (migrations), middleware (auth JWT, rate limit, CORS…),
                      httpx (erreurs/pagination), storage, mailer, realtime (WebSocket), logger
    features/         auth · profiles · photos · discovery · chat · notifications · moderation
                      chaque feature : handler.go → service.go → repository.go (+ model.go, routes.go)
apps/aurore/          application Expo (iOS / Android / web)
  src/api/            client HTTP (refresh de session single-flight), endpoints typés
  src/auth/           session (SecureStore / localStorage)
  src/realtime/       WebSocket + synchronisation du cache
  src/components/     design system (thème clair/sombre), formulaires partagés
  src/screens/        écrans ; src/navigation/ navigation
  e2e/run.mjs         parcours critique dans un vrai navigateur
infra/docker-compose.yml   PostgreSQL + API (le docker-compose.yml racine l'inclut)
```

Règles : handlers fins, logique métier dans les services, SQL uniquement dans les repositories, aucune requête réseau directe depuis l'UI (tout passe par `src/api`).

### Modèle de données

`users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests` / `user_interests`, `photos`, `swipes` (like/pass), `matches`, `conversations` / `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Contraintes d'unicité (un swipe par paire, un match par paire ordonnée), `CHECK` (âge ≥ 18, genres, bornes), clés étrangères en `ON DELETE CASCADE`, index sur les chemins chauds (voir `internal/platform/db/migrations/`).

## Démarrage rapide

Prérequis : Docker, Node 20+ (22 recommandé).

```bash
cp .env.example .env
# éditez .env : JWT_SECRET=$(openssl rand -base64 48)
docker compose up --build -d          # PostgreSQL + API sur http://localhost:18080
docker compose run --rm api ./seed    # (optionnel) 10 comptes de démonstration

cd apps/aurore
cp .env.example .env                  # EXPO_PUBLIC_API_URL
npm ci
npm run web                           # ou: npm start puis i / a / scan du QR code
```

Comptes de démonstration : `camille@demo.invalid`, `hugo@demo.invalid`, … mot de passe `DemoPassword123` (domaine réservé `.invalid`, jamais routable ; marqués `is_demo` en base).

**Sur un téléphone physique**, n'utilisez jamais `localhost` : mettez l'IP LAN de votre machine dans `apps/aurore/.env` (`EXPO_PUBLIC_API_URL=http://192.168.1.20:18080`). Vérifiez qu'aucun autre service n'occupe le port (`docker compose ps`, `curl http://<IP>:18080/health`). Si le port 18080 est pris, changez le port hôte dans `infra/docker-compose.yml` et la variable de l'app.

## Variables d'environnement

Voir [`.env.example`](.env.example) (API / compose) et [`apps/aurore/.env.example`](apps/aurore/.env.example) (app).

| Variable | Requis | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | ≥ 32 caractères, aléatoire. Les valeurs de type « change-me » sont refusées. |
| `DATABASE_URL` | oui | URL PostgreSQL (compose la construit lui-même). |
| `PORT` | non | Défaut `8080`. |
| `APP_ENV` | non | `development` (défaut) ou `production`. En production, `SMTP_HOST` est obligatoire et le seed refuse de s'exécuter. |
| `UPLOAD_DIR` | non | Dossier privé des photos (défaut `./data/uploads`, volume Docker en compose). |
| `ALLOWED_ORIGINS` | non | Origines web autorisées par CORS et par le WebSocket (inutile pour iOS/Android natifs). |
| `TRUSTED_PROXIES` | non | IP/CIDR des reverse proxies dont `X-Forwarded-For` est cru (sinon ignoré : le rate limiting voit l'IP réelle). |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | prod | Envoi des codes de récupération. Sans `SMTP_HOST`, le code est écrit dans les logs (développement uniquement). |
| `EXPO_PUBLIC_API_URL` | app | URL de l'API **joignable depuis l'appareil**. Aucun secret ne doit jamais être mis dans une variable `EXPO_PUBLIC_*`. |

## Base de données, migrations, seed

- Les migrations sont dans `services/api/internal/platform/db/migrations/*.sql`, embarquées dans le binaire et **appliquées au démarrage**, une fois chacune, dans une transaction, avec un verrou consultatif (plusieurs réplicas peuvent démarrer ensemble). L'état est suivi dans `schema_migrations`.
- Pour ajouter une évolution : créer `005_xxx.sql` (ne jamais modifier une migration déjà publiée).
- Seed : `go run ./cmd/seed` (ou `docker compose run --rm api ./seed`) crée 10 profils fictifs avec des photos générées (dégradés). `go run ./cmd/seed -purge` les supprime tous (comptes `is_demo`, fichiers inclus). Refuse de tourner avec `APP_ENV=production`.
- ⚠️ Si vous aviez une base issue de l'ancien boilerplate, repartez d'une base vide : le schéma a été entièrement remplacé.

## Lancer en local (sans Docker)

```bash
# PostgreSQL local (ex. base « app »)
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/app?sslmode=disable"
export JWT_SECRET="$(openssl rand -base64 48)"
export ALLOWED_ORIGINS=http://localhost:8081   # si vous utilisez l'app web
cd services/api
go run ./cmd/seed        # optionnel
go run ./cmd/api         # http://localhost:8080
```

## Application mobile / web

```bash
cd apps/aurore
npm ci
npm start            # Expo : touche i (iOS), a (Android), w (web)
npm run build:web    # export statique dans dist/
```

Fonctionnalités : onboarding en 5 étapes (identité 18+, préférences, photos, bio/centres d'intérêt, position), découverte avec **swipe** *et* boutons (accessibles), profil complet consultable, match avec écran dédié, messagerie temps réel (heure, lu/non lu, suppression locale), notifications avec badges, réglages de confidentialité (pause du profil, masquage de la distance), blocage/signalement, suppression de compte, thème clair/sombre, états de chargement/vides/erreurs, retour d'action par toasts.

## Tests et qualité

Avant toute livraison :

```bash
cd services/api
gofmt -l .                         # doit être vide
go vet ./...
DATABASE_URL_TEST="postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable" go test ./...
go build ./cmd/api ./cmd/seed

cd ../../apps/aurore
npx tsc --noEmit && npm run lint && npm test
```

> ⚠️ Les tests d'intégration **suppriment et recréent le schéma `public`** de `DATABASE_URL_TEST` (ce qui prouve que les migrations s'appliquent depuis zéro). Sans cette variable, ils sont ignorés. Ne la pointez jamais vers une base qui vous importe.

Couverture :
- **Unitaires** (Go) : âge à la date près, 18+, arrondi des coordonnées, paliers de distance, validation des préférences, pipeline photo (types, bombes de décompression, métadonnées), anti-traversée de chemin du stockage, rate limiter, config. (App) : dates, validation, client API (refresh unique partagé, déconnexion, réseau coupé), cache temps réel, composants et écrans d'auth.
- **Intégration** (Go, API réelle + PostgreSQL) : auth (bcrypt, rotation du refresh, récupération, JWT `none`/HS512/expiré/sans `exp`), profils, photos (upload malveillant, limites, ordre, remplacement, accès), éligibilité de la découverte, swipes/matchs (doublons, **courses concurrentes**), chat et permissions (404 pour les non-membres), blocage, signalements, notifications, suppression de compte (données + fichiers), WebSocket, rate limiting.
- **E2E** (vrai navigateur + vraie API) : `apps/aurore/e2e/run.mjs` — inscription → onboarding → découverte → like → match → conversation temps réel → notifications → suppression de compte.

```bash
# API démarrée avec le seed et ALLOWED_ORIGINS=http://localhost:8081
cd apps/aurore
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
npm run e2e          # captures dans e2e/artifacts/
```

La CI (`.github/workflows/`) exécute tout cela sur chaque push (Go + PostgreSQL, app, build Docker, E2E).

## Déploiement

1. Construisez l'image : `docker build -t aurore-api services/api` (binaire statique, utilisateur non-root, `HEALTHCHECK`).
2. Fournissez PostgreSQL 16+ (managé de préférence), `JWT_SECRET` fort, `APP_ENV=production`, **SMTP** (obligatoire), `UPLOAD_DIR` sur un **volume persistant et sauvegardé**.
3. Placez l'API derrière un reverse proxy TLS (HSTS est émis quand `X-Forwarded-Proto: https`) et renseignez `TRUSTED_PROXIES`. Le WebSocket (`/ws`) doit être relayé (`Upgrade`).
4. Web : servez `apps/aurore/dist/` (export statique, `EXPO_PUBLIC_API_URL` fixé au build) et ajoutez son origine à `ALLOWED_ORIGINS`.
5. Mobile : `eas build` (ou build natif) avec `EXPO_PUBLIC_API_URL` pointant sur l'API HTTPS.

Scalabilité : le hub temps réel, le rate limiter et les tickets WebSocket sont **en mémoire** (une instance). Pour plusieurs instances, ajoutez un pub/sub (Redis) derrière `realtime.Hub.Publish`, un limiter partagé (ou au niveau du proxy) et un stockage objet derrière `storage.Store`.

## API

Réponses d'erreur uniformes : `{"error": "message lisible", "code": "slug"}`. Toutes les routes (sauf `/health`, `/auth/*`) exigent `Authorization: Bearer <accessToken>` (15 min ; refresh rotatif 30 j).

| Domaine | Routes |
| --- | --- |
| Santé | `GET /health` |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/forgot`, `/auth/reset` · `GET /me` · `DELETE /me` (mot de passe requis) |
| Profil | `GET\|PUT /me/profile`, `PUT /me/preferences`, `PUT /me/location`, `PUT /me/interests`, `GET /interests` |
| Photos | `GET\|POST /me/photos`, `PUT /me/photos/order`, `PUT\|DELETE /me/photos/:id`, `GET /photos/:id/image\|thumb` (authentifié) |
| Découverte | `GET /discover?limit=`, `POST /discover/swipes {userId, action}`, `DELETE /discover/swipes/:userId` (annuler), `GET /profiles/:userId` |
| Matchs | `GET /matches`, `DELETE /matches/:id` |
| Chat | `GET /conversations`, `GET\|POST /conversations/:id/messages` (`?before=` curseur), `POST /conversations/:id/read`, `DELETE /conversations/:id` (suppression locale) |
| Notifications | `GET /notifications`, `GET /notifications/unread-count`, `POST /notifications/:id/read`, `POST /notifications/read-all` |
| Modération | `GET\|POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` |
| Temps réel | `POST /realtime/ticket` puis `GET /ws?ticket=` → événements `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new` |

## Sécurité et vie privée

- **Auth** : bcrypt, JWT HS256 uniquement (méthode et `exp` vérifiés), refresh rotatif à usage unique stocké haché, révocation globale après réinitialisation du mot de passe, comparaison à temps constant, faux hachage pour les emails inconnus, `/auth/forgot` répond toujours 202 (envoi en arrière-plan : pas d'oracle de timing).
- **Autorisations côté serveur partout** : un non-membre d'une conversation reçoit `404` (rien ne fuite) ; un swipe n'est accepté que si la cible figurait légitimement dans la découverte (même requête SQL d'éligibilité) ; blocage bidirectionnel appliqué à la découverte, aux profils, aux photos, aux notifications et au chat.
- **Géolocalisation** : seule une position **arrondie à ~1 km** est stockée ; les autres voient une distance **arrondie par paliers de 5 km** (anti-triangulation) ; jamais de coordonnées dans l'API publique.
- **Âge** : 18 ans minimum contrôlé à la date près côté serveur (et par `CHECK` en base) ; seul l'âge est exposé, jamais la date de naissance.
- **Photos** : type vérifié par contenu (JPEG/PNG/WebP), taille ≤ 8 Mo, dimensions bornées (anti « bombe de décompression »), ré-encodage JPEG (EXIF/GPS et charges cachées supprimés), noms aléatoires, dossier non servi statiquement, accès par l'API uniquement après contrôle d'autorisation, `Cache-Control: private`.
- **Abus** : rate limiting par IP (auth, récupération) et par utilisateur (messages, uploads, signalements, swipes), plafonds de corps de requête, 5 essais max par code de récupération (valable 30 min).
- **Données** : suppression de compte (mot de passe requis) = cascade SQL complète + fichiers + coupure des WebSockets ; les signalements sont conservés sans lien vers les comptes supprimés (historique de modération).
- Les secrets ne sont jamais dans le dépôt ni dans l'app ; les logs masquent jetons et mots de passe.

## Décisions techniques

- **Garder** Gin / pgx / JWT / sessions / logger / structure `handler → service → repository` ; **remplacer** le schéma et les features génériques ; Tamagui retiré (non utilisé).
- **Migrations versionnées** (table `schema_migrations`) plutôt que ré-exécuter tous les fichiers à chaque démarrage.
- **Un swipe par paire** (`swipes`, action `like|pass`) ; match créé dans la même transaction que le like réciproque, sous verrou consultatif de la paire → jamais de doublon ni de like « raté » en cas de simultanéité (testé). Un pass peut être annulé (retour de la carte) ; un match se supprime via « supprimer le match ».
- **Recommandation** isolée dans un seul `ORDER BY` (`discovery/repository.go`) : d'abord ceux qui m'ont liké, puis intérêts communs, proximité, activité récente. L'éligibilité (`candidateSQL`) est séparée du classement pour pouvoir brancher plus tard un autre algorithme sans toucher aux règles.
- **Profil « complet »** = profil + préférences + au moins une photo ; la position est facultative (sans position, pas de filtre ni d'affichage de distance).
- **Temps réel par WebSocket** authentifié par ticket à usage unique (le jeton d'accès ne transite jamais dans une URL) ; les écritures passent par REST ; reconnexion avec backoff et resynchronisation des caches.
- **Photos protégées** : sur le web, `<img>` ne peut pas envoyer d'en-tête d'auth → les images sont récupérées avec le jeton puis affichées via une URL objet mise en cache.
- **Pas de push natif** (APNs/FCM) : notifications in-app + temps réel tant que l'app est ouverte (voir limites).
- **Récupération de compte** par code à 8 caractères envoyé par email (SMTP standard, sans dépendance) plutôt que par lien profond.
- **Modération** : signalements stockés (`reports`, statut `open`) ; la revue se fait en SQL ou via un futur outil d'administration.

## Limites connues

- Pas de notifications push hors application (nécessite des comptes Apple/Google).
- Pas d'interface d'administration pour traiter les signalements (table `reports`).
- Hub temps réel / rate limiter / tickets en mémoire (une instance) ; stockage photos sur disque local.
- L'orientation EXIF des photos est gérée par la recompression du sélecteur d'images natif (`quality < 1`).
- Les tests Docker/Compose ne s'exécutent qu'en CI (pas de démon Docker dans l'environnement de développement de cette livraison).
