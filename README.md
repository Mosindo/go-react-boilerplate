# Lueur — application de rencontre 100 % gratuite

**Lueur** est une application de rencontre moderne, complète et entièrement gratuite :
pas d'abonnement, pas de paywall, pas de « boost » payant, pas de likes limités, pas de publicité.

Deux personnes peuvent : créer un compte → créer leur profil (photos, infos, centres d'intérêt)
→ découvrir des profils proches correspondant à leurs préférences → liker ou passer →
obtenir un match en cas d'intérêt réciproque → discuter en temps réel → recevoir des
notifications → gérer préférences et confidentialité → bloquer ou signaler → supprimer leur compte.

| Découvrir | Profil complet | Messages | Activité |
| --- | --- | --- | --- |
| Pile de profils (swipe ou boutons), photos, distance approximative, centres d'intérêt | Toutes les photos, bio, infos, bloquer / signaler | Nouveaux matchs, conversations, non-lus, temps réel, accusés de lecture | Notifications de matchs et de messages |

---

## Sommaire

1. [Stack technique](#stack-technique)
2. [Architecture](#architecture)
3. [Installation et lancement local](#installation-et-lancement-local)
4. [Variables d'environnement](#variables-denvironnement)
5. [Base de données, migrations et seed](#base-de-données-migrations-et-seed)
6. [Tests et qualité](#tests-et-qualité)
7. [Build et déploiement](#build-et-déploiement)
8. [API](#api)
9. [Sécurité et vie privée](#sécurité-et-vie-privée)
10. [Principales décisions techniques et produit](#principales-décisions-techniques-et-produit)

---

## Stack technique

| Couche | Technologies |
| --- | --- |
| Mobile (iOS, Android, web) | Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 6, TanStack Query 5, expo-image, expo-image-picker / image-manipulator, expo-location, expo-secure-store |
| API | Go 1.24, Gin, PostgreSQL 16 via `pgxpool` (SQL explicite, sans ORM), JWT HS256, bcrypt, WebSocket (`gorilla/websocket`), `golang.org/x/image` |
| Infra | Docker (image non-root, healthcheck), Docker Compose, GitHub Actions |
| Qualité | `gofmt`, `go vet`, tests unitaires + intégration HTTP sur Postgres (race detector), ESLint (config Expo), Prettier, `tsc`, Playwright (E2E) |

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS / Android / web)
│  ├─ App.tsx                   Providers (thème, React Query, session) + navigation
│  ├─ e2e/                      Tests Playwright du parcours critique
│  └─ src/
│     ├─ api/                   Client HTTP (refresh token single-flight), endpoints typés, stockage sécurisé
│     ├─ hooks/                 Session, requêtes React Query, temps réel, localisation
│     ├─ features/              Écrans par domaine : auth, onboarding, discovery, matches,
│     │                         notifications, profile, settings, safety
│     ├─ navigation/            Auth → Onboarding → Onglets + piles
│     ├─ theme/                 Design system « Lueur » (clair / sombre)
│     ├─ ui/                    Composants réutilisables (Text, Button, TextField, Chip, Sheet, Toast…)
│     └─ utils/                 Formatage FR, validation, compression photo
├─ services/api/                API Go
│  ├─ cmd/api                   Serveur HTTP (arrêt gracieux)
│  ├─ cmd/seed                  Données de démonstration
│  └─ internal/
│     ├─ app/                   Racine de composition + tests d'intégration HTTP
│     ├─ features/              auth, profiles, photos, discovery, matches, safety,
│     │                         notifications, realtime — chacun : handler → service → repository
│     └─ platform/              config, db (migrations), middleware, httpx, authtoken, ratelimit,
│                               storage, media (URL signées), mailer, realtime (hub), geo, cursor, logger
├─ scripts/smoke.sh             Test de fumée sur une API en cours d'exécution
├─ docker-compose.yml           Postgres + API (+ volume photos)
└─ docs/ARCHITECTURE.md         Détails d'architecture
```

Règles : handlers fins, logique métier dans les services, SQL uniquement dans les repositories,
un seul pool de connexions, un seul serveur HTTP (Gin). Côté mobile, aucun écran n'appelle
`fetch` directement : tout passe par `src/api`. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation et lancement local

### Prérequis

- Docker + Docker Compose **ou** Go 1.24+ et PostgreSQL 16
- Node.js 20.19+ et npm
- Pour un téléphone physique : l'application Expo Go (ou un build de développement)

### 1. Configurer l'environnement

```bash
cp .env.example .env
# Renseignez JWT_SECRET (32 caractères minimum), par exemple :
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env
```

### 2. Démarrer Postgres et l'API

```bash
docker compose up --build -d
curl http://localhost:18080/health          # {"status":"ok"}
```

Sans Docker :

```bash
cd services/api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
export JWT_SECRET=$(openssl rand -base64 48)
export PORT=18080 ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api
```

Les migrations s'appliquent automatiquement au démarrage.

### 3. (Optionnel) Charger des profils de démonstration

```bash
docker compose exec api ./seed -lat 48.8566 -lon 2.3522 -count 24
# ou, hors Docker : cd services/api && go run ./cmd/seed -lat 48.8566 -lon 2.3522
```

Placez les profils de démo près de la position que votre appareil/navigateur communiquera.

### 4. Lancer l'application

```bash
cd apps/mobile
cp .env.example .env        # EXPO_PUBLIC_API_URL
npm ci
npm run web                 # navigateur (http://localhost:8081)
npm run ios                 # simulateur iOS
npm run android             # émulateur Android (API : http://10.0.2.2:18080)
```

**Téléphone physique** : n'utilisez jamais `localhost`. Mettez
`EXPO_PUBLIC_API_URL=http://<IP_LAN_de_votre_ordinateur>:18080` dans `apps/mobile/.env`, puis
`npx expo start` et scannez le QR code. Vérifiez que le port n'est pas déjà pris
(`docker compose ps`, `curl http://<IP_LAN>:18080/health`) ; sinon changez `API_HOST_PORT`.

## Variables d'environnement

### API (`.env` à la racine, lu par Docker Compose)

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | **oui** | — | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées). Sert aussi à dériver la clé de signature des URL photo. |
| `DATABASE_URL` | **oui** (hors Compose) | — | Connexion PostgreSQL. |
| `APP_ENV` | non | `development` | `development`, `production` ou `test`. En production, SMTP est obligatoire et le seed est refusé. |
| `PORT` | non | `8080` | Port d'écoute de l'API. |
| `API_HOST_PORT` | non | `18080` | Port exposé par Compose. |
| `UPLOAD_DIR` | non | `./data/uploads` | Stockage des photos traitées (volume `uploads` dans Compose). |
| `ALLOWED_ORIGINS` | non | `http://localhost:8081` (Compose) | Origines navigateur autorisées (CORS, WebSocket). Les apps natives n'envoient pas d'origine. |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies dont `X-Forwarded-For` est pris en compte (rate limiting par IP). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | en production | vide / `587` | Envoi des codes de réinitialisation. En développement sans SMTP, les emails sont écrits dans les logs de l'API. |
| `DATABASE_URL_TEST` | non | — | Base dédiée aux tests d'intégration (`go test`). |
| `DEMO_PASSWORD` | non | `DemoLueur2026` | Mot de passe des comptes de démonstration créés par le seed. |
| `POSTGRES_USER/PASSWORD/DB/HOST_PORT` | non | `postgres/postgres/app/5432` | Conteneur Postgres de Compose. |

### Mobile (`apps/mobile/.env`)

| Variable | Rôle |
| --- | --- |
| `EXPO_PUBLIC_API_URL` | URL de l'API. Seules les variables `EXPO_PUBLIC_*` sont embarquées : n'y mettez **jamais** de secret. |

## Base de données, migrations et seed

- Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`)
  et appliquées **une seule fois chacune**, dans l'ordre, dans une transaction, sous verrou
  consultatif (plusieurs instances peuvent démarrer en même temps). Elles sont tracées dans
  `schema_migrations`. Pour ajouter une évolution : créer `002_xxx.sql`.
- Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`
  (catalogue de référence), `user_interests`, `photos`, `swipes` (likes et passes), `matches`
  (un match **est** la conversation 1:1), `messages`, `blocks`, `reports`, `notifications`.
- Contraintes : clés étrangères avec `ON DELETE CASCADE` (la suppression d'un compte efface toutes
  ses données ; les signalements sont conservés avec une référence nulle), contraintes `CHECK`
  (âge, genre, longueurs, coordonnées, valeurs d'énumération), unicité (email, swipe par paire,
  match par paire ordonnée, position de photo, signalement par paire), index sur tous les chemins
  chauds (découverte géographique, conversations par activité, messages par match, notifications).
- Une base créée par l'ancien boilerplate est détectée et refusée : réinitialisez-la
  (`docker compose down -v`).

**Seed de démonstration** (`cmd/seed`) : crée des profils clairement séparés des vrais comptes —
`users.is_demo = true`, domaine réservé `@demo.lueur.invalid`, ville « Démo », mention
« Profil de démonstration » dans la bio, images abstraites générées (aucun visage réel). Il refuse
de tourner si `APP_ENV=production`. Suppression : `go run ./cmd/seed -purge`.

## Tests et qualité

### API

```bash
cd services/api
gofmt -l .                 # doit être vide
go vet ./...
export DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable
go test -race ./...
go build ./cmd/api ./cmd/seed
```

- **Unitaires** : validation de profil (18 ans révolus, date immuable, prénom, longueurs),
  préférences, politique de mot de passe, email, classement de la découverte, arrondi et
  « bucketisation » des distances, pipeline photo (rejet PHP/SVG/HTML, suppression de charge
  utile cachée, redimensionnement), URL signées, JWT (alg `none`/HS512/expiration), rate limiter,
  hub temps réel et tickets, curseurs, stockage (path traversal).
- **Intégration HTTP** (Postgres réel, `internal/app`) : cycle complet d'authentification
  (refresh rotatif, réutilisation refusée, logout, reset par code, anti-bruteforce, rate limiting),
  validation et confidentialité du profil (coordonnées jamais renvoyées), **parcours critique**
  inscription → profil → découverte → like → match → conversation → notifications → temps réel
  (WebSocket) → désappariement, **likes simultanés** (un seul match), permissions du chat
  (un tiers ne peut ni lire, ni écrire, ni supprimer), blocage et signalement (seuil de modération),
  sécurité des photos, suppression de compte (aucune donnée résiduelle, fichiers supprimés).
  Sans `DATABASE_URL_TEST`/`DATABASE_URL`, ces tests sont ignorés.

### Mobile

```bash
cd apps/mobile
npm run typecheck          # tsc --noEmit
npm run lint               # ESLint (config Expo)
npm run format:check       # Prettier
```

### E2E (interface)

Le parcours critique (inscription → onboarding avec vraie photo et géolocalisation → découverte →
like → match → chat temps réel avec accusé de lecture → liste des conversations → notifications)
est automatisé avec Playwright sur le build web :

```bash
# API démarrée sur :18080 avec ALLOWED_ORIGINS=http://localhost:8081
cd apps/mobile
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
npx playwright install chromium     # une seule fois
npm run test:e2e
```

### Test de fumée d'un environnement

```bash
./scripts/smoke.sh http://localhost:18080
```

La CI (`.github/workflows/ci.yml`) exécute tout cela : format, vet, tests avec race detector sur
Postgres, typecheck, lint, format, export web, smoke test, Playwright et build Docker.

## Build et déploiement

- **API** : `docker build -t lueur-api ./services/api` produit une image Alpine non-root (~46 Mo)
  avec healthcheck, contenant `api` et `seed`. Montez un volume persistant sur `/app/data`
  (photos). Configurez `APP_ENV=production`, `JWT_SECRET`, `DATABASE_URL` (TLS recommandé),
  SMTP, `ALLOWED_ORIGINS` et `TRUSTED_PROXIES` si l'API est derrière un proxy. Terminez le TLS
  au niveau du proxy (HSTS est envoyé quand la requête est en HTTPS).
- **Mise à l'échelle** : le rate limiting et le hub WebSocket sont en mémoire (une instance).
  Pour plusieurs réplicas : sticky sessions ou bus partagé (Postgres `LISTEN/NOTIFY`, Redis) branché
  sur l'interface `realtime.Publisher`, rate limiting au niveau du proxy, et stockage objet
  (implémenter `storage.Store`, par exemple S3) à la place du disque local.
- **Mobile** : `npx expo export --platform web` pour le web ; iOS/Android via EAS Build
  (`eas build`) en définissant `EXPO_PUBLIC_API_URL` (HTTPS) pour chaque profil de build. Les
  identifiants `app.lueur.mobile` et les textes d'autorisation (localisation approximative, photos)
  sont déjà configurés dans `app.json`.

## API

Toutes les réponses d'erreur ont la forme `{"error": "message", "code": "code_stable"}`.
Les listes sont paginées par curseur (`?cursor=…&limit=…`). Toutes les routes, sauf
`/health`, `/auth/*` et `/media/*` (URL signées), exigent `Authorization: Bearer <accessToken>`.

| Domaine | Routes |
| --- | --- |
| Santé | `GET /health` |
| Auth | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `POST /auth/password/forgot`, `POST /auth/password/reset`, `GET /me`, `POST /me/password`, `DELETE /me` |
| Profil | `GET /interests`, `GET /profile`, `PUT /profile`, `PUT /profile/location`, `PUT /profile/preferences`, `PUT /profile/privacy`, `GET /profiles/:userId` |
| Photos | `GET /profile/photos`, `POST /profile/photos` (multipart `photo`), `PUT /profile/photos/order`, `PUT /profile/photos/:photoId`, `DELETE /profile/photos/:photoId`, `GET /media/:key?exp&sig` |
| Découverte | `GET /discovery?limit&exclude`, `POST /discovery/swipes` `{targetId, action: like\|pass}` |
| Matchs & chat | `GET /matches`, `GET /matches/:id`, `DELETE /matches/:id`, `GET /matches/:id/messages`, `POST /matches/:id/messages`, `POST /matches/:id/read` |
| Sécurité | `GET /blocks`, `POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` |
| Notifications | `GET /notifications`, `GET /notifications/unread-count`, `POST /notifications/:id/read`, `POST /notifications/read-all` |
| Temps réel | `POST /realtime/ticket` puis WebSocket `GET /realtime?ticket=…` — événements `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new` |

## Sécurité et vie privée

- **Authentification** : bcrypt (coût 12), politique de mot de passe, emails normalisés, temps de
  réponse constant à la connexion (pas d'énumération), JWT HS256 de 15 min avec `iss`, `aud`, `exp`,
  `nbf` obligatoires et algorithme verrouillé ; refresh tokens opaques (30 jours) stockés hachés et
  **rotatifs à usage unique** ; réinitialisation par code à 6 chiffres (15 min, 5 essais, 3 demandes
  par heure, réponse identique que le compte existe ou non) qui révoque toutes les sessions.
- **Autorisations côté serveur** : chaque requête de chat filtre sur l'identifiant de l'appelant
  (un tiers reçoit 404) ; un like n'est accepté que si la cible est réellement éligible (préférences
  mutuelles, profil actif, aucun blocage) ; la date de naissance (18 ans minimum) est vérifiée et
  ne peut plus être modifiée.
- **Localisation** : demandée uniquement au moment utile ; seule une position arrondie à ~1 km est
  stockée ; les coordonnées ne sont jamais renvoyées ; les distances affichées sont arrondies par
  paliers (1 km minimum, puis pas de 5 km au-delà de 20 km) pour empêcher la triangulation ; option
  pour masquer la distance ; Android ne demande que la localisation approximative.
- **Photos** : type détecté sur le contenu (pas l'extension), dimensions vérifiées avant décodage
  (anti « bombe de décompression »), ré-encodage JPEG systématique (suppression EXIF/GPS et de toute
  charge cachée), clés aléatoires non devinables, accès uniquement par **URL signées expirantes**
  remises aux personnes autorisées, `nosniff` et CSP sur les fichiers servis.
- **Abus** : rate limiting (global par IP, identifiants, actions sensibles, uploads, messages,
  swipes, signalements, tickets WebSocket), limites de taille des corps de requête, blocage
  bidirectionnel, signalement (qui bloque automatiquement) et mise en revue automatique d'un profil
  signalé par 3 personnes distinctes.
- **Données** : suppression de compte confirmée par mot de passe, effaçant profil, photos (fichiers
  compris), swipes, matchs, messages, notifications et sessions ; jetons stockés dans le
  Keychain/Keystore (expo-secure-store) ; en-têtes de sécurité HTTP ; aucun secret côté client.

## Principales décisions techniques et produit

- **Conservation de la stack existante** (Go/Gin/pgx, Expo/React Query) : elle était saine ; les
  modules génériques inadaptés (facturation Stripe, posts, commentaires, organisations, liste
  publique des utilisateurs, chat ouvert à tous, fichiers sans upload réel) ont été retirés, ainsi
  que la dépendance Tamagui inutilisée.
- **Un match = une conversation** : pas de table `conversations` redondante ; « supprimer la
  conversation » = **ne plus matcher** (effacée pour les deux, comme dans les apps de référence),
  les deux profils ne se recroisent plus.
- **Passes définitives** : un profil traité (like ou passe) ne réapparaît jamais ; pas d'« annuler »
  pour rester simple et prévisible.
- **Découverte** : règles strictes en SQL (préférences mutuelles de genre et d'âge, distance, profil
  actif avec photo, blocages, déjà vus), puis classement isolé dans `discovery/ranking.go`
  (réciprocité, proximité, activité récente, centres d'intérêt communs, légère variation) — prêt à
  être remplacé par un algorithme plus avancé sans toucher aux règles de sécurité.
- **Swipe → match sans condition de course** : verrou consultatif transactionnel sur la paire ;
  testé avec des likes simultanés.
- **Temps réel** : WebSocket avec tickets à usage unique (le jeton d'accès n'apparaît jamais dans une
  URL), re-synchronisation REST à chaque reconnexion et polling de secours dans le chat.
- **Notifications** : in-app et temps réel ; les notifications de messages sont regroupées par
  conversation (une seule non lue à la fois) et effacées à la lecture. Les notifications push
  natives (APNs/FCM via Expo) ne sont pas activées : elles nécessitent des identifiants de projet
  EAS propres au déploiement.
- **Modération** : les signalements sont stockés pour revue humaine (pas d'interface
  d'administration fournie) ; un seuil de 3 signalants masque automatiquement le profil
  (`moderation_status = 'under_review'`).
- **Interface** : identité originale « Lueur » (ivoire chaud, encre, accent braise ; titres en
  serif), mode sombre automatique, onboarding en 7 étapes, swipe avec retour visuel ou boutons
  accessibles, états de chargement, vides et d'erreur partout, textes en français.
