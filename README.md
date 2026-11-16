# Lueur — application de rencontre gratuite

Lueur est une application de rencontre moderne, **100 % gratuite** : pas d'abonnement, pas de paywall,
pas de « boost » payant, pas de likes limités, pas de publicité. Deux personnes peuvent créer un compte,
construire leur profil, découvrir des profils proches, liker ou passer, matcher quand l'intérêt est
réciproque, discuter en temps réel, et gérer leur confidentialité (pause, blocage, signalement,
suppression du compte).

- **API** : Go + Gin + PostgreSQL (`pgxpool`, SQL explicite, pas d'ORM), JWT HS256, WebSocket.
- **App** : React Native / Expo (iOS, Android, web) + TypeScript + React Query.
- **Infra** : Docker, Docker Compose, GitHub Actions.

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Architecture](#architecture)
3. [Installation et lancement local](#installation-et-lancement-local)
4. [Variables d'environnement](#variables-denvironnement)
5. [Base de données, migrations et seed](#base-de-données-migrations-et-seed)
6. [Tests et qualité](#tests-et-qualité)
7. [Build et déploiement](#build-et-déploiement)
8. [API](#api)
9. [Sécurité et vie privée](#sécurité-et-vie-privée)
10. [Décisions techniques et produit](#décisions-techniques-et-produit)

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription (18+), connexion, sessions avec rotation du refresh token, mot de passe oublié (code à 6 chiffres par email), changement de mot de passe, déconnexion, suppression définitive du compte |
| Onboarding | 5 étapes reprenables : prénom + date de naissance → genre, qui rencontrer, intention → photos → bio, métier, ville, centres d'intérêt → localisation approximative (facultative) |
| Profil | Prénom, âge (calculé), genre, bio, métier, taille, intention, ville, centres d'intérêt (36), 1 à 6 photos ordonnées avec photo principale |
| Photos | Upload, compression sur l'appareil, réordonnancement, photo principale, remplacement, suppression ; validation et ré-encodage serveur |
| Découverte | Pile de cartes swipe (glisser ou boutons), carrousel photos, profil complet, centres d'intérêt communs mis en avant, distance approximative |
| Matching | Like / pass, match si réciproque, aucun doublon, profils déjà traités jamais réaffichés, célébration du match |
| Chat | Conversations après match, temps réel (WebSocket), heure d'envoi, lu / non lu, compteur de non-lus, historique paginé, envoi optimiste et idempotent, renvoi en cas d'échec |
| Notifications | Fil d'activité (nouveaux matchs, nouveaux messages), badges d'onglets, tout marquer comme lu |
| Préférences | Genres recherchés, tranche d'âge, distance maximale — appliquées **dans les deux sens** |
| Confidentialité | Pause du profil, mise à jour de la position, liste des personnes bloquées / déblocage |
| Modération | Bloquer, signaler (avec blocage automatique), annuler un match |

Il n'existe aucune fonctionnalité payante, ni de code de facturation : le module Stripe du boilerplate
d'origine a été supprimé.

---

## Architecture

```text
.
├─ apps/mobile/                 # App Expo (iOS, Android, web)
│  ├─ App.tsx
│  ├─ e2e/web-critical-flow.js  # E2E Playwright du parcours critique
│  ├─ scripts/                  # smoke API, lancement sur téléphone
│  └─ src/
│     ├─ api/                   # client HTTP unique + modules typés (auth, profile, social)
│     ├─ components/            # composants métier (SwipeDeck, PhotoGrid, SafetySheet…)
│     ├─ hooks/                 # useAuth, React Query (queries.ts), useRealtime
│     ├─ navigation/            # RootNavigator (auth → onboarding → onglets)
│     ├─ screens/               # auth, onboarding, discover, matches, notifications, profile
│     ├─ store/                 # stockage sécurisé des tokens
│     ├─ theme/                 # design tokens
│     ├─ ui/                    # kit UI (Text, Button, TextField, Sheet, Toast…)
│     └─ utils/                 # formatage, erreurs, photo, localisation
├─ services/api/                # API Go
│  ├─ cmd/api/                  # point d'entrée + routeur + tests d'intégration
│  ├─ cmd/seed/                 # données de démonstration (dev uniquement)
│  └─ internal/
│     ├─ platform/              # config, db+migrations, middleware, httpx, ratelimit,
│     │                         # storage, imaging, media, mailer, realtime, logger
│     └─ features/              # auth, profiles, photos, discovery, matching,
│                               # chat, safety, notifications
├─ docs/ARCHITECTURE.md
├─ scripts/qa.sh                # porte qualité locale complète
└─ docker-compose.yml
```

Chaque feature suit `handler.go → service.go → repository.go` (+ `model.go`, `routes.go`) :
handlers fins, règles métier dans les services, SQL uniquement dans les repositories.
Détails : [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Installation et lancement local

### Prérequis

- Docker + Docker Compose (recommandé), **ou** Go 1.23+ et PostgreSQL 16
- Node.js 20+ (22 recommandé) et npm

### 1. Configurer l'environnement

```bash
cp .env.example .env
# Génère un secret fort et colle-le dans JWT_SECRET :
openssl rand -hex 32
```

### 2. Lancer PostgreSQL + l'API avec Docker

```bash
docker compose up --build -d
curl http://localhost:18080/health     # {"status":"ok"}
```

L'API écoute sur `http://localhost:18080`. Les migrations s'appliquent au démarrage.

### 3. (Optionnel) Données de démonstration

```bash
docker compose exec api ./seed
# Connexion : camille.01@demo.lueur.invalid / DemoPassword1
```

### 4. Lancer l'app

```bash
cd apps/mobile
npm ci
npm run web        # navigateur (http://localhost:8081)
npm run ios        # simulateur iOS
npm run android    # émulateur Android (utilise 10.0.2.2 automatiquement)
npm run start:phone  # téléphone physique : détecte l'IP LAN et configure l'URL de l'API
```

Sur un **téléphone physique**, n'utilise jamais `localhost` : définis
`EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080` (fait automatiquement par `npm run start:phone`).

### Lancer l'API sans Docker

```bash
cd services/api
export DATABASE_URL='postgresql://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/api          # PORT=8080 par défaut
go run ./cmd/seed         # données de démo (optionnel)
```

---

## Variables d'environnement

Toutes sont documentées dans [`.env.example`](.env.example).

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | oui | — | Secret HS256, 32 caractères minimum, valeurs « placeholder » refusées. Sert aussi à dériver les clés de signature des médias et des codes de réinitialisation. |
| `DATABASE_URL` | oui | — | Connexion PostgreSQL |
| `PORT` | non | `8080` | Port HTTP de l'API |
| `APP_ENV` | non | `development` | `development`, `test` ou `production` (masque les comptes démo, désactive le mailer de dev, mode release) |
| `UPLOAD_DIR` | non | `./data/uploads` | Stockage local des photos |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS + WebSocket). Les apps natives n'envoient pas d'Origin. |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies autorisés à fournir `X-Forwarded-For` |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | — | Envoi des codes de réinitialisation. Sans SMTP : codes affichés dans les logs en dev, emails ignorés en production. |
| `DATABASE_URL_TEST` | tests | — | Base dédiée aux tests d'intégration |
| `SEED_PASSWORD` | seed | `DemoPassword1` | Mot de passe des comptes démo |
| `EXPO_PUBLIC_API_URL` | app | `localhost:18080` / `10.0.2.2:18080` en dev | URL de l'API pour l'app. **Embarquée dans le bundle : jamais de secret.** |

Aucun secret n'est commité : `.env` est ignoré par git.

---

## Base de données, migrations et seed

Schéma (PostgreSQL 16) dans `services/api/internal/platform/db/migrations` :

| Migration | Tables |
| --- | --- |
| `001_accounts` | `users`, `sessions`, `password_resets` |
| `002_profiles` | `profiles`, `preferences`, `interests` (catalogue de référence), `user_interests`, `photos` |
| `003_matching` | `swipes` (likes et passes), `matches` (une ligne par paire, sert de conversation), `messages` |
| `004_safety` | `blocks`, `reports` |
| `005_notifications` | `notifications` |

- Clés étrangères avec `ON DELETE CASCADE` : supprimer un compte supprime profil, photos, swipes,
  matchs, messages, blocages et notifications. Les signalements sont conservés anonymisés
  (`SET NULL`) pour l'historique de modération.
- Contraintes : email normalisé unique, paire de match canonique (`user_a_id < user_b_id`) unique,
  swipe unique par paire, positions de photos uniques (contrainte différée), `CHECK` sur genres,
  âges, distances, longueurs de textes, raisons de signalement.
- Index sur tous les chemins chauds : découverte (genre/date de naissance, localisation),
  swipes inversés, matchs par activité, messages par conversation, non-lus, notifications.

**Migrations** : appliquées automatiquement au démarrage, une seule fois chacune, dans une
transaction, sous verrou consultatif (plusieurs instances peuvent démarrer en même temps), et
tracées dans `schema_migrations`. Pour ajouter une évolution : créer `006_xxx.sql`.

> Si l'API refuse de démarrer avec « legacy untracked schema », la base contient l'ancien schéma
> du boilerplate : recrée-la (`docker compose down -v`).

**Seed** : `go run ./cmd/seed` (ou `docker compose exec api ./seed`) crée 20 profils de
démonstration autour de Paris, idempotent. Ils sont clairement séparés des vrais comptes :
drapeau `users.is_demo`, emails en `@demo.lueur.invalid` (domaine réservé), bio commençant par
« Profil de démonstration », photos abstraites générées (aucune vraie personne). Ils sont exclus
de la découverte quand `APP_ENV=production`, et le seed refuse de s'exécuter en production.
`go run ./cmd/seed -purge` supprime tous les comptes démo.

---

## Tests et qualité

```bash
# Tout d'un coup (Go + mobile + smoke API si une API tourne sur :18080)
DATABASE_URL_TEST='postgresql://app:app@localhost:5432/app_test?sslmode=disable' ./scripts/qa.sh
```

### API

```bash
cd services/api
gofmt -l .                 # doit être vide
go vet ./...
DATABASE_URL_TEST=... go test -race ./...
go build ./cmd/api
```

- **Unitaires** : politique de mot de passe, validation email, âge, validation de profil et de
  préférences, arrondi de la position, distance approximative, haversine, classement de la
  découverte, normalisation des messages, JWT (alg, expiration, claims), rate limiter, stockage
  (anti path-traversal), traitement d'image, URL signées, tickets temps réel, configuration.
- **Intégration (PostgreSQL réel)** : cycle d'authentification complet (rotation, révocation),
  réinitialisation du mot de passe et anti-bruteforce, onboarding et validations, gestion des
  photos (types dangereux, limite de 6, ordre, remplacement, suppression, URL signées), filtres de
  découverte (orientation mutuelle, âges mutuels, distance, pause, sans photo, absence de fuite de
  données privées), like / pass / match / doublons, **likes mutuels concurrents → un seul match**,
  chat (permissions, pagination, idempotence, accusés de lecture), notifications, blocage,
  signalement, suppression de compte et purge des fichiers, temps réel WebSocket, rate limiting.

Sans `DATABASE_URL_TEST`, les tests d'intégration sont ignorés (les unitaires tournent).

### App

```bash
cd apps/mobile
npm run typecheck
npm run lint
npm run format:check
npm run build:web
```

### End-to-end

```bash
# API démarrée sur :18080 avec ALLOWED_ORIGINS incluant http://localhost:8081
cd apps/mobile
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke   # parcours complet via l'API
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
E2E_API_URL=http://localhost:18080 npm run e2e:web           # parcours complet via l'UI (Playwright)
```

Le test UI pilote la vraie interface : inscription → création du profil (upload réel d'une
photo, géolocalisation) → découverte → like → match → conversation (réponse reçue en temps
réel) → signalement → suppression du compte. Les captures sont écrites dans
`apps/mobile/e2e/report/`. Playwright doit être disponible (`npm i --no-save playwright` puis
`npx playwright install chromium`).

### CI

`.github/workflows/ci.yml` : jobs `api` (gofmt, vet, tests race sur Postgres, build),
`mobile` (typecheck, lint, format, build web), `e2e` (API + smoke + Playwright) et `docker`
(build de l'image, démarrage de la stack, seed).

---

## Build et déploiement

### API

- Image Docker multi-étapes (`services/api/Dockerfile`) : binaires statiques `api` et `seed`,
  utilisateur non-root, volume `/data/uploads`.
- En production : `APP_ENV=production`, `JWT_SECRET` fort, `DATABASE_URL` (TLS recommandé),
  `ALLOWED_ORIGINS` pour le web, `TRUSTED_PROXIES` derrière un reverse proxy, SMTP configuré.
- Servir derrière un reverse proxy TLS (HSTS est envoyé quand la requête est en HTTPS) qui
  supporte les WebSocket (`/realtime`).
- Persister `UPLOAD_DIR` sur un volume (ou implémenter `storage.Storage` pour un stockage objet
  S3-compatible : une seule interface à implémenter).
- Le hub temps réel et le rate limiter sont en mémoire : pour plusieurs réplicas, ajouter une
  diffusion via Postgres `LISTEN/NOTIFY` (ou Redis) et un rate limit au niveau du proxy.

### App

- Web : `npm run build:web` produit un site statique dans `apps/mobile/dist`.
- iOS / Android : builds natifs avec EAS (`npx eas build`), en définissant
  `EXPO_PUBLIC_API_URL` vers l'URL publique HTTPS de l'API. Les textes de permission (photos,
  localisation « pendant l'utilisation ») sont déjà configurés dans `app.json` ; la localisation
  précise et en arrière-plan est explicitement bloquée sur Android.

---

## API

Toutes les réponses d'erreur ont la forme `{"error": "message", "code": "code_machine"}`.
Les routes marquées 🔒 exigent `Authorization: Bearer <accessToken>`.

| Méthode | Route | Description |
| --- | --- | --- |
| GET | `/health` | Santé (ping base) |
| POST | `/auth/register`, `/auth/login` | Création de compte / connexion → `accessToken` (15 min) + `refreshToken` (30 j) |
| POST | `/auth/refresh`, `/auth/logout` | Rotation / révocation du refresh token |
| POST | `/auth/password/forgot`, `/auth/password/reset` | Code de réinitialisation (réponse identique que le compte existe ou non) |
| GET / DELETE | `/me` 🔒 | Compte courant / suppression définitive (mot de passe requis) |
| POST | `/me/password` 🔒 | Changement de mot de passe (révoque toutes les sessions) |
| GET | `/interests` | Catalogue des centres d'intérêt |
| GET / PUT | `/me/profile` 🔒 | Profil + préférences + complétude / création-modification |
| PUT | `/me/preferences`, `/me/location`, `/me/visibility` 🔒 | Préférences, position approximative, pause |
| GET / POST | `/me/photos` 🔒 | Liste / upload (multipart `photo`) |
| PUT | `/me/photos/order` 🔒 | Réordonnancement (index 0 = principale) |
| PUT / DELETE | `/me/photos/:photoId` 🔒 | Remplacement / suppression |
| GET | `/media/photos/:photoId?exp&sig` | Photo via URL signée expirante |
| GET | `/profiles/:userId` 🔒 | Profil public (si visible et non bloqué) |
| GET | `/discover?limit=` 🔒 | Pile de profils classés |
| POST | `/swipes` 🔒 | `{targetUserId, action: like\|pass}` → `{matched, match?}` |
| GET | `/matches`, `/matches/:matchId` 🔒 | Matchs avec dernier message et non-lus |
| DELETE | `/matches/:matchId` 🔒 | Annuler un match |
| GET / POST | `/matches/:matchId/messages` 🔒 | Historique (`?before=<id>`) / envoi (`clientMessageId` pour l'idempotence) |
| POST | `/matches/:matchId/read` 🔒 | Marquer la conversation comme lue |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId` 🔒 | Liste / bloquer / débloquer |
| POST | `/reports` 🔒 | Signaler (bloque aussi) |
| GET / POST | `/notifications`, `/notifications/:id/read`, `/notifications/read-all` 🔒 | Fil d'activité |
| POST / GET | `/realtime/ticket` 🔒, `/realtime?ticket=` | Ticket à usage unique (30 s) puis WebSocket |

Événements temps réel : `message.created`, `conversation.read`, `match.created`,
`match.removed`, `notification.created`.

---

## Sécurité et vie privée

- **Authentification** : JWT HS256 strict (algorithme, expiration et claims obligatoires), access
  token de 15 min, refresh token aléatoire stocké haché et **tourné** à chaque usage, session
  vérifiée en base à chaque requête (déconnexion, réinitialisation et suppression sont immédiates).
- **Mots de passe** : bcrypt (coût 12), 8 caractères minimum avec lettre et chiffre, temps de
  réponse constant pour les emails inconnus.
- **Réinitialisation** : code à 6 chiffres haché (HMAC), 30 min, 5 essais max, 3 demandes/heure,
  usage unique, aucune énumération de comptes.
- **Autorisations côté serveur** : toutes les règles (âge, complétude, visibilité, blocages,
  appartenance à une conversation, propriété des photos) sont vérifiées par l'API. Une
  conversation inaccessible renvoie 404 (pas d'indication d'existence).
- **Données privées** : email, date de naissance et coordonnées ne sont jamais renvoyés aux autres
  membres. La position est arrondie à ~1 km **avant stockage**, la distance affichée est
  approximative (1 km mini, paliers de 5 km au-delà de 10 km). Les notifications de message ne
  contiennent pas le contenu du message.
- **Photos** : type détecté sur les octets (JPEG/PNG/WebP), 10 Mo max, dimensions bornées avant
  décodage (anti « decompression bomb »), image ré-encodée en JPEG (supprime EXIF/GPS et tout
  contenu caché), clés de stockage générées côté serveur (aucun chemin fourni par le client),
  accès uniquement via URL signée HMAC qui expire.
- **Concurrence** : un blocage et un like simultanés sont sérialisés par le même verrou de paire
  (jamais de match entre personnes bloquées) ; un message envoyé pendant une annulation de match
  est refusé ; les essais de code de réinitialisation sont réservés atomiquement.
- **Temps réel** : chaque WebSocket est liée à une session et revalidée toutes les 25 s ; un
  changement de mot de passe ou une suppression de compte ferme immédiatement les connexions.
- **Abus** : rate limiting par IP (auth) et par utilisateur (swipes, messages, uploads,
  signalements, changements de position — contre la trilatération —, tickets temps réel), limite
  de taille de tous les corps de requête (y compris annoncés multipart), en-têtes de sécurité (CSP, nosniff,
  frame-ancestors, HSTS), CORS par liste blanche, IP client fiable uniquement via proxies déclarés.
- **Tokens côté app** : trousseau sécurisé de l'OS (expo-secure-store) ; sur le web,
  `sessionStorage` seulement.

### Limites connues

- Une URL de photo signée reste valide jusqu'à ~75 min, y compris après un blocage.
- La distance affichée (paliers de 1 km sous 10 km) combinée à de nombreux changements de
  position reste une source théorique de trilatération ; elle est freinée par le rate limit
  (10 changements/heure) et l'arrondi à ~1 km du stockage.
- Sans position partagée, le filtre de distance n'est pas appliqué (ni le sien, ni celui des autres).
- Pas de détection de réutilisation d'un refresh token déjà tourné (il est simplement refusé).

---

## Décisions techniques et produit

- **Construire sur le boilerplate** plutôt que réécrire : Go/Gin/pgx, architecture par feature,
  Expo + React Query étaient sains. Supprimés car contraires au produit : facturation Stripe
  (gratuité), posts/commentaires/votes, organisations multi-tenant, annuaire public `/users`
  (exposait les emails), Tamagui (installé mais inutilisé), scripts PowerShell.
- **Schéma réinitialisé** avec migrations tracées : l'ancien runner rejouait tous les fichiers à
  chaque démarrage sans suivi. Un garde-fou détecte l'ancien schéma au lieu de le corrompre.
- **`matches` = conversations** : une conversation n'existe qu'après un match ; inutile d'avoir
  une table séparée. Les likes et passes partagent la table `swipes`.
- **Premier choix définitif** : un like ou un pass ne se change pas (409) ; répéter le même choix
  est idempotent. Un match annulé ne renaît pas d'un nouveau like.
- **Concurrence** : un verrou consultatif transactionnel par paire garantit un seul match même si
  les deux personnes likent à la même milliseconde (testé).
- **Découverte en deux temps** : génération de candidats en SQL (toutes les règles dures :
  préférences mutuelles, distance mutuelle, blocages, déjà vus, visibilité) puis classement en Go
  derrière une interface `Ranker` (réciprocité, intérêts communs, activité récente, proximité,
  légère variation quotidienne). On peut remplacer le classement sans toucher aux règles.
- **Pas de position = pas de filtre de distance** pour cette personne (la localisation reste
  facultative et explicitement demandée).
- **Signaler bloque aussi** : option la plus sûre pour la personne qui signale. Le blocage est
  indiscernable d'une annulation de match pour la personne bloquée.
- **Date de naissance et genre** non modifiables depuis l'app après l'onboarding (anti-abus).
- **Temps réel** par WebSocket avec ticket à usage unique (le JWT ne passe jamais dans une URL),
  reconnexion avec backoff et resynchronisation des caches.
- **Swipe** codé avec `PanResponder` + `Animated` de React Native : fonctionne sur iOS, Android et
  web sans module natif supplémentaire ; boutons équivalents pour l'accessibilité.
- **Notifications push natives** non incluses : elles nécessitent un compte Expo/EAS et des
  identifiants Apple/Google. Le fil d'activité, les badges et le temps réel couvrent l'usage dans
  l'app ; l'ajout de push passerait par une table de tokens et `NotifyMatch/NotifyMessage`.
- **Identité visuelle** originale « Lueur » : palette sombre et chaude (ambre/braise), titres en
  serif, contrastes AA.

Voir aussi : [CONTRIBUTING.md](CONTRIBUTING.md), [CHANGELOG.md](CHANGELOG.md),
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
