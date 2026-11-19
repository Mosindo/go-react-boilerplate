# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre complète (mobile iOS/Android + web) : inscription, profil avec photos, découverte par swipe, likes, matchs réciproques, messagerie temps réel, notifications, préférences, confidentialité, blocage, signalement et suppression de compte.

**Tout est gratuit, pour tout le monde** : likes illimités, « qui m'a liké » visible, aucune option payante, aucun abonnement, aucune publicité. Le code ne contient aucun module de paiement.

---

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

---

## Fonctionnalités

| Domaine | Ce qui est implémenté |
| --- | --- |
| Compte | Inscription (e-mail + mot de passe, consentement 18+), connexion, sessions avec rotation des refresh tokens, récupération par code à 6 chiffres envoyé par e-mail, déconnexion, suppression définitive du compte |
| Onboarding | Prénom, date de naissance (18 ans minimum, non modifiable ensuite), genre, préférences, intention (« relation sérieuse », etc.), photos, bio, métier, centres d'intérêt, localisation approximative facultative |
| Photos | 1 à 6 photos, compression sur l'appareil, ré-encodage serveur (EXIF/GPS supprimés), photo principale, réordonnancement, remplacement, suppression, URLs signées expirantes |
| Découverte | Pile de cartes (swipe ou boutons), carrousel photo, profil complet, filtres mutuels genre/âge/distance, distance approximative uniquement, classement par signaux (intérêts communs, activité, proximité, « vous a liké ») |
| Matching | Like / passer, match uniquement si réciproque, aucun doublon (contraintes + verrou transactionnel), un profil traité ne réapparaît jamais, « Vous ont liké » gratuit, annulation de match |
| Chat | Conversation créée au match, messages texte, temps réel (WebSocket), heure d'envoi, non-lus, accusés de lecture « Lu », pagination, suppression locale de l'historique |
| Notifications | Nouveau match, nouveau message (regroupées par conversation), compteur temps réel, liste paginée |
| Sécurité & modération | Blocage (bidirectionnel, ferme la conversation), signalement avec motif (bloque aussi), personnes bloquées, rate limiting, autorisations vérifiées côté serveur |
| Confidentialité | Mettre son profil en pause, masquer sa distance, supprimer sa position, coordonnées arrondies à ~1 km et jamais exposées |
| UX | Thème clair/sombre (ou système), états de chargement/vides/erreurs, toasts, dialogues de confirmation, accessibilité (rôles, libellés), mise en page responsive (mobile et web) |

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16, pgx/pgxpool (SQL explicite, sans ORM), JWT HS256, bcrypt, WebSocket (`github.com/coder/websocket`), traitement d'image (`golang.org/x/image`).
- **App** : Expo SDK 54 / React Native 0.81 / React 19, TypeScript strict, React Navigation, TanStack Query, expo-image, expo-image-picker / image-manipulator, expo-location, expo-secure-store. Même code pour iOS, Android et web.
- **Outillage** : Docker / Docker Compose, GitHub Actions, ESLint, Prettier, Jest (+ Testing Library), Playwright.

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS, Android, web)
│  ├─ App.tsx                   Providers (thème, cache, session, feedback)
│  ├─ e2e/                      Test E2E navigateur (Playwright)
│  └─ src/
│     ├─ api/                   Client HTTP (refresh auto), endpoints typés, stockage des tokens
│     ├─ state/                 Session, temps réel (WebSocket → cache), célébration de match
│     ├─ features/              Écrans par domaine : auth, onboarding, discovery, likes, chat,
│     │                         notifications, profile, safety
│     ├─ navigation/            Pile auth / onboarding / onglets + écrans empilés
│     ├─ ui/                    Design system (Text, Button, TextField, ActionSheet, Toast…)
│     ├─ theme/                 Palettes clair/sombre, typographie, espacements
│     └─ lib/                   Validation, formatage, photos, localisation
├─ services/api/
│  ├─ cmd/api/                  Point d'entrée HTTP (arrêt propre)
│  ├─ cmd/seed/                 Données de démonstration (+ purge)
│  └─ internal/
│     ├─ app/                   Composition root + tests d'intégration bout en bout
│     ├─ features/              auth, profiles, photos, matching, chat, notifications,
│     │                         safety, realtime — chacun : handler → service → repository
│     └─ platform/              config, db (migrations), middleware (auth, CORS, rate limit),
│                               httpx, geo, media, mailer, realtime (hub), logger
├─ docs/ARCHITECTURE.md         Détails d'architecture, modèle de données, sécurité
├─ docker-compose.yml
└─ .env.example
```

Règles : handlers fins, logique métier dans les services, SQL uniquement dans les repositories, une seule instance HTTP (Gin), un seul pool DB, aucun accès réseau direct depuis les écrans (tout passe par `src/api`). Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation et lancement local

### Prérequis

- Docker + Docker Compose (ou Go 1.24 et PostgreSQL 16 en local)
- Node.js 20.19+ (22 recommandé) et npm
- Pour mobile : l'app Expo Go ou un simulateur/émulateur

### 1. Configurer

```bash
cp .env.example .env
# Générez un secret fort et collez-le dans JWT_SECRET :
openssl rand -hex 32
```

### 2. Démarrer l'API et PostgreSQL

```bash
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

Les migrations s'appliquent automatiquement au démarrage. Sans `SMTP_HOST`, les codes de récupération sont écrits dans les logs (`docker compose logs api`) — uniquement hors production.

Sans Docker :

```bash
cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/api
```

### 3. (Facultatif) Charger des profils de démonstration

```bash
docker compose exec api ./seed                        # 24 profils autour de Paris
docker compose exec api ./seed -likes vous@exemple.fr  # + la moitié vous like déjà
docker compose exec api ./seed -purge                  # supprime toutes les données de démo
```

### 4. Lancer l'application

```bash
cd apps/mobile
npm ci
npm run web        # navigateur (http://localhost:8081)
npm run start      # QR code pour Expo Go / simulateurs
```

Sur un **téléphone physique**, `localhost` désigne le téléphone : utilisez l'IP LAN de votre machine.

```bash
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npm run start
# ou, détection automatique de l'IP :
npm run start:phone
```

Vérifiez qu'aucun service local n'occupe déjà le port 18080 (`docker compose ps`, `lsof -i :18080` / `netstat -ano | findstr :18080`). En cas de conflit, changez le port hôte dans `docker-compose.yml` et `EXPO_PUBLIC_API_URL`.

Pour tester un match rapidement : créez un compte, terminez l'onboarding, lancez `./seed -likes <votre e-mail>`, puis likez des profils dans « Vous ont liké ».

## Variables d'environnement

| Variable | Où | Obligatoire | Description |
| --- | --- | --- | --- |
| `JWT_SECRET` | API | oui | Secret HMAC ≥ 32 caractères, aléatoire (les valeurs « placeholder » sont refusées). Sert aussi à dériver les clés de signature des URLs photo et des codes de récupération. |
| `DATABASE_URL` | API | oui | Chaîne de connexion PostgreSQL. |
| `APP_ENV` | API | non (`development`) | `development`, `test` ou `production`. En production : comptes de démo masqués, SMTP obligatoire, seed refusé. |
| `PORT` | API | non (`8080`) | Port HTTP interne. |
| `UPLOAD_DIR` | API | non (`./data/uploads`) | Dossier des photos traitées (volume persistant en production). |
| `ALLOWED_ORIGINS` | API | non | Origines web autorisées (CORS + WebSocket), séparées par des virgules. |
| `TRUSTED_PROXIES` | API | non | IP/CIDR des reverse proxies dont `X-Forwarded-For` est fiable (rate limiting par client). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | API | en production | Envoi des codes de récupération. |
| `POSTGRES_PASSWORD` | Compose | non | Mot de passe Postgres du conteneur. |
| `EXPO_PUBLIC_API_URL` | App | recommandé | URL publique de l'API intégrée au build (jamais de secret côté app). |

Aucun secret n'est versionné ; `.env` est ignoré par git.

## Base de données, migrations et seed

- Schéma : `services/api/internal/platform/db/migrations/*.sql`, embarqué dans le binaire.
- Chaque fichier est appliqué **une seule fois**, dans une transaction, sous verrou consultatif, et enregistré dans `schema_migrations`. Ajoutez une migration en créant `002_xxx.sql` (ne modifiez jamais une migration déjà déployée).
- Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (likes/passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Contraintes, clés étrangères (`ON DELETE CASCADE` pour les données personnelles), unicités et index détaillés dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
- Le seed (`cmd/seed`) crée des comptes `demo-NN@demo.invalid` marqués `users.is_demo`, avec des portraits abstraits générés (clairement factices) et une bio suffixée « (profil de démonstration) ». Il refuse de s'exécuter si `APP_ENV=production`, et l'API ne montre jamais ces comptes en production.

> Base d'une version précédente du boilerplate ? Le schéma a été entièrement refondu : repartez d'une base vide (`docker compose down -v`).

## Tests et qualité

### API (`services/api`)

```bash
gofmt -l .                     # doit être vide
go vet ./...
go test ./...                  # tests unitaires
# Tests d'intégration (base JETABLE : son schéma est réinitialisé)
DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable' go test -race ./...
go build ./cmd/api
```

Les tests d'intégration démarrent l'API complète sur un vrai serveur HTTP et couvrent : santé, inscription/connexion/refresh/déconnexion (révocation immédiate), récupération de compte et verrouillage, validation de profil (18+, date verrouillée, coordonnées arrondies, aucune fuite de données privées), photos (types dangereux, limite de 6, ordre, URLs signées, traversée de chemin, suppression des fichiers), filtres de découverte, like/match sans doublon y compris en **likes simultanés**, chat temps réel par WebSocket, permissions (un non-participant ne peut ni lire ni écrire), accusés de lecture, pagination, suppression locale, annulation de match, blocage/signalement, notifications et effacement complet du compte.

### App (`apps/mobile`)

```bash
npm run typecheck       # TypeScript strict
npm run lint            # ESLint (0 avertissement toléré)
npm run format:check    # Prettier
npm test                # Jest : validation, formatage, cache temps réel, composants
```

### E2E navigateur

Parcours complet à deux utilisateurs sur le build web contre une vraie API (**base vide requise**) : inscription → onboarding avec upload de photo → découverte → like → « Vous ont liké » → match → chat temps réel dans les deux sens → accusé « Lu » → notifications → suppression de compte.

```bash
npm run build:web                                   # avec EXPO_PUBLIC_API_URL défini
python3 -m http.server 8081 --directory dist &      # ou tout serveur statique
WEB_URL=http://localhost:8081 API_URL=http://localhost:18080 npm run e2e:web
```

Captures d'écran dans `apps/mobile/e2e/output/`. La CI (`.github/workflows/ci.yml`) exécute tout cela à chaque push, plus un démarrage `docker compose` vérifié par `/health`.

## Build et déploiement

- **API** : `docker build -t lueur-api ./services/api` (image Alpine, utilisateur non-root, binaires `api` et `seed`). Prévoir un volume persistant pour `UPLOAD_DIR`, un PostgreSQL managé, `APP_ENV=production`, un SMTP, `ALLOWED_ORIGINS` et `TRUSTED_PROXIES` adaptés, TLS terminé par le reverse proxy (HSTS envoyé quand `X-Forwarded-Proto=https`). Le reverse proxy doit autoriser l'upgrade WebSocket sur `/realtime`.
- **Web** : `npm run build:web` produit un site statique dans `apps/mobile/dist` (à servir derrière n'importe quel CDN).
- **Mobile** : builds natifs avec EAS (`npx eas build -p ios|android`) en définissant `EXPO_PUBLIC_API_URL` vers l'URL HTTPS de production. Les textes de permission (photos, localisation approximative) sont déclarés dans `app.json` ; la localisation précise et en arrière-plan est explicitement bloquée sur Android.
- **Mise à l'échelle** : le hub temps réel et le rate limiter sont en mémoire (une instance). Pour plusieurs instances, remplacer `realtime.Publisher` par un bus partagé (LISTEN/NOTIFY Postgres ou Redis) et placer le rate limiting au niveau de la passerelle ; les photos peuvent passer sur un stockage objet via l'interface `media.Storage`.

## API

Toutes les routes (sauf auth publique, `/health` et `/media`) exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "message", "code": "code_machine"}`.

| Méthode & route | Rôle |
| --- | --- |
| `POST /auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` | Session (access token 15 min, refresh 30 j avec rotation) |
| `POST /auth/password/forgot` · `/auth/password/reset` | Récupération de compte par code |
| `GET /me` · `DELETE /me` | Compte courant · suppression (mot de passe requis) |
| `GET /interests` | Catalogue des centres d'intérêt |
| `GET/PUT /profile` · `PUT /profile/preferences` · `PUT/DELETE /profile/location` | Profil, préférences, position approximative |
| `GET /profiles/:userId` | Profil public (si visible : non bloqué, et match ou profil découvrable) |
| `GET /photos` · `POST /photos` · `PUT /photos/order` · `PUT/DELETE /photos/:id` | Photos (multipart, champ `photo`) |
| `GET /media/:key?exp&sig` | Image via URL signée expirante |
| `GET /discovery` · `POST /discovery/:userId/like` · `POST /discovery/:userId/pass` | Découverte et décisions |
| `GET /likes/received` · `DELETE /matches/:matchId` | Qui m'a liké · annuler un match |
| `GET /conversations` · `GET/DELETE /conversations/:id` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` | Messagerie |
| `GET /notifications` · `GET /notifications/unread-count` · `POST /notifications/read` · `POST /notifications/:id/read` | Notifications |
| `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` | Blocage, signalement |
| `POST /realtime/ticket` puis `GET /realtime?ticket=` (WebSocket) | Événements `message.new`, `conversation.read`, `conversation.closed`, `match.new`, `notifications.unread` |

## Décisions techniques et produit

- **On garde la base existante** (Go/Gin/pgx, Expo/React Query) : elle était saine. Les modules sans rapport avec la rencontre (facturation Stripe, organisations multi-tenant, posts, commentaires, votes, fichiers génériques) ont été supprimés — un paywall est contraire au produit. Tamagui, installé mais inutilisé, a été retiré.
- **Majorité** : 18 ans minimum vérifiés côté serveur ; la date de naissance est figée après création pour empêcher la triche sur l'âge ; les préférences d'âge ne descendent jamais sous 18 ans.
- **Matching** : un like ou un passage est définitif (pas de « rewind ») ; le match se crée dans la même transaction que le second like, sous verrou par paire, avec contrainte d'unicité sur la paire ordonnée. Le classement est isolé derrière l'interface `Ranker` pour pouvoir évoluer.
- **Gratuité réelle** : « Vous ont liké » est accessible à tous ; les limites de débit (120 swipes/min, 60 messages/min) ne servent qu'à bloquer les robots.
- **Localisation** : demandée uniquement sur action de l'utilisateur, précision « basse », arrondie à ~1 km sur l'appareil puis côté serveur ; les autres ne voient qu'une distance arrondie (paliers 1/5/10 km) et chacun peut la masquer ou supprimer sa position. Sans position, aucun filtre de distance n'est appliqué.
- **Photos** : type déterminé par le contenu (JPEG/PNG uniquement), garde contre les « decompression bombs », redimensionnement à 1440 px et ré-encodage JPEG qui supprime toutes les métadonnées. Les fichiers ne sont servis que via des URLs HMAC expirantes (≥ 1 h, cache-friendly) et disparaissent avec la photo ou le compte.
- **Sessions** : chaque requête vérifie que la session n'est pas révoquée ; logout, changement de mot de passe et suppression de compte coupent l'accès immédiatement.
- **Temps réel** : WebSocket authentifié par ticket à usage unique de 30 s (pas de token longue durée dans les URLs) ; à chaque (re)connexion le client resynchronise par REST, donc aucun événement manqué n'est perdu.
- **Blocage / signalement** : un signalement bloque automatiquement ; le blocage est symétrique (découverte, profil, chat), ferme le match et supprime les notifications mutuelles. Les signalements sont conservés (anonymisés si un compte est supprimé) dans `reports` pour la modération (outil de modération à brancher).
- **Suppression** : `DELETE /me` efface en cascade profil, photos (et fichiers), swipes, matchs, conversations, messages, notifications et sessions. Côté partenaire, la conversation disparaît.
- **Suppression locale d'une conversation** : masque l'historique pour soi seulement ; elle réapparaît si un nouveau message arrive.
- **Notifications push natives** (APNs/FCM) : non incluses ; les notifications sont in-app et temps réel. Le point d'extension est `notifications.Service`.
