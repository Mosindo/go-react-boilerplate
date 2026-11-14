# Lueur

**Lueur** est une application de rencontre **100 % gratuite** : pas d'abonnement, pas de boost payant, pas de limite de likes, pas de publicité. Deux personnes créent leur profil, se découvrent, se likent ; quand l'intérêt est réciproque, c'est un match et elles peuvent discuter en temps réel.

- API : Go · Gin · PostgreSQL (pgx, SQL explicite, sans ORM) · WebSocket
- Application : React Native / Expo (iOS, Android, Web) · TypeScript · React Query
- Infra : Docker · Docker Compose · GitHub Actions

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Architecture](#architecture)
3. [Démarrage rapide (Docker)](#démarrage-rapide-docker)
4. [Développement local sans Docker](#développement-local-sans-docker)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données et migrations](#base-de-données-et-migrations)
7. [Données de démonstration](#données-de-démonstration)
8. [Tests et qualité](#tests-et-qualité)
9. [Build et déploiement](#build-et-déploiement)
10. [Référence de l'API](#référence-de-lapi)
11. [Décisions techniques et produit](#décisions-techniques-et-produit)

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation du refresh token, déconnexion, récupération de compte par code à 6 chiffres envoyé par e-mail, suppression définitive du compte |
| Onboarding | Prénom, date de naissance (18 ans minimum), genre, qui rencontrer, type de relation, photos, bio, profession, centres d'intérêt, position approximative |
| Profil | Édition complète, jusqu'à 6 photos (ajout, remplacement, suppression, ordre, photo principale), aperçu public |
| Découverte | Cartes swipables (ou boutons), navigation dans les photos, profil complet, like / passer, filtres mutuels genre/âge/distance, classement (personnes qui vous ont liké, intérêts communs, activité, proximité) |
| Matching | Like réciproque = match, protection contre les doublons et les likes répétés, création de match sûre en cas de likes simultanés, annulation de match |
| Chat | Conversation après match uniquement, temps réel (WebSocket), heure d'envoi, statut lu/envoyé, compteur de non-lus, pagination, effacement local de l'historique |
| Notifications | Centre d'activité (nouveaux matchs, messages regroupés par conversation), badges, toasts en temps réel |
| Confidentialité | Masquer son profil, masquer sa distance, position arrondie à ~1 km, distance affichée approximative, métadonnées photo (GPS) supprimées |
| Sécurité | Blocage / déblocage, signalement (6 motifs) avec blocage automatique, suspension de la découverte après 3 signalements distincts, rate limiting, autorisations côté serveur |
| UI | Identité visuelle originale, mode clair/sombre automatique, états de chargement / vide / erreur, accessibilité (rôles, labels, contrôles ajustables) |

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS / Android / Web)
│  ├─ App.tsx                   Fournisseurs : thème, React Query, toasts, auth, temps réel
│  ├─ e2e/                      Test end-to-end Playwright (parcours critique sur le build web)
│  └─ src/
│     ├─ api/                   Client HTTP (refresh automatique), endpoints, types, session
│     ├─ realtime/              Client WebSocket + pont vers le cache React Query
│     ├─ state/                 AuthProvider, QueryClient, hooks de données
│     ├─ navigation/            Pile auth / onboarding / application (onglets + écrans)
│     ├─ features/              auth · onboarding · discovery · chat · notifications · profile · safety
│     ├─ ui/                    Design system (Text, Button, TextField, Chip, Sheet, Toast…)
│     ├─ theme/                 Palettes claire/sombre, typographie, espacements
│     └─ utils/                 Formatage (dates FR, distances), validation, libellés
├─ services/api/                API Go
│  ├─ cmd/api/                  Point d'entrée + tests d'intégration HTTP/WebSocket
│  ├─ cmd/seed/                 Générateur de profils de démonstration
│  └─ internal/
│     ├─ platform/              config · db (migrations) · middleware (auth, CORS, rate limit…)
│     │                         httpx · logger · mailer · media (URL signées) · realtime · storage
│     └─ features/              auth · profiles · photos · matching · chat · notifications · safety
├─ docker-compose.yml
├─ .env.example
└─ docs/ARCHITECTURE.md         Détails : flux, modèle de données, sécurité
```

Chaque fonctionnalité backend suit `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`) : les handlers sont fins, la logique métier est dans les services, le SQL uniquement dans les repositories. Côté application, aucun écran n'appelle `fetch` directement : tout passe par `src/api`.

Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) pour le détail.

## Démarrage rapide (Docker)

Prérequis : Docker, Node.js 20+, npm.

```bash
cp .env.example .env
# Renseignez au minimum JWT_SECRET et POSTGRES_PASSWORD :
#   JWT_SECRET=$(openssl rand -base64 48)
#   POSTGRES_PASSWORD=$(openssl rand -hex 16)
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

L'API écoute sur `http://localhost:18080`. Les migrations s'appliquent automatiquement au démarrage.

Ajoutez des profils de démonstration (facultatif) :

```bash
docker compose exec api ./seed                       # 16 profils autour de Paris
docker compose exec api ./seed -like vous@exemple.fr # ils vous likent déjà → match immédiat
```

Lancez l'application :

```bash
cd apps/mobile
npm ci
cp .env.example .env     # EXPO_PUBLIC_API_URL
npm run web              # navigateur (http://localhost:8081)
npm run ios              # simulateur iOS
npm run android          # émulateur Android
```

> **Téléphone physique** : `localhost` désigne le téléphone lui-même. Utilisez l'IP de votre machine sur le réseau local : `EXPO_PUBLIC_API_URL=http://192.168.x.y:18080`. Si le port 18080 est déjà pris, changez `API_HOST_PORT` dans `.env` et alignez `EXPO_PUBLIC_API_URL`.

## Développement local sans Docker

```bash
# PostgreSQL 15+ en local
createdb lueur
cd services/api
export DATABASE_URL='postgres://<user>:<password>@localhost:5432/lueur?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api          # PORT=8080 par défaut ; utilisez PORT=18080 pour coller à l'app
go run ./cmd/seed         # profils de démonstration
```

## Variables d'environnement

### API (`services/api`, ou `.env` à la racine pour Docker Compose)

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `DATABASE_URL` | oui | — | Connexion PostgreSQL (pool partagé unique) |
| `JWT_SECRET` | oui | — | Clé HS256 des access tokens, ≥ 32 caractères, valeurs « placeholder » refusées. Dérive aussi la clé de signature des URL de photos |
| `APP_ENV` | non | `development` | `development`, `test` ou `production` (mode Gin release, SMTP obligatoire) |
| `PORT` | non | `8080` | Port HTTP |
| `UPLOAD_DIR` | non | `./data/uploads` | Stockage des photos (volume `uploads` en Docker) |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS + WebSocket), séparées par des virgules. Les apps natives n'en ont pas besoin |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies dont `X-Forwarded-For` est fiable (sinon l'IP réelle de connexion est utilisée pour le rate limiting) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | en production | `587` pour le port | Envoi des codes de réinitialisation. Sans SMTP (développement), le code est écrit dans les logs de l'API |

Docker Compose utilise en plus `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` et `API_HOST_PORT` (voir [.env.example](.env.example)).

### Application (`apps/mobile/.env`)

| Variable | Rôle |
| --- | --- |
| `EXPO_PUBLIC_API_URL` | URL de l'API. En développement, valeur par défaut `http://localhost:18080` (iOS/Web) ou `http://10.0.2.2:18080` (émulateur Android). **Obligatoire** pour un build de production |

Aucun secret n'est présent côté application : seules des valeurs publiques préfixées `EXPO_PUBLIC_` y sont lues. Les fichiers `.env` sont ignorés par git.

## Base de données et migrations

Le schéma est dans `services/api/internal/platform/db/migrations/*.sql`, embarqué dans le binaire. Au démarrage, l'API applique chaque migration **une seule fois**, dans l'ordre, chacune dans sa transaction, sous un verrou consultatif (plusieurs instances peuvent démarrer en même temps). Les versions appliquées sont enregistrées dans `schema_migrations`.

Ajouter une migration : créez `002_ma_modification.sql` (ne modifiez jamais une migration déjà appliquée).

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (likes et passes), `matches` (un match = une conversation), `conversation_participants` (lu / effacé), `messages`, `blocks`, `reports`, `notifications`. Contraintes `CHECK`, clés étrangères, unicités et index sont détaillés dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md#modèle-de-données).

> Une base créée par l'ancien boilerplate (tables `organizations`, `posts`…) est détectée et refusée avec un message explicite : repartez d'une base vide (`docker compose down -v`).

## Données de démonstration

`go run ./cmd/seed` (ou `./seed` dans le conteneur) crée 16 profils variés autour d'un point (`-lat`, `-lng`, Paris par défaut) :

- comptes marqués `users.is_demo = true`, e-mails en `@demo.lueur.invalid` (domaine réservé, non routable) ;
- bios préfixées `[Démo]`, ville « Démo », photos **abstraites générées** (jamais de visages réels) ;
- commande idempotente ; `-reset` supprime tous les comptes de démo et leurs fichiers ;
- `-like <email>` fait liker votre compte par tous les profils de démo pour tester le match et le chat ;
- **refusée si `APP_ENV=production`**.

## Tests et qualité

```bash
# API
cd services/api
gofmt -l .                     # doit être vide
go vet ./...
DATABASE_URL_TEST='postgres://…/lueur_test?sslmode=disable' go test -race ./...
go build ./cmd/api ./cmd/seed

# Application
cd apps/mobile
npm run typecheck
npm run lint
npm run build:web
```

**Tests unitaires** (Go) : validation de profil (âge ≥ 18, prénom, préférences), arrondi de la position et des distances, pipeline photo (types refusés, images trop petites, orientation EXIF, redimensionnement, ré-encodage), signature des URL, stockage (path traversal), normalisation des messages, configuration.

**Tests d'intégration** (`cmd/api/integration_test.go`, vrai serveur HTTP + PostgreSQL + WebSocket) : cycle d'authentification complet, réinitialisation de mot de passe (usage unique, verrouillage après 5 essais), profil/photos (mineurs refusés, upload HTML refusé, limite de 6, URL signées falsifiées refusées, réordonnancement), découverte (filtres mutuels, distance, aucune coordonnée exposée), likes/passes/doublons, match et événement temps réel, likes simultanés → un seul match, permissions du chat (non-membres → 404), pagination, accusés de lecture, effacement local, annulation de match, blocage, signalements et suspension, suppression de compte (données et fichiers effacés, sockets fermées), rate limiting.

Les tests d'intégration sont ignorés si aucune base n'est configurée (`DATABASE_URL_TEST` ou `DATABASE_URL`).

**Test end-to-end** (`apps/mobile/e2e/web-critical-flow.js`) : pilote le build web dans Chromium contre une vraie API — inscription → onboarding (upload photo, géolocalisation) → découverte → like → match → message → réception en temps réel → « Lu » → notifications → blocage → suppression du compte. Le second membre est piloté par l'API.

```bash
# API sur :18080 avec ALLOWED_ORIGINS=http://localhost:8081, puis :
cd apps/mobile
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
python3 -m http.server 8081 --directory dist &
npx playwright install chromium   # première fois
npm run e2e:web                   # captures dans e2e/.artifacts/
```

La CI GitHub Actions (`.github/workflows/ci.yml`) exécute tout cela : gofmt, vet, tests avec PostgreSQL, build ; typecheck, lint, build web ; parcours end-to-end ; image Docker avec contrôle `/health`.

## Build et déploiement

**API** : l'image Docker (`services/api/Dockerfile`) est multi-étapes, tourne en utilisateur non-root et contient `api` et `seed`. En production :

- `APP_ENV=production`, `JWT_SECRET` aléatoire long, configuration SMTP ;
- base PostgreSQL managée avec sauvegardes, `DATABASE_URL` avec `sslmode=require` ;
- volume persistant pour `UPLOAD_DIR` (ou un stockage objet via l'interface `storage.Storage`) ;
- TLS terminé par un reverse proxy déclaré dans `TRUSTED_PROXIES` ; HSTS est envoyé automatiquement en HTTPS ;
- **une seule instance** de l'API tant que le hub temps réel et le rate limiter sont en mémoire (voir décisions ci-dessous).

**Application** : builds natifs via EAS (`npx eas build -p ios|android`) avec `EXPO_PUBLIC_API_URL` pointant vers l'API publique en HTTPS ; version web via `npm run build:web` (dossier `dist/` statique, à servir derrière HTTPS, origine ajoutée à `ALLOWED_ORIGINS`).

## Référence de l'API

Toutes les réponses d'erreur ont la forme `{"error": "message lisible", "code": "code_machine"}`. Les routes marquées 🔒 exigent `Authorization: Bearer <accessToken>`.

| Méthode | Route | Description |
| --- | --- | --- |
| GET | `/health` | Santé (ping base) |
| POST | `/auth/register` · `/auth/login` | Création de compte / connexion → `accessToken` (15 min), `refreshToken` (30 j) |
| POST | `/auth/refresh` · `/auth/logout` | Rotation du refresh token / révocation de la session |
| POST | `/auth/password/forgot` · `/auth/password/reset` | Envoi d'un code / nouveau mot de passe (toutes les sessions sont révoquées) |
| GET · DELETE 🔒 | `/me` | Compte courant / suppression définitive (mot de passe requis) |
| GET | `/interests` | Catalogue des centres d'intérêt |
| GET · PUT 🔒 | `/me/profile` | Profil, préférences, photos, `complete` / mise à jour |
| PUT 🔒 | `/me/preferences` · `/me/privacy` | Préférences de rencontre / visibilité et affichage de la distance |
| PUT · DELETE 🔒 | `/me/location` | Position approximative / suppression |
| GET · POST 🔒 | `/me/photos` | Liste / upload multipart (`photo`) |
| PUT · DELETE 🔒 | `/me/photos/:id` · PUT `/me/photos/order` | Remplacement, suppression / ordre |
| GET | `/media/photos/:id?exp=&sig=` | Photo via URL signée (1 à 2 h) |
| GET 🔒 | `/profiles/:userId` | Profil public (si visible et non bloqué) |
| GET 🔒 | `/discovery?limit=` | Prochains profils à découvrir |
| POST 🔒 | `/swipes` | `{targetUserId, action: like|pass}` → `{matched, match?}` |
| DELETE 🔒 | `/matches/:matchId` | Annuler un match |
| GET 🔒 | `/conversations` | Matchs et conversations, dernier message, non-lus |
| GET · POST · DELETE 🔒 | `/conversations/:matchId/messages` | Messages (`before`, `limit`) / envoi / effacement local |
| POST 🔒 | `/conversations/:matchId/read` | Marquer comme lu |
| GET 🔒 | `/notifications` · POST `/notifications/:id/read` · `/notifications/read-all` | Centre d'activité |
| GET · POST 🔒 | `/blocks` · DELETE `/blocks/:userId` | Blocages |
| POST 🔒 | `/reports` | Signalement `{userId, reason, details}` |
| GET | `/ws` | WebSocket ; premier message `{"type":"auth","token":"…"}` puis événements `message.created`, `conversation.read`, `match.created`, `match.removed`, `notification.created` |

## Décisions techniques et produit

- **On garde la base existante** (Go/Gin/pgx, Expo, React Query) : elle était saine. Ont été retirés car incompatibles avec une app de rencontre gratuite : facturation Stripe, multi-tenant « organisations », posts/commentaires/votes, annuaire `/users` (exposait les e-mails), module fichiers sans upload réel, Tamagui (inutilisé).
- **Gratuité réelle** : aucune fonctionnalité payante ni quota de likes. Les limites de débit existantes (ex. 120 swipes/min, 40 messages/min) protègent uniquement contre les robots et le spam.
- **Âge** : 18 ans minimum vérifié côté serveur à la création et à chaque modification du profil ; seul l'âge est exposé, jamais la date de naissance.
- **Localisation minimale** : position lue ponctuellement (pas de suivi en arrière-plan, permission « approximative » sur Android), arrondie à 2 décimales (~1 km) côté client *et* serveur, jamais renvoyée ; distance affichée arrondie (« à moins de 1 km », puis par pas de 5 km au-delà de 20 km). Sans position, la découverte fonctionne sans filtre de distance.
- **Photos** : type détecté par le contenu (JPEG/PNG/WebP), 10 Mo max, 200 px min, 8000 px / 40 Mpx max (anti « bombe de décompression »), orientation EXIF appliquée, redimensionnement à 1600 px et ré-encodage JPEG qui **supprime toutes les métadonnées** (GPS compris). Clés de stockage aléatoires, fichiers servis uniquement via des URL signées HMAC à durée limitée (stables par tranche d'heure pour le cache). Une photo minimum est conservée.
- **Matching** : une décision par paire (clé primaire) ; le match est créé dans la même transaction que le like réciproque, sous verrou consultatif par paire (deux likes simultanés → un seul match). Les profils déjà traités, bloqués ou suspendus ne réapparaissent pas. L'algorithme de classement est une expression SQL unique (`matching/repository.go`) facile à faire évoluer.
- **Match = conversation** : pas de table de conversations séparée ; `conversation_participants` porte l'état lu et l'effacement local. Un non-membre reçoit 404 (pas 403) pour ne rien révéler.
- **Temps réel** : WebSocket authentifié par le premier message (le token n'apparaît jamais dans une URL), reconnexion avec backoff et resynchronisation REST. Les notifications push natives (APNs/FCM via Expo) ne sont pas incluses car elles exigent des identifiants EAS propres au projet ; l'application notifie en temps réel lorsqu'elle est ouverte et via le centre d'activité.
- **Sessions** : access token HS256 de 15 min (claims stricts, expiration obligatoire) + refresh token opaque (haché en base) à usage unique ; la session est vérifiée à chaque requête, donc déconnexion, réinitialisation de mot de passe et suppression de compte révoquent l'accès immédiatement. Jetons stockés dans le Keychain/Keystore (SecureStore) ; sur le web (cible de dev/test) en `sessionStorage`.
- **Récupération de compte** : code à 6 chiffres valable 30 min, à usage unique, 5 essais maximum ; la demande ne révèle jamais si l'e-mail existe. L'inscription indique en revanche qu'un e-mail est déjà utilisé (compromis UX habituel, atténué par le rate limiting).
- **Modération** : un signalement bloque automatiquement la personne pour l'auteur ; à 3 auteurs distincts, le compte est retiré de la découverte (`users.suspended_at`) en attente de revue. Les signalements survivent à la suppression des comptes (`ON DELETE SET NULL`). Revue manuelle par SQL, par exemple :
  `SELECT reported_id, reason, details, created_at FROM reports WHERE status = 'open' ORDER BY created_at;`
- **Suppression de compte** : mot de passe requis ; suppression en cascade de toutes les données personnelles et des fichiers photo, fermeture des connexions temps réel.
- **Limites connues / évolutions** : hub temps réel et rate limiter en mémoire (une instance) — passer à Redis ou PostgreSQL `LISTEN/NOTIFY` pour plusieurs instances ; pas de vérification d'e-mail à l'inscription ni d'interface d'administration ; pas de notifications push natives.

## Contribuer

Voir [CONTRIBUTING.md](CONTRIBUTING.md), [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) et [CHANGELOG.md](CHANGELOG.md).
