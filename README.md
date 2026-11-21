# Lueur

**Lueur** est une application de rencontre moderne et **100 % gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de limite artificielle de likes, pas de publicité. Chaque membre peut créer son profil, ajouter ses photos, découvrir des personnes compatibles près de chez lui, liker ou passer, matcher quand l'intérêt est réciproque, discuter en temps réel, et garder la maîtrise de sa vie privée (position approximative, pause du profil, blocage, signalement, suppression de compte).

Le dépôt est construit sur le boilerplate `go-react-saas` (API Go + client Expo) dont il conserve l'architecture et les modules génériques.

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Installation et lancement local](#installation-et-lancement-local)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données et migrations](#base-de-données-et-migrations)
7. [Données de démonstration](#données-de-démonstration)
8. [Tests et qualité](#tests-et-qualité)
9. [Build et déploiement](#build-et-déploiement)
10. [API](#api)
11. [Principales décisions techniques](#principales-décisions-techniques)
12. [Limites connues et pistes](#limites-connues-et-pistes)

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation des refresh tokens, déconnexion immédiate, récupération du mot de passe par code à 6 chiffres envoyé par e-mail, suppression définitive du compte |
| Onboarding | Prénom, date de naissance (18 ans minimum, vérifié côté serveur), genre, qui rencontrer, 1 à 6 photos, bio, ce que je recherche, centres d'intérêt, position approximative (facultative) |
| Profil | Édition complète, gestion des photos (ajout, remplacement, suppression, photo principale), préférences (genres, tranche d'âge, distance), aperçu de son profil |
| Découverte | Pile de cartes avec swipe (ou boutons accessibles), plusieurs photos par carte, distance approximative, centres d'intérêt, badge « Vous a liké », profil complet |
| Matching | Like / passer, match uniquement si réciproque, aucun doublon, aucun profil déjà traité ne réapparaît, résistant aux likes simultanés |
| Chat | Conversation ouverte après match, temps réel (WebSocket), heure d'envoi, état lu / envoyé, compteurs de non-lus, pagination, effacement local, annuler le match |
| Notifications | Nouveaux matchs et messages (regroupés par conversation), en temps réel dans l'app, badges, « tout marquer comme lu » |
| Confidentialité | Coordonnées arrondies (~1 km) avant stockage, distance affichée arrondie (jamais < 1 km), option « masquer ma distance », pause du profil, métadonnées EXIF (dont GPS) supprimées des photos |
| Sécurité / modération | Blocage (invisible pour la personne bloquée), signalement (avec blocage automatique et conservation d'un extrait de la conversation pour la modération), suspension automatique après 3 signalements distincts, rate limiting |
| UX | Identité visuelle originale (crème, encre, terracotta), mode sombre automatique ou forcé, états de chargement / vide / erreur, toasts, accessibilité (labels, rôles, actions alternatives au swipe) |

---

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16 via `pgxpool` (un seul pool partagé), SQL explicite (pas d'ORM), JWT HS256 (15 min) + refresh tokens opaques hachés en base, bcrypt, WebSocket (`gorilla/websocket`), traitement d'images (`golang.org/x/image`).
- **Client** : React Native 0.81 / Expo SDK 54, TypeScript strict, React Navigation (stack natif + onglets), TanStack Query, `expo-image`, `expo-image-picker`, `expo-location`, `expo-secure-store`. Le même code tourne sur iOS, Android et le Web (React Native Web).
- **Infra** : Docker (image Alpine non-root), Docker Compose, GitHub Actions (tests Go avec Postgres, typecheck/lint/format/tests front, E2E API + navigateur).

---

## Architecture

```text
.
├─ apps/mobile/                 # Client Expo (iOS, Android, Web)
│  ├─ App.tsx                   # Providers : thème, React Query, auth, temps réel
│  ├─ src/api/                  # Client HTTP (refresh automatique), endpoints typés, clés de cache
│  ├─ src/features/             # Écrans par domaine : auth, onboarding, discovery, matches,
│  │                            #   chat, notifications, profile, safety
│  ├─ src/navigation/           # Stack racine (auth / onboarding / app) + onglets
│  ├─ src/realtime/             # WebSocket → cache React Query
│  ├─ src/theme/ + src/ui/      # Design system (tokens clair/sombre, composants accessibles)
│  ├─ scripts/e2e-smoke.js      # Smoke test API du parcours critique
│  └─ e2e/web-flow.js           # E2E navigateur (Playwright) du parcours critique
├─ services/api/
│  ├─ cmd/api/                  # Point d'entrée (composition, routes, serveur HTTP)
│  ├─ cmd/seed/                 # Données de démonstration (jamais en production)
│  └─ internal/
│     ├─ platform/              # Transverse : config, db (migrations), middleware (auth, CORS,
│     │                         #   rate limit, en-têtes), httpx, storage, media (URLs signées),
│     │                         #   realtime (hub WebSocket), mailer, geo, logger
│     └─ features/              # handler → service → repository par domaine :
│        ├─ auth  profiles  photos  discovery  matching  chat  safety  notifications
│        └─ users posts comments files   # modules génériques (désactivés par défaut)
├─ docker-compose.yml
├─ .env.example
└─ docs/ARCHITECTURE.md         # Détails : modèle de données, flux, sécurité
```

Chaque feature backend suit `handler.go` (HTTP fin) → `service.go` (règles métier) → `repository.go` (SQL) avec `model.go` et `routes.go`. Le client ne fait jamais d'appel réseau depuis l'UI : tout passe par `src/api`.

Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) pour le modèle de données, l'algorithme de découverte, le temps réel et les choix de sécurité.

---

## Installation et lancement local

### Prérequis

- Docker + Docker Compose **ou** Go 1.24+ et PostgreSQL 16
- Node.js 22+ et npm
- Pour le mobile : l'app **Expo Go** sur un téléphone, ou un simulateur iOS / émulateur Android

### 1. Configurer

```bash
git clone <repo> && cd <repo>
cp .env.example .env
# Générez un secret fort et collez-le dans JWT_SECRET :
openssl rand -base64 48
```

### 2. Lancer l'API et la base (Docker)

```bash
docker compose up --build -d
curl http://localhost:18080/health      # {"status":"ok"}
```

L'API écoute sur `http://localhost:18080`, PostgreSQL sur `127.0.0.1:5432`. Les migrations s'appliquent automatiquement au démarrage. Les photos sont stockées dans le volume `uploads`.

> En `APP_ENV=development` sans SMTP configuré, les codes de récupération de mot de passe sont **affichés dans les logs de l'API** (`docker compose logs api`). Ce mode n'existe pas en production.

### 2 bis. Lancer l'API sans Docker

```bash
cd services/api
export APP_ENV=development
export DATABASE_URL='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export ALLOWED_ORIGINS=http://localhost:8081
export PORT=18080
go run ./cmd/api
```

### 3. Ajouter des profils de démonstration (optionnel)

```bash
docker compose exec api ./seed -count 18            # ou : cd services/api && go run ./cmd/seed
```

### 4. Lancer l'application

```bash
cd apps/mobile
cp .env.example .env        # EXPO_PUBLIC_API_URL=http://localhost:18080
npm ci
npm run web                 # navigateur : http://localhost:8081
npm run start               # QR code pour Expo Go / simulateurs
```

**Sur un téléphone physique**, `localhost` désigne le téléphone lui-même : utilisez l'IP locale de votre ordinateur.

```bash
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npm run start
# ou, pour détecter l'IP automatiquement :
npm run start:phone
```

Vérifiez avant de tester que rien d'autre n'occupe le port de l'API (`docker compose ps`, puis `curl http://<IP_LAN>:18080/health`). En cas de conflit, changez le port hôte dans `docker-compose.yml` (ex. `18081:8080`) et `EXPO_PUBLIC_API_URL`.

---

## Variables d'environnement

### API (`.env` à la racine, lu par Docker Compose)

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | oui | — | Secret HS256, ≥ 32 caractères, valeur aléatoire (les valeurs « placeholder » sont refusées) |
| `DATABASE_URL` | oui | (Compose la construit) | Connexion PostgreSQL |
| `APP_ENV` | non | `production` | `development`, `test` ou `production` |
| `PORT` | non | `8080` | Port HTTP du processus |
| `UPLOAD_DIR` | non | `./data/uploads` | Dossier de stockage des photos traitées |
| `MEDIA_SIGNING_SECRET` | non | dérivé de `JWT_SECRET` | Clé de signature des URLs de photos (≥ 32 caractères) |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS), ex. `http://localhost:8081` |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies autorisés à fournir `X-Forwarded-For` |
| `ENABLE_COMMUNITY_MODULES` | non | `false` | Active les modules génériques (annuaire, posts, commentaires, fichiers) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | port `587` | Envoi des e-mails de récupération (STARTTLS) |
| `POSTGRES_PASSWORD` | non | `postgres` | Mot de passe du conteneur Postgres local |

### Client (`apps/mobile/.env`)

| Variable | Rôle |
| --- | --- |
| `EXPO_PUBLIC_API_URL` | URL publique de l'API. **Aucun secret** côté client : les variables `EXPO_PUBLIC_*` sont embarquées dans l'application. |

Les fichiers `.env` sont ignorés par Git ; seuls les `.env.example` sont versionnés.

---

## Base de données et migrations

Les migrations SQL sont dans `services/api/internal/platform/db/migrations/` et embarquées dans le binaire. Au démarrage, l'API :

1. prend un verrou consultatif PostgreSQL (plusieurs instances peuvent démarrer en même temps) ;
2. applique dans l'ordre chaque fichier non encore présent dans `schema_migrations`, chacun dans sa transaction.

Tables principales : `users`, `sessions`, `password_reset_codes`, `profiles`, `preferences`, `photos`, `interests`, `user_interests`, `swipes` (likes et passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Contraintes (`CHECK`, unicité, clés étrangères avec `ON DELETE CASCADE`), index et fonction `haversine_km` sont décrits dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

Pour ajouter une migration : créez `NNN_description.sql` (numéro suivant), écrite de façon idempotente (`IF NOT EXISTS`) ; elle sera appliquée au prochain démarrage.

---

## Données de démonstration

```bash
cd services/api
go run ./cmd/seed -count 18                        # crée des membres de démo autour de Paris
go run ./cmd/seed -lat 45.76 -lng 4.83             # … autour de Lyon
go run ./cmd/seed -like-email vous@exemple.fr      # ils vous likent tous : match immédiat si vous likez
go run ./cmd/seed -purge                           # supprime toutes les données de démo
```

Garanties de séparation : le seed **refuse de s'exécuter si `APP_ENV=production`**, les comptes utilisent le domaine réservé `demo.lueur.invalid` (impossible à confondre avec un vrai e-mail), les profils portent `is_demo = true` et un badge « DÉMO » est affiché dans l'application. Mot de passe des comptes de démo : `DemoPassword1` (ex. `demo01@demo.lueur.invalid`).

---

## Tests et qualité

### API

```bash
cd services/api
gofmt -l .                      # doit être vide
go vet ./...
DATABASE_URL_TEST='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable' go test -race ./...
```

- **Tests unitaires** : validation de profil et âge, préférences, classement de la découverte, normalisation des messages, signalements, politique de mot de passe, géolocalisation, URLs signées, rate limiter, JWT, stockage (anti path traversal), pipeline d'images (types dangereux, images corrompues, orientation EXIF), hub WebSocket.
- **Tests d'intégration** (Postgres réel, via le routeur HTTP) : parcours complet inscription → profil → photo → découverte → like → match → chat → lu → blocage → signalement → suppression de compte ; likes simultanés (un seul match) ; permissions du chat ; gestion des photos ; récupération de mot de passe (anti-brute-force, révocation des sessions) ; temps réel ; rate limiting (y compris `X-Forwarded-For` falsifié) ; confidentialité (coordonnées arrondies, distance masquée, pause).

Sans base disponible, les tests d'intégration sont ignorés (`t.Skip`) et les tests unitaires s'exécutent.

### Client

```bash
cd apps/mobile
npm run typecheck      # TypeScript strict
npm run lint           # ESLint (config Expo), `any` interdit
npm run format:check   # Prettier
npm test               # tests unitaires (runner natif Node 22)
```

### Bout en bout

Avec une API lancée (`APP_ENV=development`, `ALLOWED_ORIGINS=http://localhost:8081`) :

```bash
cd apps/mobile
npm run e2e:smoke                                  # parcours critique via l'API
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
npm run e2e:web                                    # parcours critique dans Chromium (Playwright)
```

Le test navigateur couvre : inscription → onboarding avec upload de photo → découverte → like → écran de match → conversation temps réel (envoi, réception, « Lu ») → liste des matchs → notifications → suppression du compte. Les captures d'écran sont écrites dans `apps/mobile/e2e/.artifacts/`. Playwright doit être installé (`npm i -g playwright && npx playwright install chromium`, puis `NODE_PATH=$(npm root -g)`).

La CI GitHub Actions (`.github/workflows/ci.yml`) exécute l'ensemble : tests Go avec Postgres, contrôles front, puis smoke API + E2E navigateur et build de l'image Docker.

---

## Build et déploiement

### API

```bash
docker build -t lueur-api ./services/api       # image Alpine, utilisateur non-root, binaires api + seed
```

En production :

- `APP_ENV=production`, `JWT_SECRET` (et idéalement `MEDIA_SIGNING_SECRET`) générés aléatoirement et stockés dans un gestionnaire de secrets ;
- PostgreSQL managé avec `sslmode=require` dans `DATABASE_URL` ;
- un volume persistant (ou un stockage objet, voir pistes) monté sur `UPLOAD_DIR` ;
- un reverse proxy TLS (HTTPS + WSS) devant l'API, déclaré dans `TRUSTED_PROXIES` ; activer la mise à niveau WebSocket sur `/realtime` ;
- `ALLOWED_ORIGINS` limité au domaine du client web ;
- SMTP configuré pour la récupération de compte.

Le hub temps réel est en mémoire : déployez **une instance** de l'API, ou ajoutez un bus partagé (voir pistes) avant de passer à plusieurs instances.

### Client

- **Web** : `EXPO_PUBLIC_API_URL=https://api.example.com npm run build:web` produit `apps/mobile/dist/`, à servir par n'importe quel hébergement statique (avec redirection SPA vers `index.html`).
- **iOS / Android** : builds via [EAS Build](https://docs.expo.dev/build/introduction/) (`npx eas build`), en définissant `EXPO_PUBLIC_API_URL` dans le profil de build. Les permissions sont déclarées dans `app.json` (localisation approximative uniquement sur Android, accès photos/caméra avec textes explicatifs).

---

## API

Toutes les routes (hors auth publique, santé et médias signés) exigent `Authorization: Bearer <accessToken>`. Erreurs au format `{"error": "...", "code": "..."}`.

| Méthode | Route | Rôle |
| --- | --- | --- |
| GET | `/health` | Santé (DB incluse) |
| POST | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Session |
| POST | `/auth/password/forgot`, `/auth/password/reset` | Récupération de compte |
| GET / DELETE | `/me` | Compte / suppression (mot de passe requis) |
| GET / PUT | `/me/profile` | Profil complet du membre (+ préférences, photos, intérêts, complétude) |
| PUT | `/me/preferences`, `/me/interests` | Préférences, centres d'intérêt |
| PUT / DELETE | `/me/location` | Position approximative |
| PATCH | `/me/settings` | `isVisible`, `showDistance` |
| GET | `/interests` | Catalogue des centres d'intérêt |
| GET / POST | `/me/photos` | Lister / ajouter (multipart `photo`) |
| PUT | `/me/photos/order`, `/me/photos/:id` | Réordonner / remplacer |
| DELETE | `/me/photos/:id` | Supprimer |
| GET | `/media/photos/:id?v=&exp=&sig=` | Photo via URL signée expirante |
| GET | `/discovery?limit=&exclude=` | Profils à découvrir |
| GET | `/profiles/:userId` | Profil public (si autorisé) |
| POST | `/swipes` | `{targetUserId, action: like|pass}` → `{matched, match}` |
| DELETE | `/matches/:userId` | Annuler un match |
| GET | `/chats?limit=&offset=` | Conversations (matchs) avec dernier message et non-lus |
| GET / POST / DELETE | `/chats/:userId/messages` | Messages (`?before=<id>`), envoi, effacement local |
| POST | `/chats/:userId/read` | Accusé de lecture |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId` | Blocages |
| POST | `/reports` | Signalement (bloque aussi) |
| GET / POST | `/notifications`, `/notifications/:id/read`, `/notifications/read-all` | Notifications |
| WS | `/realtime` | Événements : `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new` |

---

## Principales décisions techniques

- **Construire sur l'existant** : l'architecture Go feature-first et le client Expo étaient sains ; ils ont été étendus plutôt que remplacés. Le module de facturation Stripe a été **supprimé** (produit gratuit). Les modules génériques (annuaire, posts, commentaires, fichiers) sont conservés mais **désactivés par défaut** ; l'annuaire n'expose plus jamais d'e-mail.
- **Matching sûr en concurrence** : verrou consultatif PostgreSQL par paire + clé primaire `(actor, target)` sur `swipes` + clé unique `(low, high)` sur `matches`. Deux likes simultanés produisent exactement un match (testé).
- **Découverte** : tous les filtres stricts (préférences mutuelles de genre et d'âge, distance dans les deux rayons, profils déjà traités, blocages dans les deux sens, profils en pause ou suspendus) sont en SQL ; le classement est une interface Go (`Ranker`) avec un score explicable (a liké, intérêts communs, activité récente, proximité, complétude, nouveauté), remplaçable sans toucher à l'API.
- **Vie privée de la localisation** : coordonnées arrondies à 0,01° (~1 km) côté client et côté serveur, jamais renvoyées ; seules des distances entières ≥ 1 km sont exposées, et le membre peut les masquer.
- **Photos** : validation par le contenu (pas par l'extension ni le type déclaré), limites de taille et de pixels (anti « decompression bomb »), orientation EXIF appliquée puis **ré-encodage JPEG** (supprime toutes les métadonnées, dont le GPS), miniatures, noms de fichiers aléatoires, URLs **signées HMAC et expirantes** (pas de listing public).
- **Sessions** : jeton d'accès court (15 min) + refresh token rotatif haché ; chaque requête vérifie que la session existe toujours, donc déconnexion, réinitialisation du mot de passe et suppression de compte prennent effet immédiatement.
- **Chat** : une conversation n'existe qu'à travers un match actif ; le même code 404 est renvoyé pour « jamais matché », « bloqué » ou « compte supprimé » afin de ne rien divulguer.
- **Blocage silencieux** : la personne bloquée voit seulement le match disparaître. Un signalement bloque automatiquement et conserve les 30 derniers messages pour les modérateurs ; 3 signalements distincts suspendent le profil de la découverte en attendant une revue.
- **Rate limiting** en mémoire sur l'inscription, la connexion, la récupération (par IP, IPv6 regroupé par /64, et par compte), la suppression de compte, l'envoi de messages, les uploads, les signalements et les swipes (plafond anti-robot de 1 000/heure, pas une limite produit). Hors production, les seuils sont multipliés par 20 pour ne pas gêner le développement et les tests E2E. Le refresh de session n'est pas limité par IP (jetons non devinables, NAT d'opérateur).
- **Sessions révoquées = sockets fermées** : déconnexion, réinitialisation du mot de passe et suppression de compte ferment aussi les WebSockets ouverts ; le refresh token est échangé par compare-and-swap.
- **Concurrence modération / matching** : blocages et signalements prennent le même verrou par paire que les likes ; un membre suspendu ne peut plus découvrir, liker ni écrire ; seuls les signalements de comptes complets comptent pour la suspension automatique.
- **Temps réel** : WebSocket authentifié par le premier message (le jeton n'apparaît jamais dans les URLs ni les logs), fermeture à l'expiration du jeton, reconnexion avec backoff et resynchronisation côté client.
- **Web inclus** : React Native Web permet un vrai test E2E navigateur du parcours critique et une version web de l'app.

---

## Limites connues et pistes

- **Notifications push** (téléphone verrouillé) : non incluses, car elles nécessitent un projet EAS et des identifiants Apple/Google propres au compte de l'éditeur. Les notifications sont persistées et poussées en temps réel dans l'app ; l'ajout de `expo-notifications` + un envoi via l'API Expo Push lorsque le destinataire n'a pas de connexion WebSocket active est la suite logique.
- **Mise à l'échelle horizontale** : le hub WebSocket et le rate limiter sont en mémoire. Pour plusieurs instances, brancher `realtime.Publisher` sur PostgreSQL `LISTEN/NOTIFY` ou Redis, et le limiter sur Redis.
- **Stockage des photos** : disque local par défaut ; l'interface `storage.Storage` permet d'ajouter un backend S3 compatible.
- **Modération** : les signalements sont stockés (`reports`) avec leur statut ; il n'y a pas encore d'interface d'administration (revue via SQL).
- **Dépendances front** : `npm audit` signale des vulnérabilités dans l'outillage Expo / Metro (serveur de développement, non embarqué dans l'app) et dans React Navigation 6 ; les corriger demande de passer à Expo SDK 55+ et React Navigation 7 (`npm audit fix` casse l'arbre de dépendances Expo 54, ne pas l'utiliser).
- **Vérification d'âge** : déclarative (date de naissance, 18+ contrôlé côté serveur). Une vérification d'identité nécessiterait un prestataire tiers.
