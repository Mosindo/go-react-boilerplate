# Seren — application de rencontre 100 % gratuite

Seren est une application de rencontre complète (mobile iOS/Android avec Expo, et web) adossée à une API Go. Elle couvre tout le parcours : création de compte, profil et photos, découverte de profils, likes, matchs réciproques, messagerie en temps réel, notifications, préférences, confidentialité, blocage, signalement et suppression de compte.

**Gratuit, sans compromis** : aucun abonnement, aucun paywall, aucun « boost » payant, aucun quota de likes, aucune publicité. Les seules limites existantes sont des protections anti-abus (rate limiting) bien au-dessus d'un usage humain normal.

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Installation rapide (Docker)](#installation-rapide-docker)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données et migrations](#base-de-données-et-migrations)
7. [Données de démonstration (seed)](#données-de-démonstration-seed)
8. [Lancement local sans Docker](#lancement-local-sans-docker)
9. [Application mobile](#application-mobile)
10. [Tests et qualité](#tests-et-qualité)
11. [Build et déploiement](#build-et-déploiement)
12. [Référence de l'API](#référence-de-lapi)
13. [Sécurité et vie privée](#sécurité-et-vie-privée)
14. [Décisions techniques](#décisions-techniques)
15. [Limites connues et pistes](#limites-connues-et-pistes)

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription (confirmation 18+), connexion, sessions avec rotation du refresh token, récupération de mot de passe par code e-mail, déconnexion, suppression définitive du compte |
| Profil | Prénom, date de naissance (seul l'âge est affiché, 18 ans minimum vérifié côté serveur), genre, bio, métier, ville, centres d'intérêt (catalogue de 30, 10 max) |
| Photos | 1 à 6 photos, compression sur l'appareil, ré-encodage serveur (métadonnées EXIF/GPS supprimées), photo principale, réorganisation, remplacement, suppression |
| Préférences | Genres recherchés, tranche d'âge, distance maximale |
| Localisation | Position demandée uniquement à l'action de l'utilisateur, arrondie à ~1 km avant envoi **et** avant stockage, distance affichée approximative, suppression possible |
| Découverte | Pile de cartes (glisser ou boutons), carrousel de photos, profil complet, filtres réciproques (genre, âge, distance), classement par activité, points communs et proximité |
| Matching | Like / passer, match uniquement si réciproque, aucun doublon (contrainte unique + verrou par paire), profils déjà traités jamais reproposés, annulation de match |
| Chat | Conversations après match uniquement, messages texte, temps réel (WebSocket), heure d'envoi, statut « Vu », compteurs non lus, pagination, suppression locale d'une conversation |
| Notifications | In-app : nouveau match, nouveaux messages (regroupés par conversation), badge, tout marquer comme lu, push en temps réel |
| Sécurité | Blocage (dissout le match), signalement avec capture de preuves + blocage automatique, liste des profils bloqués, profil masquable, distance masquable |
| UX | Identité visuelle originale (ivoire chaud, encre, terracotta), mode sombre automatique, états de chargement / vide / erreur, toasts, dialogues de confirmation, libellés d'accessibilité |

## Stack technique

| Couche | Choix |
| --- | --- |
| API | Go 1.24, Gin, PostgreSQL 16, `pgxpool`, SQL explicite (sans ORM), JWT HS256, bcrypt, gorilla/websocket, `golang.org/x/image` |
| Mobile / web | Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 7, TanStack Query 5, expo-image, expo-image-picker, expo-image-manipulator, expo-location, expo-secure-store |
| Infra | Docker (image non-root), Docker Compose, GitHub Actions |
| Qualité | `gofmt`, `go vet`, tests Go (unitaires + intégration HTTP), ESLint (config Expo), Prettier, tests unitaires Node, smoke test API, test E2E Playwright sur la version web |

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS, Android, web)
│  ├─ App.tsx                   Providers (Query, thème, safe area, feedback, auth)
│  ├─ src/api/                  Client HTTP typé + endpoints (aucun fetch dans l'UI)
│  ├─ src/hooks/                Session, cache React Query, temps réel
│  ├─ src/navigation/           Auth → Onboarding → Onglets + pile
│  ├─ src/screens/              Écrans par domaine
│  ├─ src/features/             Formulaires et blocs réutilisés (photos, préférences…)
│  ├─ src/ui/ · src/theme/      Design system (clair/sombre)
│  ├─ tests/                    Tests unitaires (node --test)
│  └─ e2e/                      Smoke API + parcours Playwright
├─ services/api/
│  ├─ cmd/api/                  Point d'entrée + tests d'intégration HTTP
│  ├─ cmd/seed/                 Données de démonstration
│  └─ internal/
│     ├─ platform/              Transverse : config, db/migrations, middleware, tokens,
│     │                         rate limit, stockage, URLs signées, mailer, hub temps réel
│     └─ features/              auth, profiles, photos, matching, chat, safety,
│                               notifications, realtime
├─ docker-compose.yml
└─ .env.example
```

Chaque feature backend suit `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`) : handlers fins, règles métier dans les services, SQL uniquement dans les repositories. Détails dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation rapide (Docker)

Prérequis : Docker et Docker Compose, Node.js 22+ pour l'application.

```bash
cp .env.example .env
# Génère un secret et colle-le dans JWT_SECRET :
openssl rand -hex 32
docker compose up --build -d
curl http://localhost:18080/health      # {"status":"ok"}
```

Facultatif — remplir l'application avec des profils de démonstration :

```bash
docker compose exec api ./seed -lat 48.8566 -lng 2.3522
```

Puis lancer l'application (voir [Application mobile](#application-mobile)).

## Variables d'environnement

Toutes les variables sont documentées dans [.env.example](.env.example). Aucun secret n'est versionné.

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | oui | — | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées). Sert aussi à dériver les clés des URLs de médias et des codes de réinitialisation |
| `DATABASE_URL` | oui (hors Compose) | — | Connexion PostgreSQL |
| `APP_ENV` | non | `development` | `development`, `test` ou `production` |
| `PORT` | non | `8080` | Port d'écoute de l'API |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS + WebSocket), séparées par des virgules |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies dont `X-Forwarded-For` est fiable (rate limiting par IP) |
| `UPLOAD_DIR` | non | `./data/uploads` | Dossier de stockage des photos |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | — | Envoi des codes de réinitialisation. Sans SMTP : en développement le code est écrit dans les logs de l'API ; en production l'envoi échoue explicitement |
| `RATE_LIMIT_DISABLED` | non | `false` | Réservé aux tests automatisés, refusé en production |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | non | `app` | Identifiants Postgres utilisés par Compose |
| `EXPO_PUBLIC_API_URL` | mobile | émulateurs : `10.0.2.2` / `localhost` | URL publique de l'API intégrée au bundle mobile (jamais de secret ici) |

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées **automatiquement au démarrage** de l'API (et du seed). Chaque migration est enregistrée dans `schema_migrations`, exécutée dans sa propre transaction, et l'ensemble est sérialisé par un verrou consultatif Postgres : plusieurs instances peuvent démarrer en même temps sans risque.

| Table | Rôle | Contraintes / index principaux |
| --- | --- | --- |
| `users`, `sessions`, `password_resets` | Comptes, refresh tokens (hachés), codes de réinitialisation (HMAC) | e-mail unique ; hash de token unique |
| `profiles`, `preferences` | Profil public, critères de recherche | âge ≥ 18 côté service, CHECK sur genre/longueurs/coordonnées ; index partiels « découvrable » |
| `interests`, `user_interests` | Catalogue et choix | PK composite |
| `photos` | Métadonnées des photos (fichiers sur disque) | `(user_id, position)` unique et différable, 6 max |
| `swipes` | Likes et passes | PK `(swiper_id, target_id)` : impossible de liker deux fois |
| `matches`, `match_members` | Paire canonique (`user_low_id < user_high_id`) et état par membre (lu, masqué) | `UNIQUE(user_low_id, user_high_id)` : aucun doublon |
| `messages` | Messages | index `(match_id, created_at DESC)` |
| `blocks`, `reports` | Modération | rapports conservés (ids anonymisés) si un compte est supprimé |
| `notifications` | In-app | index `(user_id, created_at)` ; dédoublonnage des non lus par conversation |

Toutes les données d'un utilisateur sont supprimées en cascade avec son compte, y compris les fichiers photos.

> Bases créées avec l'ancien boilerplate : la première migration supprime les tables SaaS obsolètes (posts, billing, organisations…). Pour repartir de zéro en local : `docker compose down -v`.

## Données de démonstration (seed)

`cmd/seed` crée des profils fictifs **clairement séparés des vrais membres** :

- e-mails sur le domaine réservé `demo.seren.invalid` (ne peut jamais exister) ;
- colonne `profiles.is_demo = true` et bio préfixée « Profil de démonstration » ;
- photos abstraites générées (aucun visage réel) ;
- refus d'exécution si `APP_ENV=production`.

```bash
cd services/api
DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable go run ./cmd/seed            # 24 profils autour de Paris
go run ./cmd/seed -lat 45.76 -lng 4.83 -count 30                                                 # ailleurs
go run ./cmd/seed -like-email moi@exemple.fr                                                     # ces profils te likent déjà → match immédiat
go run ./cmd/seed -reset                                                                         # supprime tous les profils de démo
```

Mot de passe des comptes de démo : `DemoPassword1` (ex. `demo01@demo.seren.invalid`), pratique pour répondre depuis un second appareil.

## Lancement local sans Docker

```bash
# PostgreSQL 16 local avec une base "app"
cd services/api
export DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export ALLOWED_ORIGINS='http://localhost:8081'
go run ./cmd/api                     # écoute sur :8080 (PORT pour changer)
```

## Application mobile

```bash
cd apps/mobile
npm ci
npm run start          # Expo : QR code pour Expo Go, ou touche a / i / w
npm run web            # version navigateur
```

- **Téléphone physique** : n'utilise jamais `localhost`. Lance `npm run start:phone` (détecte l'IP LAN) ou définis `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`.
- **Émulateur Android** : `http://10.0.2.2:18080` est utilisé par défaut en développement ; **simulateur iOS** : `http://localhost:18080`.
- Vérifie qu'aucun autre service n'occupe le port : `docker compose ps`, `curl http://<IP_LAN>:18080/health`. En cas de conflit, change le port hôte dans `docker-compose.yml` et aligne `EXPO_PUBLIC_API_URL`.
- Pour la version web, ajoute son origine (`http://localhost:8081`) à `ALLOWED_ORIGINS`.

Parcours : Bienvenue → inscription → onboarding en 5 étapes (infos, préférences, centres d'intérêt, position, photos) → Découvrir / Matchs / Activité / Profil.

## Tests et qualité

```bash
# API — unitaires + intégration HTTP complète (Postgres requis pour l'intégration)
cd services/api
gofmt -l .                        # doit être vide
go vet ./...
DATABASE_URL_TEST='postgres://app:app@localhost:5432/app_test?sslmode=disable' go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm run typecheck && npm run lint && npm run format:check && npm test
npm run build:web

# Bout en bout (API lancée sur :18080)
npm run e2e:smoke                                   # parcours API complet
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
python3 -m http.server 8099 --directory dist &      # ALLOWED_ORIGINS doit contenir http://localhost:8099
npx playwright install chromium && npm run e2e:web  # parcours UI, captures dans e2e/.results
```

Sans `DATABASE_URL_TEST`, les tests d'intégration sont ignorés (les tests unitaires tournent toujours).

Couverture des tests d'intégration (`services/api/cmd/api/*_test.go`) : santé ; inscription/connexion/refresh/déconnexion avec révocation immédiate ; validation de profil (mineur refusé, genre, longueurs) ; arrondi des coordonnées ; uploads dangereux (SVG, script, image trop petite) ; limite de 6 photos, réorganisation, remplacement, suppression ; URLs signées et falsifiées ; découverte avec filtres réciproques et sans fuite de données privées ; like/pass, doublons refusés, match unique même en **likes simultanés** ; temps réel (WebSocket) ; permissions du chat pour un tiers ; pagination ; accusés de lecture ; notifications regroupées ; masquage local ; blocage ; signalement idempotent avec blocage automatique ; visibilité ; réinitialisation de mot de passe (code faux, réutilisation, révocation des sessions) ; suppression de compte (données et fichiers) ; rate limiting.

La CI GitHub Actions (`.github/workflows/ci.yml`) exécute tout cela : API, mobile, E2E (smoke + Playwright) et image Docker.

## Build et déploiement

**API**

```bash
docker build -t seren-api ./services/api
docker run -p 8080:8080 \
  -e APP_ENV=production -e DATABASE_URL=... -e JWT_SECRET=... \
  -e ALLOWED_ORIGINS=https://app.exemple.fr -e TRUSTED_PROXIES=10.0.0.0/8 \
  -e SMTP_HOST=... -e SMTP_FROM=no-reply@exemple.fr \
  -v seren-uploads:/data/uploads seren-api
```

- Mettre l'API derrière un reverse proxy HTTPS (le header HSTS est émis quand la requête arrive en HTTPS) qui supporte les WebSockets (`/realtime`).
- Monter un volume persistant sur `/data/uploads` (ou remplacer `platform/storage` par un stockage objet).
- L'image tourne en utilisateur non-root et inclut aussi le binaire `seed` (inutile en production).

**Application**

```bash
cd apps/mobile
EXPO_PUBLIC_API_URL=https://api.exemple.fr npx eas build --platform all   # binaires iOS/Android (compte Expo requis)
EXPO_PUBLIC_API_URL=https://api.exemple.fr npm run build:web               # site statique dans dist/
```

## Référence de l'API

Toutes les routes, sauf mention contraire, exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "message lisible", "code": "code_stable"}`.

| Méthode | Route | Description |
| --- | --- | --- |
| GET | `/health` | Santé (public) |
| POST | `/auth/register`, `/auth/login` | Public, renvoie `accessToken` (15 min) + `refreshToken` (30 j) |
| POST | `/auth/refresh`, `/auth/logout` | Rotation / révocation du refresh token |
| POST | `/auth/password/forgot`, `/auth/password/reset` | Code à 6 chiffres (15 min, 5 essais), réponse identique que l'e-mail existe ou non |
| GET / DELETE | `/me` | Compte courant / suppression (mot de passe requis) |
| GET | `/interests` | Catalogue |
| GET | `/me/profile` | Profil complet de l'utilisateur, `isComplete` |
| PUT | `/me/profile`, `/me/preferences`, `/me/interests`, `/me/visibility` | Mises à jour |
| PUT / DELETE | `/me/location` | Position (arrondie) / suppression |
| POST | `/me/photos` | Upload multipart (`photo`) |
| PUT / DELETE | `/me/photos/:id` | Remplacement / suppression |
| PUT | `/me/photos/order` | Ordre (`photoIds`, le premier = principale) |
| GET | `/media/:key?exp&sig` | Photo via URL signée expirante (public, signature requise) |
| GET | `/profiles/:userId` | Profil public (si non bloqué et découvrable ou matché) |
| GET | `/discover?limit=` | Profils proposés |
| POST | `/swipes` | `{targetUserId, direction: like\|pass}` → `{matched, match?}` |
| DELETE | `/matches/:matchId` | Annuler un match |
| GET | `/conversations?cursor=` | Conversations (pagination par curseur) |
| GET / POST | `/conversations/:matchId/messages` | Lire (curseur) / envoyer |
| POST | `/conversations/:matchId/read` | Marquer comme lu |
| DELETE | `/conversations/:matchId` | Masquer localement |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId` | Blocages |
| POST | `/reports` | Signalement (+ blocage automatique) |
| GET / POST | `/notifications`, `/notifications/:id/read`, `/notifications/read-all` | Notifications |
| POST | `/realtime/ticket` | Ticket de 60 s pour ouvrir `GET /realtime?ticket=` (WebSocket) |

Événements temps réel : `message.created`, `conversation.read`, `match.created`, `match.removed`, `notification.created`.

## Sécurité et vie privée

- **Authentification** : JWT HS256 avec vérification stricte (`alg`, type de jeton, expiration obligatoire), access tokens courts, refresh tokens aléatoires stockés hachés et tournants, **session vérifiée à chaque requête** (déconnexion, réinitialisation de mot de passe et suppression de compte effectives immédiatement). Les tickets WebSocket ne sont pas acceptés comme access tokens et inversement.
- **Mots de passe** : bcrypt (coût 12), 8–72 caractères avec lettre et chiffre, comparaison à temps constant même pour un e-mail inconnu (pas d'énumération).
- **Autorisations côté serveur** : chaque requête de chat vérifie l'appartenance au match (404 sinon), les profils ne sont visibles que s'ils ne sont pas bloqués et sont découvrables ou matchés, les règles de découverte et de match sont entièrement calculées par le serveur.
- **Données exposées** : jamais d'e-mail, de date de naissance ni de coordonnées dans les profils publics ; distance arrondie (1 km, puis 5 km au-delà de 20 km) et masquable.
- **Uploads** : type détecté par le contenu (JPEG/PNG/WebP uniquement), 10 Mo max, protection contre les bombes de décompression, ré-encodage JPEG (supprime EXIF, GPS et toute charge cachée), noms de fichiers aléatoires générés par le serveur, URLs signées et expirantes, fichiers inaccessibles dès la suppression.
- **Abus** : rate limiting par IP (identifiants) et par utilisateur (likes, messages, uploads, signalements), limites de taille de requête, en-têtes de sécurité, CORS restrictif.
- **Modération** : les signalements conservent un instantané (profil + 50 derniers messages) même si un compte est supprimé, et bloquent immédiatement.

## Décisions techniques

- **Construire sur le boilerplate existant** : l'architecture Go feature-first et le client Expo étaient sains ; les modules SaaS incompatibles avec un produit gratuit (Stripe, organisations, posts, annuaire d'utilisateurs, messagerie ouverte à tous) ont été retirés plutôt que contournés.
- **Conversation = match** : un message n'existe que dans un match, ce qui rend l'autorisation du chat simple et sûre ; l'état par personne (lu, masqué) vit dans `match_members`.
- **Matching sans course** : verrou consultatif par paire + contrainte unique ; deux likes simultanés produisent exactement un match (test dédié).
- **Recommandation extensible** : le SQL filtre (règles strictes et réciproques), un `Ranker` en Go ordonne (activité, intérêts communs, proximité, a déjà liké). Remplacer `DefaultRanker` suffit pour expérimenter.
- **Temps réel** : WebSocket natif (aucune dépendance côté app), hub en mémoire, tickets courts pour ne jamais mettre l'access token dans une URL ; repli automatique en polling si la connexion tombe.
- **Photos** : stockage disque derrière une interface (`platform/storage`) et URLs signées alignées sur des fenêtres de 6 h pour un cache image efficace.
- **Récupération de compte par code** plutôt que par lien : plus simple sur mobile, sans deep link ; codes HMAC, 15 min, 5 essais.
- **Notifications in-app** avec regroupement des messages par conversation, poussées en temps réel. Les notifications push système nécessitent un projet EAS et ne sont pas activées (voir plus bas).
- **UI sans librairie de composants** : Tamagui (quasi inutilisé) a été retiré au profit d'un petit design system typé, thème clair/sombre automatique.

## Limites connues et pistes

- **Mise à l'échelle horizontale** : le hub temps réel et le rate limiter sont en mémoire. Pour plusieurs instances : bus partagé (Postgres `LISTEN/NOTIFY` ou Redis) derrière `realtime.Publisher` et rate limiting au niveau du proxy.
- **Notifications push** (Expo Notifications) : à brancher sur `notifications.Service.Notify` une fois un projet EAS configuré.
- **Back-office de modération** : les signalements sont stockés (`reports`, statut `open`) mais aucune interface d'administration n'est fournie.
- **Vérification d'identité / d'âge** : l'âge est déclaratif (date de naissance + confirmation 18+), contrôlé côté serveur.
- **Stockage objet** : remplacer `storage.Local` par une implémentation S3/GCS pour un déploiement multi-instance.

## Documentation complémentaire

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CHANGELOG.md](CHANGELOG.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
