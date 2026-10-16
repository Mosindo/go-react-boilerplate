# Lien — une application de rencontre 100 % gratuite

**Lien** est une application de rencontre complète : créer un compte, composer son profil et ses photos, découvrir des personnes près de chez soi, liker ou passer, matcher, discuter en temps réel, gérer sa confidentialité, bloquer, signaler et supprimer son compte.

**Gratuité garantie par conception** : aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de likes, aucune fonctionnalité verrouillée, aucune publicité. Même « qui m'a liké » est visible par tout le monde. Il n'existe aucun code de paiement dans le dépôt.

Le dépôt est un monorepo : une API Go, une base PostgreSQL et une application mobile Expo (React Native, qui s'exécute aussi dans un navigateur pour le développement et les tests).

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Installation et lancement local](#installation-et-lancement-local)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données, migrations, seed](#base-de-données-migrations-seed)
7. [Tests et qualité](#tests-et-qualité)
8. [Build et déploiement](#build-et-déploiement)
9. [Modération](#modération)
10. [Référence de l'API](#référence-de-lapi)
11. [Sécurité et vie privée](#sécurité-et-vie-privée)
12. [Décisions techniques et produit](#décisions-techniques-et-produit)
13. [Limites connues](#limites-connues)

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription, connexion, session renouvelable (refresh token rotatif), récupération de mot de passe par code e-mail, déconnexion, **suppression de compte** (mot de passe requis, effacement en cascade des données et des fichiers) |
| Profil | Prénom, âge (date de naissance **immuable** et majorité vérifiée côté serveur), genre, bio, profession, ce que l'on cherche, ville, centres d'intérêt (32 au catalogue, 10 max) |
| Photos | Jusqu'à 6 photos : validation par contenu (JPEG/PNG), 8 Mo max, anti-« decompression bomb », rotation EXIF appliquée, **ré-encodage JPEG qui supprime toutes les métadonnées (GPS…)**, ordre, photo principale, remplacement, suppression |
| Préférences | Genres recherchés, tranche d'âge, distance maximale — appliqués des **deux côtés** (je dois correspondre aux critères de l'autre aussi) |
| Découverte | Cartes à swiper (geste + boutons accessibles), carrousel de photos, profil complet, ordre de proposition : ceux qui m'ont déjà liké → centres d'intérêt communs → proximité → activité récente |
| Matching | Like réciproque = match, **un seul match possible par paire** même si les deux likent au même instant, jamais de profil déjà traité qui revient, annulation de match |
| Chat | Réservé aux matchs, messages texte, temps réel (WebSocket), heure, **non lu / lu** (accusé de lecture), liste des conversations avec compteur, suppression locale d'une conversation, historique paginé |
| Notifications | Match et nouveau message, persistées + poussées en direct, badge, « tout lire » |
| Sécurité | Blocage (supprime le match et la conversation, invisibilité mutuelle), signalement (6 motifs), limites de débit, suspension d'un compte par le CLI de modération |
| Vie privée | Masquer son profil de la découverte, masquer sa distance, position jamais exposée (distance approximative uniquement) |

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool` (SQL explicite, sans ORM), JWT HS256, bcrypt, WebSocket (`gorilla/websocket`)
- **Mobile** : Expo SDK 54, React Native 0.81, TypeScript strict, React Navigation, TanStack Query, `expo-image-picker`, `expo-location`, `expo-secure-store`
- **Infra** : Docker, Docker Compose, GitHub Actions
- **Tests** : `go test` (unitaires + intégration sur vrai PostgreSQL), Vitest (mobile), script de parcours API en Node, flux Maestro, **parcours navigateur Playwright** (inscription → onboarding → découverte → like → match → chat en direct)

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS, Android, web)
│  ├─ App.tsx                   Racine : providers, bascule connexion / onboarding / app
│  ├─ src/api/                  Client HTTP (refresh de token, timeouts), endpoints, types, WebSocket
│  ├─ src/hooks/                Session, requêtes React Query, temps réel, deck de swipe
│  ├─ src/screens/              Écrans (Auth, Onboarding, Découvrir, Matchs, Messages, Conversation…)
│  ├─ src/components/           Composants métier réutilisables (SwipeDeck, PhotoManager, …)
│  ├─ src/shared/{ui,layout,feedback}/  Design system et états vides / erreur / chargement
│  ├─ src/lib/                  Logique pure testée (validation, formats, file de swipe, conversation)
│  ├─ e2e/ scripts/             Parcours navigateur, parcours API, flux Maestro
├─ services/api/
│  ├─ cmd/api                   Serveur HTTP (racine de composition)
│  ├─ cmd/seed                  Comptes de démonstration (jamais en production)
│  ├─ cmd/moderate              Outil d'exploitation : signalements, suspension
│  └─ internal/
│     ├─ platform/              config, db (+ migrations), middleware, httpx, storage, mediaurl, mailer, realtime, logger
│     └─ features/              auth, profiles, photos, discover, matching, chat, safety, notifications, realtime, account
├─ docker-compose.yml           PostgreSQL + API (+ volume des photos)
└─ .github/workflows/           CI (Go, mobile, bundle, Docker, parcours navigateur) et QA lite
```

Règles d'architecture (reprises de `AGENTS.md`) : `handler → service → repository`, pas de SQL dans les handlers, un seul serveur HTTP, une seule pool de connexions, préoccupations transverses dans `platform`, domaine dans `features`. Les features ne s'importent pas entre elles : elles dépendent de petites **interfaces** définies côté consommateur (`ProfileSource`, `Events`…), branchées dans `cmd/api` (voir `events.go` pour le pont notifications + temps réel).

**Modèle de données** (migrations `014`–`017`) : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `conversation_states`, `messages`, `blocks`, `reports`, `notifications`. Contraintes `CHECK`/`UNIQUE`/clés étrangères `ON DELETE CASCADE` partout, index sur chaque chemin chaud (voir `internal/platform/db/migrations/015_dating_schema.sql`).

## Installation et lancement local

Prérequis : Docker (ou PostgreSQL 16), Go 1.24+, Node.js 20.19+ (22 recommandé), npm.

### 1. Configurer

```bash
cp .env.example .env
# Générez un vrai secret JWT (32 caractères minimum) et collez-le dans .env :
openssl rand -base64 48
```

### 2. Démarrer PostgreSQL + l'API

```bash
export JWT_SECRET="$(openssl rand -base64 48)"
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

L'API écoute sur `http://localhost:18080`. Les migrations s'appliquent toutes seules au démarrage. Les photos sont stockées dans le volume `uploads_data`.

Sans Docker pour l'API :

```bash
docker compose up -d postgres
cd services/api
export DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
go run ./cmd/api
```

### 3. (Optionnel) Charger des données de démonstration

```bash
cd services/api
go run ./cmd/seed          # 12 profils de démo, mot de passe DemoPass123!
go run ./cmd/seed -reset   # les supprime (comptes + fichiers)
```

Les comptes de démo utilisent le domaine réservé `demo.lien.test` (ex. `camille@demo.lien.test`), des portraits **générés** (dégradés) et le seed **refuse de tourner avec `APP_ENV=production`**.

### 4. Lancer l'application

```bash
cd apps/mobile
npm install
```

**Navigateur** (le plus rapide pour essayer) :

```bash
EXPO_PUBLIC_API_URL=http://localhost:18080 npx expo start --web
# Pour le web, autorisez l'origine côté API : ALLOWED_ORIGINS=http://localhost:8081
```

**Téléphone réel** — jamais `localhost` : utilisez l'IP locale de votre machine.

```bash
npm run start:phone     # détecte l'IP du réseau local, définit EXPO_PUBLIC_API_URL et lance Expo
# ou manuellement : EXPO_PUBLIC_API_URL=http://192.168.1.10:18080 npx expo start
```

Vérifications recommandées avant de dire « prêt » (voir `AGENTS.md`) : `docker compose ps`, `curl http://<IP>:18080/health` depuis le téléphone, aucun autre service sur le port. En cas de conflit de port, changez `18080:8080` dans `docker-compose.yml` et alignez `EXPO_PUBLIC_API_URL`.

Émulateur Android : `10.0.2.2` remplace `localhost` (valeur par défaut en développement).

## Variables d'environnement

Modèle complet : [`.env.example`](.env.example). Aucun secret n'est jamais dans le code ni dans l'application mobile (seule `EXPO_PUBLIC_API_URL`, publique, y figure).

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `DATABASE_URL` | oui | — | URL PostgreSQL |
| `JWT_SECRET` | oui | — | ≥ 32 caractères, refusé s'il ressemble à un exemple. Sert aussi à dériver la clé de signature des URLs de photos |
| `APP_ENV` | non | `development` | `development`, `test` ou `production`. En production : SMTP obligatoire, Gin en mode release |
| `PORT` | non | `8080` | Port d'écoute |
| `APP_NAME` | non | `Lien` | Nom utilisé dans les e-mails |
| `UPLOAD_DIR` | non | `./data/uploads` | Dossier des photos (volume persistant en production) |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS + WebSocket). Inutile pour les apps natives |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse-proxies autorisés à fixer `X-Forwarded-For`. **À renseigner derrière un proxy**, sinon toutes les requêtes semblent venir du proxy et partagent leurs limites de débit |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | prod | vide | E-mails de récupération. Vide en développement : les codes sont **écrits dans les logs de l'API** |
| `POSTGRES_PASSWORD` | non | `app` | Mot de passe PostgreSQL du `docker-compose.yml` (à changer hors local) |
| `EXPO_PUBLIC_API_URL` | mobile | `localhost:18080` / `10.0.2.2:18080` en dev | URL de l'API vue par l'app |

## Base de données, migrations, seed

- Les migrations SQL sont **embarquées dans le binaire** (`services/api/internal/platform/db/migrations`) et appliquées au démarrage, dans l'ordre, chacune dans sa transaction, protégées par un verrou consultatif (plusieurs instances peuvent démarrer ensemble). La table `schema_migrations` mémorise ce qui est appliqué.
- Pour ajouter une migration : créer `0NN_description.sql` (numéro suivant), ne **jamais** modifier une migration déjà publiée.
- `014` supprime les tables héritées du boilerplate générique (posts, abonnements…) ; les bases existantes convergent ainsi vers le même schéma que les installations neuves.
- Sauvegardes : `pg_dump` de la base **et** sauvegarde du dossier `UPLOAD_DIR` (les photos ne sont pas en base).

## Tests et qualité

```bash
# API : formatage, analyse statique, tests unitaires + intégration, build
cd services/api
gofmt -l .                       # ne doit rien lister
go vet ./...
export DATABASE_URL_TEST='postgres://app:app@localhost:5432/app?sslmode=disable'
go test -race ./...              # sans DATABASE_URL_TEST, les tests d'intégration sont ignorés
go build ./cmd/api ./cmd/seed ./cmd/moderate

# Mobile
cd apps/mobile
npm ci
npm run typecheck && npm run lint && npm run format:check
npm test                         # Vitest
npx expo export --platform android --output-dir /tmp/lien-export   # le bundle compile

# Parcours complets (API lancée sur :18080)
npm run e2e:smoke                # inscription → profil → photos → découverte → match → chat → blocage → suppression
```

Les tests d'intégration créent **un schéma PostgreSQL jetable par test** : ils ne se marchent pas dessus et nettoient tout. Ils couvrent notamment l'authentification, la récupération de mot de passe, la découverte et ses filtres, les likes concurrents, les matchs, le chat et ses permissions, les photos, le blocage, le signalement et la suppression de compte.

**Parcours navigateur** (le vrai parcours utilisateur, dans l'UI) :

```bash
cd apps/mobile
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run export:web
npx http-server dist-web -p 19006 -s &        # l'API doit autoriser http://localhost:19006 (ALLOWED_ORIGINS)
npx playwright install chromium
SHOTS=/tmp/lien-shots npm run e2e:web         # SHOTS = captures d'écran (optionnel) ; utiliser une base vierge
```

Flux Maestro sur appareil/émulateur : `npm run e2e:ui:run` (voir `apps/mobile/scripts/run-maestro.sh`).

## Build et déploiement

**API** : image Docker multi-étapes (`services/api/Dockerfile`), binaire statique, utilisateur non-root, `HEALTHCHECK` sur `/health`, volume `/data/uploads`.

```bash
docker build -t lien-api services/api
docker run -e DATABASE_URL=... -e JWT_SECRET=... -e APP_ENV=production \
  -e SMTP_HOST=... -e SMTP_FROM=... -e TRUSTED_PROXIES=10.0.0.0/8 \
  -v lien_uploads:/data/uploads -p 8080:8080 lien-api
```

Checklist de production :

1. HTTPS obligatoire devant l'API (reverse proxy) ; `TRUSTED_PROXIES` renseigné ; `APP_ENV=production`.
2. `JWT_SECRET` aléatoire et stocké dans un gestionnaire de secrets ; mot de passe PostgreSQL fort ; PostgreSQL non exposé publiquement.
3. SMTP configuré (sans lui, l'API refuse de démarrer en production).
4. Volume persistant sur `UPLOAD_DIR`, sauvegardé.
5. Une seule instance d'API tant que le temps réel et les limites de débit sont en mémoire (voir [Limites connues](#limites-connues)).

**Mobile** : `EXPO_PUBLIC_API_URL` doit pointer vers l'API HTTPS au moment du build, puis `eas build` (iOS/Android). Remplacez `app.lien.mobile` (identifiants de bundle dans `app.json`) et ajoutez vos icônes. Le build web (`npm run export:web`) produit un site statique.

## Modération

Il n'y a volontairement pas de back-office : un petit CLI exploite la même base.

```bash
cd services/api
go run ./cmd/moderate reports                 # signalements ouverts (motif, détails, nombre de signalements du compte)
go run ./cmd/moderate suspend <user-id>       # bloque la connexion, coupe les sessions, masque le profil, supprime les matchs
go run ./cmd/moderate restore <user-id>
go run ./cmd/moderate dismiss <report-id>
```

Dans l'image Docker, le même outil est disponible : `docker compose exec api ./moderate reports`.

Un compte suspendu reçoit `403 account suspended` à la connexion et ne peut plus renouveler sa session. Les signalements pour « personne mineure » doivent être traités en priorité.

## Référence de l'API

Toutes les routes sauf `/health`, `/auth/*`, `/media/photos/:id` (URL signée) et `/realtime` (ticket) exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "message"}`. Listes paginées par `limit`/`offset` bornés (les messages par curseur `before`).

| Méthode | Route | Rôle |
| --- | --- | --- |
| POST | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Session (access token 15 min, refresh token 30 jours rotatif) |
| POST | `/auth/forgot-password`, `/auth/reset-password` | Récupération (toujours `202`, pas d'énumération ; 5 essais par code) |
| GET / DELETE | `/me` | Compte courant / suppression (mot de passe requis) |
| GET / PUT | `/me/profile`, `/me/preferences` | Profil (+ étapes manquantes) / préférences |
| PATCH | `/me/privacy` | `discoverable`, `showDistance` |
| GET | `/interests`, `/profiles/:userId` | Catalogue / profil public (404 si bloqué ou masqué) |
| POST / PUT / DELETE | `/me/photos`, `/me/photos/:id`, `/me/photos/order`, `/me/photos/:id/main` | Gestion des photos |
| GET | `/media/photos/:id?exp&sig` | Image via URL signée (≤ 2 h) |
| GET | `/discover?limit=` | Prochains profils |
| POST | `/swipes` | `{targetId, action: like\|pass}` → `{matched, matchId}` |
| GET / DELETE | `/matches`, `/matches/:id` | Matchs / annulation |
| GET | `/likes/received` | Qui m'a liké (gratuit) |
| GET | `/conversations`, `/conversations/unread-count` | Liste + non lus |
| GET / POST | `/conversations/:id/messages` | Historique paginé / envoi (2000 caractères max) |
| POST / DELETE | `/conversations/:id/read`, `/conversations/:id` | Marquer lu / supprimer pour moi |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId` | Blocages |
| POST | `/reports` | Signalement |
| GET / POST | `/notifications`, `/notifications/:id/read`, `/notifications/read-all` | Notifications |
| POST + GET (WS) | `/realtime/ticket`, `/realtime?ticket=` | Temps réel : `message.new`, `message.read`, `match.new`, `notification.new` |

## Sécurité et vie privée

- **Autorisation côté serveur partout.** Un chat n'est lisible que par les deux membres du match (sinon `404`, indistinguable d'un chat inexistant). Une photo est une ressource privée : jamais de chemin de fichier public, uniquement des URLs signées (HMAC, expiration à l'heure, clé dérivée du secret).
- **JWT** : `HS256` uniquement, expiration obligatoire, claims stricts. Refresh tokens stockés **hachés**, rotatifs, révoqués à la déconnexion, au changement de mot de passe et à la suspension.
- **Mots de passe** : bcrypt, 8 à 72 octets, comparaison factice pour ne pas révéler les e-mails inconnus par le temps de réponse.
- **Données de localisation minimales** : coordonnées arrondies à ~1 km au stockage, jamais renvoyées aux autres ; les autres voient une distance volontairement floue (« moins de 1 km », puis au km près jusqu'à 10 km, puis par paliers de 5 km). Le client n'envoie la position que si l'utilisateur la partage ; sinon il choisit une ville.
- **Entrées** : validation stricte et normalisation (e-mail, texte, listes bornées, pagination, UUID), SQL toujours paramétré.
- **Anti-abus** : limites de débit par IP (authentification) et par utilisateur (écritures, messages, envois, signalements). Elles ne plafonnent pas les likes de façon produit : seulement un seuil anti-script de 240/minute.
- **Uploads** : type décidé sur le contenu, dimensions et poids bornés, ré-encodage (supprime métadonnées et charges ajoutées), nom de fichier généré, entêtes `nosniff` + CSP stricte.
- **Suppression de compte** : une instruction SQL en cascade + suppression des fichiers + fermeture des connexions temps réel.
- **Secrets** : jamais dans le dépôt (`.env` ignoré), jamais dans l'app ; les logs masquent jetons et mots de passe.

## Décisions techniques et produit

- **Reprise du boilerplate** : API Go/Gin/pgx, sessions JWT et structure `handler → service → repository` conservées. Retirés : Stripe et abonnements (contraire au produit), posts/commentaires, multi-tenant « organisations », chat direct sans match, Tamagui (inutilisé).
- **Un match est aussi la conversation** : les messages pointent `matches.id`. Moins de tables, impossible d'écrire sans match, annuler/bloquer supprime tout d'un coup.
- **Paire canonique `user_a < user_b` + verrou consultatif par paire** pour garantir un seul match même sous likes simultanés.
- **Le swipe est la mémoire** : la table `swipes` empêche tout profil traité de revenir ; changer un pass en like est permis avant match, pas l'inverse après un match (il faut annuler le match).
- **Classement isolé** dans `internal/features/discover/ranking.go` pour faire évoluer l'algorithme sans toucher aux filtres.
- **Âge** : date de naissance obligatoire, ≥ 18 ans, **non modifiable** (sinon la vérification serait contournable).
- **Genres** : femme, homme, non-binaire ; on choisit un ou plusieurs genres recherchés ; l'intérêt doit être mutuel pour apparaître en découverte.
- **Position** : coordonnées arrondies ; filtrage par bande de latitude indexée puis formule de haversine en SQL (sans PostGIS, plus simple à déployer).
- **Temps réel par WebSocket avec ticket à usage unique** (le jeton d'accès ne circule jamais dans une URL) ; repli automatique sur interrogation périodique si la socket tombe ; resynchronisation à chaque reconnexion.
- **Notifications message regroupées** : une seule notification non lue par conversation, mise à jour à chaque nouveau message.
- **« Supprimer la conversation »** masque l'historique **pour soi seul** (l'autre garde le sien).
- **Interface en français**, thème clair unique (chaleureux, crème et framboise). Pas de mode sombre pour l'instant : le design system est centralisé dans `apps/mobile/src/shared/ui/tokens.ts`, il se prête à une variante sombre.
- **Pas de back-office** : un CLI (`cmd/moderate`) suffit à ce stade et se contrôle facilement.

## Limites connues

- **Notifications push natives (FCM/APNs) non incluses** : elles exigent des identifiants propres à votre compte développeur. Les notifications sont persistées, poussées en direct (WebSocket) et affichées dans l'app (badge, onglet Alertes).
- **Une seule instance d'API** : le hub temps réel, les tickets WebSocket et les compteurs de débit sont en mémoire. Pour monter en charge, brancher Redis (ou `LISTEN/NOTIFY`) derrière les interfaces `realtime.Hub` et `middleware.Limiter`.
- **Stockage des photos sur disque local** : l'interface `storage.Store` permet d'ajouter un backend S3 sans toucher au reste.
- **Révocation de session** : chaque requête vérifie que la session n'est pas révoquée ni le compte suspendu (cache positif de 10 s) ; les sockets WebSocket déjà ouvertes d'un compte suspendu persistent jusqu'à leur fermeture (5 connexions max par utilisateur).
- **Triangulation de position** : la position enregistrée ne peut changer qu'une fois par 24 h ; un attaquant patient garde néanmoins un oracle de distance à ~1 km.
- **Preuves de signalement** : le signalement conserve les 20 derniers messages **s'il est fait avant** que l'auteur bloque ou annule le match.
- **Pas de limite de tentatives par compte** (uniquement par IP) sur la connexion.
- **Inscription** : une adresse déjà utilisée est signalée (`409`) — compromis UX, protégé par limite de débit.
- **Web** : la session n'y est pas persistée (pas de stockage sécurisé navigateur) ; la cible web sert au développement et aux tests.
- **Pas de mode sombre ni d'internationalisation** (les textes sont en français dans les écrans).
