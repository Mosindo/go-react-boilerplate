# Lumen — application de rencontre 100 % gratuite

Lumen est une application de rencontre mobile (Expo / React Native) adossée à une API Go.
Deux personnes créent un profil, se découvrent, se likent, matchent quand l'intérêt est réciproque, puis discutent en temps réel.

**Gratuit, vraiment** : aucun abonnement, aucun paywall, aucun « boost » payant, aucun like limité, aucune publicité.
(Les seules limites sont des garde-fous anti-abus : voir *Sécurité*.)

- Découverte par swipe (ou par boutons — accessible), carte avec photos, centres d'intérêt, distance approximative
- Match réciproque sans doublon, y compris sous concurrence
- Chat temps réel (WebSocket) réservé aux matchs actifs : lu / non lu, heure d'envoi, effacement local
- Notifications (match, nouveau message) avec badge et push temps réel
- Photos : upload sécurisé, recompression, ordre, photo principale, remplacement, suppression
- Confidentialité : masquer son profil, masquer sa distance, supprimer sa position, blocage, signalement, suppression de compte
- Récupération de compte par e-mail, changement de mot de passe, mode sombre automatique

## Stack

| Couche | Technologie |
| --- | --- |
| Mobile | Expo SDK 54, React Native 0.81, TypeScript strict, React Navigation, TanStack Query |
| API | Go 1.24, Gin, pgx/pgxpool (pas d'ORM), JWT HS256, bcrypt, gorilla/websocket |
| Base | PostgreSQL 16 |
| Infra | Docker, Docker Compose |

## Architecture

```
apps/mobile/                  Application Expo
  src/api/                    Client HTTP typé (refresh de token automatique), endpoints, types
  src/hooks/                  useAuth, useRealtime (WebSocket → cache React Query), useData
  src/screens/                Écrans (Auth, Onboarding, Découvrir, Messages, Conversation, Profil…)
  src/components/             SwipeDeck, PhotoManager, PreferencesEditor, MatchModal…
  src/shared/                 Design system (ui/), layout, feedback, images protégées
  src/navigation/             Stack + onglets
services/api/
  cmd/api/                    Serveur HTTP (+ `promote-admin`)
  cmd/seed/                   Données de démonstration (développement uniquement)
  internal/platform/          config, db + migrations, middleware, realtime, storage, mailer, httpx
  internal/features/          auth · users (compte) · profiles · photos · matching · chat · notifications · safety
```

Chaque feature suit `handler → service → repository` (+ `model.go`, `routes.go`) : pas de SQL dans les handlers, règles métier dans les services.

### Modèle de données (migrations `014`, `015`)

`users` · `profiles` · `preferences` · `interests` / `user_interests` · `photos` · `swipes` · `matches` · `conversations` / `conversation_participants` / `messages` · `blocks` · `reports` · `notifications` · `sessions` · `password_reset_tokens`.

Contraintes : clés étrangères avec `ON DELETE CASCADE` (la suppression d'un compte efface tout), unicité des swipes `(from, to)`, des matchs (`user_a < user_b` + `UNIQUE`), des positions de photos ; `CHECK` sur genre, âge, longueurs, statuts. Index sur chaque chemin chaud (découverte, messages non lus, notifications, matchs actifs).

### API (résumé)

| Domaine | Endpoints |
| --- | --- |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/password/forgot`, `/auth/password/reset` · `GET /me` · `POST /me/password` · `DELETE /me` |
| Profil | `GET/PUT /me/profile` · `GET/PUT /me/preferences` · `PUT/DELETE /me/location` · `GET /interests` · `GET /profiles/:id` |
| Photos | `GET/POST /me/photos` · `PUT /me/photos/order` · `PUT/DELETE /me/photos/:id` · `GET /photos/:id/file` |
| Matching | `GET /discover?limit=` · `POST /swipes` · `GET /matches` · `DELETE /matches/:id` |
| Chat | `GET /conversations` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` · `DELETE /conversations/:id` |
| Notifications | `GET /notifications` · `POST /notifications/read-all` · `POST /notifications/:id/read` |
| Sécurité | `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` · `GET /admin/reports` · `POST /admin/reports/:id/resolve` |
| Temps réel | `POST /ws/ticket` puis `GET /ws?ticket=…` |

Erreurs : `{"error": "message"}` ; listes paginées (`limit`, `offset` ou curseur `before`) avec bornes serveur.

## Installation

Prérequis : Go 1.24+, Node 20+ (testé en 22), Docker (ou un PostgreSQL 15+ local).

```bash
git clone <repo> && cd go-react-boilerplate
cp .env.example .env              # puis renseignez JWT_SECRET (openssl rand -hex 32)
```

### Variables d'environnement

Toutes sont décrites dans [`.env.example`](.env.example).

| Variable | Rôle | Défaut |
| --- | --- | --- |
| `JWT_SECRET` | **Obligatoire.** ≥ 32 caractères, valeurs « placeholder » refusées | — |
| `DATABASE_URL` | **Obligatoire.** Chaîne PostgreSQL | — |
| `PORT` | Port d'écoute de l'API | `8080` |
| `APP_ENV` | `development` ou `production` (production exige SMTP et le rate limit) | `development` |
| `UPLOAD_DIR` | Dossier des photos (volume persistant en prod) | `./data/uploads` |
| `ALLOWED_ORIGINS` | Origines CORS autorisées (CSV) | vide |
| `TRUSTED_PROXIES` | Proxys dont `X-Forwarded-For` est cru (CSV) | vide |
| `RATE_LIMIT` | `on` / `off` (`off` interdit en production) | `on` |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USERNAME` `SMTP_PASSWORD` `SMTP_FROM` | E-mails de récupération | vide |
| `EXPO_PUBLIC_API_URL` | (mobile) URL de l'API — **IP LAN sur téléphone physique, jamais `localhost`** | `localhost:18080` / `10.0.2.2:18080` |

Aucun secret ne doit être commité ni placé dans le frontend (les variables `EXPO_PUBLIC_*` sont publiques par nature).

### Base de données et migrations

Les migrations SQL (`services/api/internal/platform/db/migrations`) sont embarquées dans le binaire et appliquées **au démarrage**, de façon idempotente et sous verrou consultatif (plusieurs instances peuvent démarrer ensemble).

```bash
docker compose up -d postgres     # PostgreSQL sur localhost:5432 (postgres/postgres, base "app")
```

### Données de démonstration (développement uniquement)

```bash
cd services/api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
go run ./cmd/seed -city paris                      # 14 membres fictifs, photos générées
go run ./cmd/seed -likes-me vous@exemple.fr        # 5 membres ont déjà liké votre compte
```

Les comptes de démo sont `demo.<prénom>@demo.invalid` (TLD réservé `.invalid` : impossible de les confondre avec de vrais utilisateurs ou de leur écrire), mot de passe `DemoPassword123` (surchargeable via `DEMO_PASSWORD`). La commande **refuse de s'exécuter** avec `APP_ENV=production`.

### Lancer en local

```bash
# API (dans services/api) — 8080 par défaut
export DATABASE_URL=... JWT_SECRET=...
go run ./cmd/api

# ou toute la stack avec Docker (API sur http://localhost:18080)
JWT_SECRET=$(openssl rand -hex 32) docker compose up --build

# Mobile
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start      # ou : npm run start:phone
```

En développement sans SMTP, le **code de récupération de mot de passe est écrit dans les logs de l'API** (`[dev mailer]`).

Avant d'annoncer l'app « prête » (réseau) : `docker compose ps`, `curl http://<IP_LAN>:18080/health`, puis `npm run e2e:smoke` (voir ci-dessous). Si un service local détourne le port 18080, changez le port hôte dans `docker-compose.yml`, `apps/mobile/src/api/client.ts`, `scripts/start-phone.js` et ce README.

### Promouvoir un modérateur

```bash
cd services/api && go run ./cmd/api promote-admin moderateur@exemple.fr
```

L'admin liste les signalements (`GET /admin/reports`) et les traite (`POST /admin/reports/:id/resolve`, option `suspendUser`).

## Tests, qualité, build

```bash
# API  (les tests d'intégration lisent DATABASE_URL_TEST ; sans base ils sont ignorés)
cd services/api
export DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable   # base créée au préalable
gofmt -l . && go vet ./... && go test ./... && go build ./cmd/api

# Mobile
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test

# Parcours critique de bout en bout contre une API qui tourne
npm run e2e:smoke                # inscription → profil+photo → découverte → like → match → chat → notifs → sécurité → suppression
npm run e2e:ui:setup && npm run e2e:ui:run     # flux Maestro sur appareil/émulateur (Windows : run-maestro.ps1)
```

La suite d'intégration Go couvre : auth (rotation de refresh, révocation de session), validation de profil (18+, date immuable), pipeline photo (types, bombes de décompression, recompression, accès), filtres de découverte (préférences mutuelles, distance, blocages, comptes suspendus), idempotence des swipes, **match unique sous likes simultanés**, chat (permissions, lu/non lu, pagination, effacement local), blocage/signalement/modération, suppression de compte (données + fichiers), WebSocket (tickets, livraison temps réel), rate limiting.

### Build / déploiement

```bash
docker build -t lumen-api services/api      # image non-root, binaires api + seed, volume /data/uploads
```

En production : `APP_ENV=production`, un `JWT_SECRET` fort, SMTP configuré, HTTPS devant l'API (reverse proxy ; renseigner `TRUSTED_PROXIES` pour que le rate limit voie la vraie IP), un volume persistant sur `UPLOAD_DIR`, des sauvegardes PostgreSQL. Application mobile : `eas build` (définir `EXPO_PUBLIC_API_URL` vers l'URL HTTPS de l'API).

## Sécurité

- JWT HS256 uniquement, expiration obligatoire (15 min) + refresh tokens opaques hachés et rotatifs ; **chaque requête vérifie que la session est vivante et le compte actif** → déconnexion, suspension et suppression sont immédiates.
- bcrypt, mots de passe limités à 72 caractères (limite de bcrypt), e-mail normalisé (trim + minuscules).
- Toutes les règles métier sont côté serveur : âge ≥ 18 et date de naissance immuable, éligibilité d'un swipe (mêmes critères que la découverte), accès au chat limité aux **matchs actifs sans blocage** (un tiers reçoit 404, jamais 403).
- Aucun endpoint ne liste les utilisateurs ; un profil n'est visible que s'il est découvrable, matché, et sans blocage. Jamais d'e-mail, de date de naissance ni de coordonnées dans les réponses.
- Localisation : coordonnées arrondies à ~1 km **côté serveur**, jamais renvoyées ; distance affichée arrondie (au km, puis par 5 km) et désactivable.
- Photos : taille ≤ 5 Mo, JPEG/PNG détectés par contenu, dimensions vérifiées **avant** décodage (anti-bombe), ré-encodage en JPEG (métadonnées EXIF/GPS et charges cachées supprimées), noms de fichiers générés, stockage hors web-root, service uniquement via l'API après contrôle d'accès.
- Rate limiting (token bucket) : global par IP, strict sur login/inscription/récupération, par utilisateur sur swipes, messages, uploads, signalements ; corps JSON limités à 1 Mo ; en-têtes de sécurité ; CORS en liste blanche.
- WebSocket : ticket de 60 s à usage dédié (le JWT d'accès ne transite jamais dans une URL ; un JWT d'accès n'ouvre pas de socket).
- La récupération de compte ne révèle pas l'existence d'un compte ; codes de 10 caractères à usage unique, valables 15 min.

## Principales décisions techniques

- **Construit sur le boilerplate** (Go/Gin/pgx, Expo, JWT, sessions, notifications) plutôt que réécrit. Les modules `posts`, `comments`, `billing` (Stripe), `files` et la notion d'organisation n'ont aucun sens pour une app gratuite : ils ne sont **plus routés**.
- **Swipes immuables** : un like ou un « passer » n'est jamais réécrit ; un profil traité ne revient jamais (y compris après désappairage ou déblocage). Choix le plus simple et le plus sûr contre le harcèlement par re-like.
- **Match = verrou consultatif par paire** + `UNIQUE (user_a, user_b)` : deux likes simultanés créent exactement un match et une conversation.
- **Recommandation** : un seul point d'extension (`candidateScore` dans `matching/repository.go`) — intérêts communs, personnes ayant déjà liké, activité récente, proximité, légère variété. Les filtres (`eligibleFrom`) sont séparés du classement et réutilisés pour valider les swipes.
- **Distance** : formule de haversine en SQL sur coordonnées arrondies (pas de PostGIS). Au-delà de plusieurs centaines de milliers de profils, passer à PostGIS/`earthdistance` + index spatial.
- **Temps réel** : WebSocket serveur → client uniquement ; toutes les écritures passent par l'API REST validée. Le hub est en mémoire (1 instance) ; pour plusieurs instances, relayer via Redis ou `LISTEN/NOTIFY`. Sans socket, l'app bascule sur du polling léger.
- **Swipe sans dépendance native** (PanResponder + Animated) ; les boutons Passer / J'aime offrent la même action pour l'accessibilité.
- **Notifications** : stockées + poussées en direct ; un flot de messages ne produit qu'une notification non lue par conversation, sans contenu de message (confidentialité écran verrouillé). Pas de push APNs/FCM (hors périmètre sans comptes développeur) : voir *Limites*.
- **Langue** : interface en français, sans bibliothèque d'i18n (les chaînes sont dans les composants).
- **Suppression de compte** : suppression physique en cascade + effacement des fichiers (conformité RGPD), confirmée par mot de passe.

## Limites connues et pistes

- Pas de notifications push hors-application (APNs/FCM) ni de vérification d'e-mail à l'inscription.
- Pas de modération automatique des photos (seulement signalement + modération manuelle).
- Hub temps réel et rate limiter en mémoire : un seul réplica d'API en l'état.
- `POST /auth/register` répond 409 si l'e-mail existe (énumération possible, limitée par le rate limit).
- Les tests UI Maestro nécessitent un appareil/émulateur ; ils n'ont pas pu être exécutés dans l'environnement de développement automatisé (les tests de composants Jest et le parcours API de bout en bout, eux, le sont).
