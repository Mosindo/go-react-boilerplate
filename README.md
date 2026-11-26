# Lueur : application de rencontre 100 % gratuite

Lueur est une application de rencontre complète (mobile iOS/Android et web) avec une API Go.
Elle permet de créer un compte, de construire son profil avec photos, de découvrir des profils proches,
de liker ou passer, d'obtenir un match quand l'intérêt est réciproque, puis de discuter en temps réel.

**Tout est gratuit** : pas d'abonnement, pas de paywall, pas de « boost », likes illimités,
« qui m'a liké » visible sans flou, pas de publicité.

| Découverte | Match | Conversation | Mode sombre |
| --- | --- | --- | --- |
| <img src="docs/screenshots/discover.png" width="190" alt="Découverte"> | <img src="docs/screenshots/match.png" width="190" alt="Match"> | <img src="docs/screenshots/chat.png" width="190" alt="Conversation"> | <img src="docs/screenshots/profile-dark.png" width="190" alt="Profil en mode sombre"> |

<sub>Captures issues du test E2E sur le build web, avec des profils de démonstration aux photos abstraites.</sub>

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Installation et lancement local](#installation-et-lancement-local)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données, migrations et seed](#base-de-données-migrations-et-seed)
7. [Tests et qualité](#tests-et-qualité)
8. [Build et déploiement](#build-et-déploiement)
9. [API](#api)
10. [Décisions techniques et produit](#décisions-techniques-et-produit)
11. [Limites connues et pistes](#limites-connues-et-pistes)

## Fonctionnalités

- **Compte** : inscription (18 ans et plus, acceptation des conditions), connexion, sessions avec rotation
  des refresh tokens, récupération du mot de passe par code envoyé par e-mail, changement de mot de passe,
  déconnexion, **suppression définitive du compte** (profil, photos, matchs, messages, notifications).
- **Onboarding en 5 étapes** : prénom et date de naissance, genre et genres recherchés, photos,
  bio/métier/ville/centres d'intérêt, localisation approximative (facultative).
- **Profil** : jusqu'à 6 photos (ajout, photo principale, réordonnancement, remplacement, suppression),
  bio, métier, ville, 10 centres d'intérêt parmi 40, préférences (genres, tranche d'âge, distance).
- **Découverte** : cartes à glisser (ou boutons accessibles), plusieurs photos par profil, profil complet,
  filtres **mutuels** (genre, âge, distance), classement par affinités, aucun profil déjà traité ne revient.
- **Matching** : like + like = match, sans doublon même en cas de likes simultanés ; annulation de match.
- **Likes reçus** : liste des personnes qui vous ont liké, pour liker en retour.
- **Messagerie** : conversations ouvertes uniquement après un match, temps réel (WebSocket),
  heure d'envoi, séparateurs de jour, état envoyé/lu, compteurs de non-lus, pagination.
- **Notifications** : match, like reçu, nouveau message (regroupés par conversation), en temps réel,
  marquage lu / tout lire.
- **Confidentialité** : mise en pause du profil, masquage de la distance, localisation approximative
  modifiable/supprimable, métadonnées des photos (dont GPS) supprimées.
- **Sécurité et modération** : blocage (supprime le match et la conversation), signalement motivé
  (bloque aussi), mise en revue automatique après 3 signalements distincts, rate limiting anti-abus.

## Stack technique

| Couche | Choix |
| --- | --- |
| API | Go 1.23+, Gin, PostgreSQL 16, `pgxpool`, SQL explicite (sans ORM), JWT HS256, bcrypt |
| Temps réel | WebSocket (`gorilla/websocket`) avec tickets à usage unique |
| Images | Décodage/redimensionnement/ré-encodage côté serveur (`golang.org/x/image`) |
| App | Expo SDK 54, React Native 0.81, React 19, TypeScript strict |
| Navigation / données | React Navigation 6 (tabs + native stack), TanStack Query 5 |
| Médias / capteurs | `expo-image`, `expo-image-picker`, `expo-image-manipulator`, `expo-location`, `expo-secure-store` |
| Qualité | `go vet`, `gofmt`, tests Go (unitaires + intégration), ESLint, Prettier, Jest, Playwright |
| Infra | Docker, Docker Compose, GitHub Actions |

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS, Android, web)
│  ├─ App.tsx                   Providers : thème, React Query, session, navigation, toasts
│  ├─ src/api/                  Client HTTP typé (seul endroit qui parle au réseau) + WebSocket
│  ├─ src/hooks/                Session, requêtes, deck de découverte, temps réel
│  ├─ src/navigation/           Auth → Onboarding → Onglets + écrans empilés
│  ├─ src/screens/              auth, onboarding, discover, likes, chat, notifications, profile
│  ├─ src/components/           Composants métier (célébration de match, blocage/signalement…)
│  ├─ src/ui/                   Design system (Text, Button, TextField, Dialog, Toast…)
│  ├─ src/theme/                Palettes claire/sombre et tokens
│  ├─ scripts/e2e-smoke.js      Smoke test API du parcours critique
│  └─ e2e/web-flow.js           Test E2E UI (Playwright) du parcours critique
├─ services/api/
│  ├─ cmd/api/                  Point d'entrée HTTP (arrêt propre)
│  ├─ cmd/seed/                 Données de démonstration (interdit en production)
│  └─ internal/
│     ├─ app/                   Composition root + tests d'intégration HTTP
│     ├─ platform/              config, db (migrations), middleware, httpx, errors,
│     │                         media (stockage, images, URLs signées), realtime, mailer, logger
│     └─ features/              auth, profiles, photos, matching, chat, safety, notifications
│                               (handler → service → repository, + model, routes)
├─ scripts/qa.sh / qa.ps1       Toutes les vérifications en une commande
├─ docker-compose.yml           PostgreSQL + API (+ volume photos)
└─ docs/ARCHITECTURE.md         Détail de l'architecture, du modèle de données et de la sécurité
```

Règles : handlers minces, logique métier dans les services, SQL uniquement dans les repositories,
préoccupations transverses dans `platform`. Côté app, aucun écran n'appelle `fetch` directement :
tout passe par `src/api`. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation et lancement local

### Prérequis

- Docker et Docker Compose (ou Go 1.23+ et PostgreSQL 14+)
- Node.js 20+ et npm
- Pour un téléphone : l'app Expo Go ou un build de développement

### 1. API et base de données (Docker)

```bash
cp .env.example .env
# Renseignez JWT_SECRET, par exemple : openssl rand -base64 48
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

PowerShell :

```powershell
Copy-Item .env.example .env
# éditez .env et renseignez JWT_SECRET
docker compose up --build -d
Invoke-RestMethod http://localhost:18080/health
```

Les migrations s'appliquent automatiquement au démarrage. Les photos sont stockées dans le volume `uploads`.

### 2. Données de démonstration (facultatif)

```bash
docker compose exec api ./seed
```

Crée 16 profils de démonstration autour de Paris (voir [seed](#seed)).

### 3. Application

```bash
cd apps/mobile
cp .env.example .env      # EXPO_PUBLIC_API_URL
npm ci
npm run web               # navigateur
npm start                 # QR code pour Expo Go (iOS/Android)
```

- **Téléphone physique** : n'utilisez jamais `localhost`. Mettez `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`
  ou lancez `npm run start:phone`, qui détecte l'IP du réseau local.
- **Émulateur Android** : `http://10.0.2.2:18080` (valeur par défaut en développement).
- **Web** : ajoutez l'origine de la page (par défaut `http://localhost:8081`) à `ALLOWED_ORIGINS`.
- Si le port 18080 est déjà pris, changez `18080:8080` dans `docker-compose.yml` et `EXPO_PUBLIC_API_URL`.

### Sans Docker

```bash
createdb lueur
cd services/api
export DATABASE_URL=postgres://user:pass@localhost:5432/lueur?sslmode=disable
export JWT_SECRET=$(openssl rand -base64 48)
export PORT=18080 ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api
go run ./cmd/seed          # facultatif
```

### Tester un match seul

Créez deux comptes (deux navigateurs/onglets privés, ou web + téléphone) aux préférences compatibles, likez-vous
mutuellement, puis discutez. Avec le seed, `SEED_LIKE_EMAIL=votre@email docker compose exec api ./seed`
fait liker votre compte par tous les profils de démonstration : liker l'un d'eux crée un match immédiat.

## Variables d'environnement

Toutes sont documentées dans [.env.example](.env.example). Aucun secret n'est versionné ni présent dans l'app.

| Variable | Requis | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256 (≥ 32 caractères, valeurs « exemple » refusées). Sert aussi à dériver la clé de signature des URLs de photos. |
| `DATABASE_URL` | oui (hors Compose) | Connexion PostgreSQL |
| `PORT` | non (8080) | Port HTTP de l'API |
| `APP_ENV` | non (`development`) | `development`, `test` ou `production` (exige SMTP, interdit le seed) |
| `ALLOWED_ORIGINS` | web | Origines navigateur autorisées (CORS + WebSocket) |
| `TRUSTED_PROXIES` | derrière un proxy | IP/CIDR dont `X-Forwarded-For` est fiable (rate limiting par IP) |
| `UPLOAD_DIR` | non (`./data/uploads`) | Répertoire de stockage des photos |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | production | Envoi des codes de réinitialisation. Sans SMTP (dev), le code est écrit dans les logs de l'API. |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | non | Base locale Docker Compose |
| `SEED_PASSWORD`, `SEED_LIKE_EMAIL`, `SEED_LATITUDE`, `SEED_LONGITUDE` | non | Seed de démonstration |
| `EXPO_PUBLIC_API_URL` | app | URL publique de l'API embarquée dans l'app (non secrète) |
| `DATABASE_URL_TEST` | tests | Base jetable pour les tests d'intégration Go |

## Base de données, migrations et seed

### Modèle

`users`, `sessions`, `password_reset_tokens`, `profiles`, `preferences`, `interests`, `user_interests`,
`photos`, `swipes` (likes **et** passes), `matches`, `conversations`, `conversation_participants`,
`messages`, `blocks`, `reports`, `notifications`.
Clés étrangères en cascade (supprimer un utilisateur efface toutes ses données), contraintes `CHECK`
(âges, genres, longueurs, coordonnées, motifs de signalement), unicités (un swipe par paire, un match par paire
ordonnée, une conversation par match) et index sur tous les chemins chauds. Détails : [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#data-model).

### Migrations

Fichiers SQL embarqués dans `services/api/internal/platform/db/migrations`, appliqués **une seule fois**,
dans l'ordre, en transaction, et tracés dans `schema_migrations` (verrou consultatif pour plusieurs instances).
Ajoutez un fichier `008_….sql` pour faire évoluer le schéma ; ne modifiez jamais une migration déjà appliquée.

> Mise à jour depuis l'ancien boilerplate : l'API refuse de démarrer sur une base contenant l'ancien schéma
> non tracé. Repartez d'une base vide (`docker compose down -v`).

### Seed

`go run ./cmd/seed` (ou `docker compose exec api ./seed`) crée 16 membres de démonstration :

- marqués `users.is_demo = true`, e-mails `demo-<n>@demo.lueur.invalid` (domaine réservé, non routable) ;
- affichés avec un badge **« Démo »** dans l'app ; leurs photos sont des illustrations abstraites générées ;
- mot de passe `SEED_PASSWORD` (par défaut `DemoPassword1`) ;
- relancer le seed remplace les données de démo ; il **refuse de s'exécuter si `APP_ENV=production`**.

## Tests et qualité

Tout vérifier d'un coup :

```bash
DATABASE_URL_TEST=postgres://…/lueur_test ./scripts/qa.sh     # Windows : ./scripts/qa.ps1
```

| Commande | Contenu |
| --- | --- |
| `cd services/api && go test ./...` | Tests unitaires : âge, validations, distance, classement, JWT, rate limiting, images, URLs signées, mots de passe |
| idem avec `DATABASE_URL_TEST` | **Intégration HTTP** sur vraie base : auth complète, reset, profils, photos, parcours like → match → chat, temps réel, permissions, blocage/signalement, filtres de découverte, **likes simultanés** (un seul match), suppression de compte, rate limiting |
| `npm run typecheck` / `lint` / `format:check` | TypeScript strict, ESLint (zéro warning), Prettier |
| `npm test` | Jest : validations, formats, client API (refresh unique sur 401), synchronisation temps réel, composants |
| `npm run e2e:smoke` | Parcours critique contre une API en marche |
| `npm run e2e:web` | **E2E UI Playwright** : inscription → onboarding avec photo → like → match → message → réception temps réel → accusé de lecture → suppression du compte |

Pour l'E2E UI : API sur `API_URL` (défaut `http://localhost:18080`), app web servie sur `WEB_URL`
(défaut `http://localhost:8081`, par exemple `npm run build:web && cd dist && python3 -m http.server 8081`), origine ajoutée à `ALLOWED_ORIGINS`.

La CI GitHub Actions (`.github/workflows/ci.yml`) exécute backend (avec PostgreSQL et `-race`), mobile, puis smoke et E2E UI sur la stack Docker.

## Build et déploiement

- **Image API** : `docker build -t lueur-api services/api` (binaire statique, utilisateur non root,
  healthcheck, contient aussi `seed`). Prévoir un volume persistant sur `/app/data/uploads`.
- **Production** : `APP_ENV=production`, `JWT_SECRET` fort, PostgreSQL managé (`sslmode=require`),
  SMTP configuré, TLS terminé par un reverse proxy (renseigner `TRUSTED_PROXIES`), `ALLOWED_ORIGINS` limité au
  domaine web. Les migrations s'exécutent au démarrage.
- **Une seule instance d'API** par défaut : le hub temps réel, les tickets WebSocket et le rate limiting sont en
  mémoire (voir [limites](#limites-connues-et-pistes)).
- **App** : `npm run build:web` produit un site statique (`dist/`). Pour iOS/Android : EAS Build
  (`npx eas build`) en définissant `EXPO_PUBLIC_API_URL` sur l'URL HTTPS de l'API.

## API

Toutes les routes (sauf auth, `/health`, `/interests`, `/media`) exigent `Authorization: Bearer <accessToken>`.
Erreurs : `{"error": "<message lisible>", "code": "<code_machine>"}`. Listes paginées par curseur (`?cursor=`, `nextCursor`).

| Domaine | Routes |
| --- | --- |
| Auth | `POST /auth/register` · `POST /auth/login` · `POST /auth/refresh` · `POST /auth/logout` · `POST /auth/password/forgot` · `POST /auth/password/reset` · `GET /me` · `PUT /me/password` · `DELETE /me` |
| Profil | `GET /interests` · `GET,PUT /profile` · `PUT /profile/preferences` · `PUT,DELETE /profile/location` · `PATCH /profile/privacy` · `GET /profiles/:userId` |
| Photos | `GET,POST /photos` (multipart `photo`) · `PUT /photos/order` · `PUT,DELETE /photos/:photoId` · `GET /media/:key?exp&sig` |
| Matching | `GET /discover` · `POST /swipes` · `GET /likes/received` · `GET /matches` · `DELETE /matches/:matchId` |
| Chat | `GET /conversations` · `GET /conversations/unread-count` · `GET /conversations/:id` · `GET,POST /conversations/:id/messages` · `POST /conversations/:id/read` |
| Sécurité | `GET,POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` |
| Notifications | `GET /notifications` · `GET /notifications/unread-count` · `POST /notifications/read-all` · `POST /notifications/:id/read` |
| Temps réel | `POST /realtime/ticket` puis `GET /realtime?ticket=` (WebSocket) : `message.created`, `conversation.read`, `match.created`, `match.removed`, `notification.created` |

## Décisions techniques et produit

- **Construire sur l'existant** : Go/Gin/pgx et Expo étaient sains ; ils sont conservés. Supprimés car contraires au
  produit ou inutiles : facturation Stripe (paywall), multi-organisation, posts/commentaires/votes, liste globale
  des utilisateurs, module `files` qui n'enregistrait que des métadonnées, Tamagui (inutilisé), Maestro (obsolète,
  remplacé par Playwright sur le build web).
- **Gratuité réelle** : aucun quota de likes. Le seul plafond (120 swipes/min) bloque les scripts, pas les humains.
- **Majorité** : 18 ans minimum vérifiés côté serveur ; la date de naissance est **verrouillée** après saisie
  (empêche de contourner l'âge). Seul l'âge est exposé.
- **Localisation** : demande au premier plan seulement, précision « balanced » ; le serveur arrondit à ~1 km
  avant stockage ; la distance affichée est arrondie (pas de 5 km au-delà de 10 km) et masquable. Un membre sans
  localisation voit des profils sans filtre de distance, mais n'apparaît pas aux membres qui filtrent par distance.
- **Photos** : type détecté par signature binaire (pas par l'extension), garde-fou contre les bombes de
  décompression, orientation EXIF appliquée puis **ré-encodage JPEG** (supprime toutes les métadonnées, dont le GPS),
  noms de fichiers aléatoires, accès par **URLs signées HMAC expirant en 1 à 2 h** (stables par tranche horaire pour le cache).
- **Matching** : un swipe par paire et pour toujours (clé primaire) ; création du match sous verrou consultatif
  de la paire, donc deux likes simultanés donnent exactement un match. Un match annulé ne se reforme pas.
- **Recommandation** : filtres stricts en SQL (préférences mutuelles, blocages, déjà vus), puis `Ranker`
  remplaçable (score : intérêts communs, activité récente, proximité, intérêt réciproque, léger aléa).
- **Chat** : conversation créée avec le match ; chaque requête vérifie l'appartenance (404 sinon, sans révéler
  l'existence) et l'absence de blocage. Annuler le match ou bloquer supprime la conversation pour les deux.
- **Sessions** : access token 15 min (HS256, `exp` obligatoire, algorithme vérifié) + refresh token 30 jours
  stocké haché, rotation à chaque usage, révocation vérifiée à chaque requête (déconnexion et reset immédiats).
  Sur le web, les tokens restent en mémoire (pas de `localStorage`) ; sur mobile, dans le trousseau sécurisé.
- **Signalement** : bloque automatiquement ; 3 signalants distincts en 30 jours passent le compte en revue
  (`moderation_status = 'under_review'` : retiré de la découverte). Les signalements restent dans `reports` pour
  l'équipe de modération ; un compte `suspended` ne peut plus se connecter.
- **Notifications** : en base + temps réel ; les messages d'une même conversation sont regroupés en une notification.

## Limites connues et pistes

- **Mise à l'échelle horizontale** : hub WebSocket, tickets et rate limiting sont en mémoire. Pour plusieurs
  instances, brancher un bus partagé (PostgreSQL `LISTEN/NOTIFY` ou Redis) derrière l'interface `realtime.Publisher`
  et un store partagé pour le rate limiting.
- **Notifications push** (APNs/FCM via Expo) non incluses : nécessitent un projet EAS et des identifiants.
  Les notifications fonctionnent dans l'app et en temps réel.
- **Stockage objet** : `media.Storage` a une implémentation disque ; une implémentation S3/GCS se branche sans
  toucher aux features.
- **Modération** : pas d'interface d'administration ; les signalements se traitent en base (`reports`,
  `users.moderation_status`). La vérification d'e-mail à l'inscription n'est pas encore exigée.
