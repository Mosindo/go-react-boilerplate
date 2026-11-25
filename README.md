# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre moderne, **entièrement gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de likes limités, pas de publicité. Deux personnes créent leur profil, se découvrent, se likent, obtiennent un **match** quand l'intérêt est réciproque, puis discutent **en temps réel**.

| | |
| --- | --- |
| **Backend** | Go 1.24 · Gin · PostgreSQL 16 · pgxpool · SQL explicite (pas d'ORM) · JWT HS256 · WebSocket |
| **Mobile** | React Native 0.81 · Expo SDK 54 · TypeScript strict · React Navigation · TanStack Query · expo-image |
| **Infra** | Docker (image non-root) · Docker Compose · GitHub Actions |

## Sommaire

- [Fonctionnalités](#fonctionnalités)
- [Architecture](#architecture)
- [Installation rapide (Docker)](#installation-rapide-docker)
- [Variables d'environnement](#variables-denvironnement)
- [Base de données et migrations](#base-de-données-et-migrations)
- [Données de démonstration (seed)](#données-de-démonstration-seed)
- [Lancement local sans Docker](#lancement-local-sans-docker)
- [Application mobile](#application-mobile)
- [Tests et qualité](#tests-et-qualité)
- [Build et déploiement](#build-et-déploiement)
- [Référence de l'API](#référence-de-lapi)
- [Sécurité et confidentialité](#sécurité-et-confidentialité)
- [Décisions techniques](#décisions-techniques)
- [Limites connues et pistes](#limites-connues-et-pistes)

## Fonctionnalités

- **Compte** : inscription (18 ans et plus), connexion, sessions avec rotation des refresh tokens, récupération de compte par code envoyé par e-mail, changement de mot de passe, déconnexion, **suppression définitive du compte** (profil, photos, matchs, messages, fichiers).
- **Onboarding en 5 étapes** : prénom, date de naissance, genre → préférences (genres recherchés, tranche d'âge, distance) → bio, intention, centres d'intérêt, métier, ville, taille → photos → position approximative.
- **Photos** : jusqu'à 6 photos, compression sur l'appareil, ré-encodage serveur (métadonnées EXIF/GPS supprimées, orientation corrigée), ordre, photo principale, remplacement, suppression. Servies via des URL signées et expirantes.
- **Découverte** : pile de cartes avec swipe (ou boutons accessibles), carrousel de photos, distance approximative, centres d'intérêt communs mis en avant, profil complet. Filtres **mutuels** (genre, âge, distance des deux côtés).
- **Matching** : like/pass, match uniquement si réciproque, aucun doublon (contraintes + verrou transactionnel), un profil traité ne réapparaît jamais, annulation de match.
- **Chat** : réservé aux matchs, temps réel (WebSocket), heure d'envoi, lu/non lu (« Envoyé » / « Lu »), pagination, effacement local d'une conversation, badges de non-lus.
- **Notifications** in-app en temps réel (nouveau match), compteur de non-lus, « tout marquer comme lu ».
- **Confidentialité et modération** : masquer son profil de la découverte, masquer sa distance, actualiser sa position, **bloquer**, **signaler** (avec blocage automatique), liste des personnes bloquées.

## Architecture

```text
.
├─ services/api/                 API Go (un seul serveur HTTP Gin)
│  ├─ cmd/api/                   point d'entrée (arrêt propre, timeouts)
│  ├─ cmd/seed/                  données de démonstration
│  └─ internal/
│     ├─ app/                    composition root + tests d'intégration HTTP
│     ├─ platform/               préoccupations transverses
│     │  ├─ config/  db/ (migrations SQL embarquées)  errors/  httpx/
│     │  ├─ middleware/ (auth, CORS, en-têtes, rate limit, logs)
│     │  ├─ token/ (JWT)  urlsign/ (URL signées)  storage/ (fichiers)
│     │  ├─ mailer/ (SMTP)  realtime/ (hub WebSocket)  logger/  handlers/
│     └─ features/               un dossier par domaine
│        ├─ auth/  profiles/  photos/  matching/
│        └─ chat/  notifications/  safety/
├─ apps/mobile/                  application Expo (iOS, Android, web)
│  ├─ App.tsx
│  ├─ e2e/                       tests E2E (API + Playwright web)
│  ├─ scripts/                   e2e-smoke, start-phone
│  └─ src/
│     ├─ api/                    client HTTP unique + modules par domaine + realtime
│     ├─ hooks/                  session, requêtes React Query, synchro temps réel
│     ├─ navigation/             auth → onboarding → onglets
│     ├─ screens/                auth, onboarding, discover, chat, notifications, profile
│     ├─ components/             composants métier (photos, préférences, sécurité…)
│     ├─ ui/                     design system (Text, Button, TextField, Sheet, Toast…)
│     ├─ theme/  store/  utils/
├─ docs/ARCHITECTURE.md
├─ scripts/qa.sh, scripts/qa-lite.ps1
└─ docker-compose.yml
```

Chaque feature backend suit `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`) : handlers minces, règles métier dans les services, SQL uniquement dans les repositories. Côté mobile, aucun écran n'appelle le réseau directement : tout passe par `src/api/client.ts`. Détails : [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation rapide (Docker)

Prérequis : Docker Desktop / Docker Engine, Node.js 20+.

```bash
cp .env.example .env
# Renseignez JWT_SECRET dans .env (32+ caractères aléatoires) :
#   openssl rand -hex 32
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# Optionnel : 24 profils de démonstration autour de Paris
docker compose run --rm api ./seed
```

Puis lancez l'application (voir [Application mobile](#application-mobile)).

Ports par défaut : API `18080`, PostgreSQL `5432` (modifiables via `API_HOST_PORT` et `POSTGRES_HOST_PORT` si un service local occupe déjà le port).

## Variables d'environnement

Toutes les variables sont documentées dans [`.env.example`](.env.example). Aucun secret n'est versionné.

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | **oui** | — | Clé HS256 (≥ 32 caractères, valeurs « placeholder » refusées). Sert aussi à dériver la clé de signature des URL de photos. |
| `DATABASE_URL` | **oui** | — (défini par Compose) | Chaîne de connexion PostgreSQL. |
| `APP_ENV` | non | `development` | `production` : profils de démo masqués, Gin en mode release, codes de récupération envoyés uniquement par SMTP. |
| `PORT` | non | `8080` | Port d'écoute de l'API. |
| `UPLOAD_DIR` | non | `./data/uploads` | Dossier de stockage des photos (volume `uploads` dans Compose). |
| `ALLOWED_ORIGINS` | non | vide | Origines CORS autorisées (web). Les apps natives n'en ont pas besoin. |
| `TRUSTED_PROXIES` | non | vide | Proxys dont `X-Forwarded-For` est pris en compte (sinon l'IP réelle est utilisée pour le rate limiting). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | — | Envoi des codes de récupération. Sans SMTP en développement, le code est écrit dans les logs de l'API. |
| `EXPO_PUBLIC_API_URL` | mobile | `http://localhost:18080` (dev) | URL de l'API embarquée dans l'app. **Téléphone physique : IP LAN, jamais `localhost`.** |

## Base de données et migrations

Les migrations SQL sont dans `services/api/internal/platform/db/migrations/` et **embarquées dans le binaire**. Au démarrage, l'API applique les migrations manquantes, chacune dans une transaction, enregistrée dans `schema_migrations`, sous verrou consultatif (plusieurs instances peuvent démarrer en même temps).

| Migration | Contenu |
| --- | --- |
| `001_users_and_sessions.sql` | `users`, `sessions`, `password_reset_tokens` |
| `002_profiles.sql` | `profiles`, `preferences`, `interests` (catalogue), `user_interests`, `photos`, fonction `distance_km` |
| `003_matching_and_safety.sql` | `swipes`, `matches`, `blocks`, `reports` |
| `004_chat.sql` | `conversations`, `conversation_participants`, `messages` |
| `005_notifications.sql` | `notifications` |

Toutes les tables ont clés étrangères (`ON DELETE CASCADE` pour les données personnelles), contraintes `CHECK` (genres, âges ≥ 18, longueurs, coordonnées…), contraintes d'unicité (un swipe par paire, un match par paire stockée dans l'ordre canonique) et index sur les chemins chauds (découverte, conversations, messages `(conversation_id, created_at)`, notifications `(user_id, created_at)`…).

Pour ajouter une migration : créez `006_xxx.sql` ; elle sera appliquée au prochain démarrage. Ne modifiez jamais une migration déjà déployée.

> Cette version remplace le schéma de l'ancien boilerplate. Sur une base existante issue de l'ancien projet, repartez d'une base vide (`docker compose down -v`).

## Données de démonstration (seed)

```bash
# Docker
docker compose run --rm api ./seed                     # 24 profils autour de Paris
docker compose run --rm api ./seed -count 40 -lat 45.76 -lon 4.83
docker compose run --rm api ./seed -like-user vous@exemple.fr   # la moitié des démos vous ont déjà liké
docker compose run --rm api ./seed -reset              # supprime toutes les démos

# Sans Docker (depuis services/api)
go run ./cmd/seed -count 24
```

Les comptes de démonstration sont **clairement séparés** des vrais membres : e-mail sous le domaine réservé `demo.lueur.invalid`, colonne `users.is_demo`, mot de passe aléatoire jeté (connexion impossible), bio préfixée « [Profil de démonstration] », badge « Démo » dans l'app, illustrations abstraites (pas de photos de vraies personnes), **jamais affichés quand `APP_ENV=production`**, et le seed refuse de s'exécuter en production.

## Lancement local sans Docker

```bash
# PostgreSQL 16 accessible localement, puis :
cd services/api
export DATABASE_URL='postgresql://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export ALLOWED_ORIGINS='http://localhost:8081'
go run ./cmd/api
```

PowerShell :

```powershell
cd services/api
$env:DATABASE_URL='postgresql://app:app@localhost:5432/app?sslmode=disable'
$env:JWT_SECRET='<32+ caractères aléatoires>'
go run ./cmd/api
```

## Application mobile

```bash
cd apps/mobile
npm ci
npm run start          # QR code Expo Go / simulateurs
npm run web            # version web sur http://localhost:8081
npm run start:phone    # téléphone physique : détecte l'IP LAN et définit EXPO_PUBLIC_API_URL
```

- **Émulateur Android** : `10.0.2.2:18080` est utilisé automatiquement en développement.
- **Téléphone physique** : `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`. Vérifiez que rien d'autre n'écoute sur le port : `docker compose ps`, `netstat -ano | findstr :18080` (Windows) ou `lsof -i :18080`, puis `curl http://<IP_LAN>:18080/health`.
- **Web** : ajoutez l'origine web (`http://localhost:8081`) à `ALLOWED_ORIGINS`. Sur le web, les jetons restent en mémoire (pas de stockage persistant dans le navigateur) : recharger la page déconnecte.

Parcours pour tester rapidement : créez un compte → onboarding → `docker compose run --rm api ./seed -like-user <votre email>` → likez des profils de démo dans Découvrir pour obtenir des matchs.

## Tests et qualité

```bash
./scripts/qa.sh                                    # Linux/macOS : tout sauf l'E2E
API_URL=http://localhost:18080 ./scripts/qa.sh     # + smoke E2E de l'API
powershell -ExecutionPolicy Bypass -File .\scripts\qa-lite.ps1 [-ApiBaseUrl http://localhost:18080]
```

| Niveau | Commande | Couverture |
| --- | --- | --- |
| Unitaires Go | `cd services/api && go test ./...` | JWT strict, URL signées, rate limiter, stockage (path traversal), traitement d'image (EXIF, redimensionnement, fichiers invalides), validation e-mail/mot de passe/profil/préférences, âge, distance, messages |
| Intégration API | `DATABASE_URL_TEST=postgresql://…/app_test go test -p 1 ./...` | Serveur HTTP réel + PostgreSQL : cycle d'authentification (rotation, révocation, récupération), photos, découverte avec filtres mutuels, like/pass/match, **like mutuel concurrent → un seul match**, permissions du chat, WebSocket, accusés de lecture, pagination, blocage, signalement, suppression de compte. ⚠️ la base de test est vidée. |
| Unitaires mobile | `cd apps/mobile && npm test` | validation, formats, formulaire de profil, cache des messages |
| E2E API | `npm run e2e:smoke` (`E2E_API_URL`) | inscription → profil → photo → découverte → like → match → chat → lu → notification → blocage → suppression |
| E2E UI (web) | `npm run build:web`, servir `dist/` sur :8081, puis `npm run e2e:web` | Playwright dans Chromium : inscription → onboarding complet (upload photo, géolocalisation) → like → match → chat temps réel + « Lu » → notification → blocage → suppression du compte. Captures dans `e2e/web/.output/`. Nécessite Playwright (`npm i -g playwright && npx playwright install chromium`). |
| Qualité mobile | `npm run typecheck`, `npm run lint`, `npm run format:check` | TypeScript strict, ESLint (config Expo), Prettier |

La CI GitHub Actions (`.github/workflows/ci.yml`) exécute tout cela : gofmt/vet/tests (avec PostgreSQL), typecheck/lint/format/tests/build web, E2E API + Playwright, puis build de l'image Docker et démarrage Compose avec seed.

## Build et déploiement

**API** : l'image Docker (`services/api/Dockerfile`) est multi-étapes, statique, exécutée en utilisateur non-root, avec healthcheck et un volume `/data/uploads`. Elle contient aussi le binaire `seed`.

Recommandations de production :

1. `APP_ENV=production`, `JWT_SECRET` fort et propre à l'environnement, SMTP configuré.
2. PostgreSQL managé avec sauvegardes ; `DATABASE_URL` avec `sslmode=require`.
3. Volume persistant (ou disque réseau) pour `UPLOAD_DIR`, sauvegardé.
4. Reverse proxy TLS (Caddy, Nginx, load balancer) devant l'API ; renseignez `TRUSTED_PROXIES` ; le proxy doit laisser passer les WebSocket sur `/realtime`.
5. **Une seule instance d'API** (le hub temps réel et le rate limiting sont en mémoire) — voir les pistes ci-dessous pour scaler.

**Mobile** :

- Web : `EXPO_PUBLIC_API_URL=https://api.example.com npm run build:web` produit un site statique dans `dist/`.
- iOS / Android : builds natifs avec [EAS Build](https://docs.expo.dev/build/introduction/) (`npx eas build`), en définissant `EXPO_PUBLIC_API_URL` dans le profil de build. Identifiants : `app.lueur.mobile` (à adapter).

## Référence de l'API

Toutes les réponses d'erreur ont la forme `{"error": "message", "code": "code_stable"}`. Les routes marquées 🔒 exigent `Authorization: Bearer <accessToken>`.

| Méthode | Route | Description |
| --- | --- | --- |
| GET | `/health` | Santé (ping base) |
| POST | `/auth/register` · `/auth/login` | `{email, password}` → `{accessToken, refreshToken, user}` |
| POST | `/auth/refresh` · `/auth/logout` | `{refreshToken}` (rotation à chaque refresh) |
| POST | `/auth/password/forgot` | `{email}` → 202 (ne révèle jamais si l'e-mail existe) |
| POST | `/auth/password/reset` | `{code, password}` → 204, révoque toutes les sessions |
| GET / DELETE | `/me` 🔒 | Compte courant / suppression définitive `{password}` |
| PUT | `/me/password` 🔒 | `{currentPassword, newPassword}`, déconnecte les autres sessions |
| GET | `/me/profile` 🔒 | Profil, préférences, confidentialité, photos, complétude |
| PUT | `/me/profile` · `/me/preferences` · `/me/privacy` · `/me/location` 🔒 | Mises à jour (position arrondie à ~1 km) |
| GET | `/interests` 🔒 | Catalogue des centres d'intérêt |
| GET | `/profiles/:userId` 🔒 | Profil public (si visible : pas de blocage, découvrable ou matché) |
| GET / POST | `/me/photos` 🔒 | Liste / upload multipart (`photo`, JPEG/PNG ≤ 10 Mo) |
| PUT / DELETE | `/me/photos/:photoId` 🔒 | Remplacement (multipart) / suppression |
| PUT | `/me/photos/order` 🔒 | `{photoIds}` dans le nouvel ordre |
| GET | `/photos/:photoId?exp&sig` | Fichier photo (URL signée, expirante) |
| GET | `/discover?limit=` 🔒 | Profils à découvrir (profil complet requis) |
| POST | `/swipes` 🔒 | `{targetUserId, action: "like" \| "pass"}` → `{match}` si réciproque |
| DELETE | `/matches/:matchId` 🔒 | Annuler un match |
| GET | `/conversations` · `/conversations/unread-count` · `/conversations/:id` 🔒 | Matchs/conversations, non-lus |
| GET / POST | `/conversations/:id/messages` 🔒 | Messages (pagination `before`) / envoi `{body}` |
| POST | `/conversations/:id/read` 🔒 | Marquer comme lu (accusé de lecture temps réel) |
| DELETE | `/conversations/:id` 🔒 | Effacer la conversation de son côté |
| GET | `/notifications` · `/notifications/unread-count` 🔒 | Notifications (pagination `before`) |
| POST | `/notifications/:id/read` · `/notifications/read-all` 🔒 | Marquer comme lu |
| GET / POST | `/blocks` 🔒 | Liste / bloquer `{userId}` (met fin au match) |
| DELETE | `/blocks/:userId` 🔒 | Débloquer |
| POST | `/reports` 🔒 | `{userId, reason, details, block?}` (bloque par défaut) |
| POST | `/realtime/ticket` 🔒 | Ticket à usage unique (60 s) |
| GET | `/realtime?ticket=` | WebSocket : `message.created`, `conversation.read`, `match.created`, `match.ended`, `notification.created` |

## Sécurité et confidentialité

- **Authentification** : bcrypt ; JWT HS256 de 15 min avec validation stricte (`alg`, `iss`, `aud`, `exp`, `sub`, `sid`) ; refresh tokens aléatoires stockés hachés (SHA-256), rotation à chaque usage, réutilisation refusée ; **chaque requête vérifie que la session n'est pas révoquée** (déconnexion, changement/réinitialisation du mot de passe et suppression de compte coupent l'accès immédiatement) ; comparaison à temps constant contre l'énumération d'e-mails.
- **Autorisations côté serveur** : seuls les participants d'un match actif lisent/écrivent la conversation (404 sinon, sans révéler l'existence) ; les notifications ne peuvent pas être créées par les clients ; les profils bloqués/masqués ne sont pas visibles ; chaque mutation de photo est limitée à son propriétaire.
- **Validation** : e-mail normalisé, mot de passe (8+ caractères, lettre + chiffre), âge ≥ 18 (vérifié côté serveur et en base), énumérations, longueurs, caractères de contrôle supprimés, pagination bornée, corps JSON ≤ 1 Mo.
- **Fichiers** : type détecté par décodage réel (JPEG/PNG), taille ≤ 10 Mo, dimensions ≤ 40 Mpx (anti « decompression bomb »), **ré-encodage JPEG** (supprime EXIF/GPS et neutralise les fichiers polyglottes), noms de stockage aléatoires non devinables, clés validées (pas de path traversal), servis via URL HMAC expirantes.
- **Localisation** : coordonnées arrondies à 0,01° (~1 km) **avant** stockage, jamais renvoyées ; distance affichée arrondie (« À moins de 2 km », pas de 5 km au-delà de 20 km) ; affichage désactivable. Sur Android, seule la permission de localisation approximative est demandée.
- **Abus** : rate limiting (auth, récupération, upload, swipes anti-bot, messages, signalements), `ReadHeaderTimeout`, en-têtes de sécurité, CORS par liste blanche, proxys de confiance explicites, journaux sans secrets.
- **Suppression** : la suppression du compte efface en cascade toutes les données personnelles et les fichiers photo. Les signalements sont conservés anonymisés (`ON DELETE SET NULL`) pour la modération.

## Décisions techniques

| Sujet | Décision | Pourquoi |
| --- | --- | --- |
| Base existante | Conserver Go/Gin/pgx/Expo/React Query, supprimer Stripe, organisations, posts/commentaires, `/users` | Architecture saine ; le paiement contredit la gratuité ; `/users` exposait tous les e-mails |
| Migrations | Nouveau schéma, migrations suivies et transactionnelles | L'ancien runner rejouait tout à chaque démarrage |
| Temps réel | WebSocket (`github.com/coder/websocket`, seule nouvelle dépendance Go) + tickets à usage unique + polling de secours | React Native ne supporte pas le streaming `fetch` (SSE) ; les tickets évitent les jetons dans les URL |
| Matching | Swipes uniques par paire, verrou consultatif par paire, match stocké en ordre canonique | Garantit un seul match même avec deux likes simultanés (testé) |
| Classement | Centres d'intérêt communs → activité récente → distance, isolé dans `rankingOrder` | Simple, explicable, remplaçable par un vrai moteur de recommandation |
| Profils « passés » | Un profil qui vous a passé n'apparaît plus | Évite des likes impossibles à convertir |
| Après un unmatch / blocage | La paire ne peut plus rematcher | Choix le plus sûr pour la personne qui a mis fin |
| Conversation | Une conversation par match ; « supprimer » = effacement local (`cleared_at`) | Ne détruit pas l'historique de l'autre personne |
| Notifications | In-app + temps réel ; pas de push natif | Les push nécessitent des identifiants (APNs/FCM) propres au déploiement |
| Photos | Stockage disque derrière une interface `Storage` | Fonctionne immédiatement ; un stockage objet (S3…) se branche sans toucher aux features |
| Localisation | Pas de géocodage ; ville saisie librement | Aucun service tiers ni clé API nécessaire |
| UI | Identité « Lueur » sombre et chaleureuse (braise + or), typographie serif pour les titres, sans dépendance de police | Adulte, élégante, lisible (contrastes AA), rapide à charger |
| Gestes | `PanResponder` + `Animated` natifs, boutons équivalents | Pas de dépendance supplémentaire, accessible |
| Langue | Interface en français | Cible du produit |

## Limites connues et pistes

- **Mise à l'échelle horizontale** : hub temps réel et rate limiting en mémoire → une seule instance. Pour plusieurs instances : `LISTEN/NOTIFY` PostgreSQL ou Redis derrière l'interface `realtime.Publisher`, limiter partagé.
- **Modération** : les signalements sont stockés (`reports`, statut `open`) mais il n'existe pas encore d'interface d'administration ; la revue se fait en base.
- **Push natifs** (Expo Notifications) et **stockage objet** pour les photos : non activés par défaut.
- **Découverte** : le calcul des intérêts communs est fait pour tous les candidats filtrés ; pour de très gros volumes, pré-calculer ou utiliser PostGIS.

## Documentation complémentaire

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — architecture détaillée
- [CONTRIBUTING.md](CONTRIBUTING.md) · [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) · [CHANGELOG.md](CHANGELOG.md)
