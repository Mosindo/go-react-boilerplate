# Lueur

**Lueur** est une application de rencontre moderne et **100 % gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de limite artificielle de likes, pas de publicité. Toutes les fonctionnalités sont accessibles à tous les membres.

Le dépôt contient :

- une **API Go** (Gin, PostgreSQL via `pgx`, SQL explicite, sans ORM) ;
- une **application Expo / React Native** (iOS, Android et web) en TypeScript ;
- l'infrastructure **Docker Compose**, la CI GitHub Actions et des tests de bout en bout.

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation des refresh tokens, mot de passe oublié (code à 6 chiffres par e-mail), changement de mot de passe, déconnexion immédiate côté serveur, suppression définitive du compte |
| Onboarding | Prénom, date de naissance (18 ans minimum, non modifiable ensuite), genre, préférences, photos, bio, métier, centres d'intérêt, localisation approximative (facultative) |
| Profil | Édition, jusqu'à 6 photos (ajout, remplacement, suppression, ordre, photo principale), taille, ville, intention de rencontre, aperçu « tel que les autres me voient » |
| Découverte | Pile de profils avec swipe (et boutons équivalents), carrousel de photos, profil complet, centres d'intérêt communs, distance arrondie |
| Matching | Like / passer, match quand l'intérêt est réciproque, célébration du match, annulation de match |
| Chat | Conversations réservées aux matchs, temps réel (WebSocket), heure d'envoi, état lu (« Vu »), non-lus, pagination de l'historique |
| Notifications | Nouveau match, nouveau message (regroupés tant que non lus), compteurs, tout marquer comme lu, push temps réel dans l'app |
| Confidentialité | Mettre son profil en pause, masquer sa distance, supprimer sa position |
| Sécurité | Bloquer (discret), signaler (bloque aussi), masquage automatique d'un profil après 3 signalements distincts, liste des membres bloqués |

---

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256, bcrypt, `gorilla/websocket` (temps réel), `golang.org/x/image` (décodage WebP et redimensionnement des photos).
- **Mobile / web** : Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 6 (native stack + bottom tabs), TanStack Query 5, `expo-image`, `expo-image-picker`, `expo-image-manipulator`, `expo-location`, `expo-secure-store`.
- **Qualité** : `gofmt`, `go vet`, tests Go unitaires + intégration (Postgres réel, détecteur de courses), ESLint (config Expo), Prettier, Playwright pour le test UI de bout en bout.
- **Infra** : Docker (image non-root), Docker Compose, GitHub Actions.

---

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS / Android / web)
│  ├─ App.tsx                   Providers : thème, React Query, feedback, auth
│  ├─ src/api/                  Client HTTP (refresh auto), endpoints typés, messages d'erreur
│  ├─ src/hooks/                Session, requêtes React Query, temps réel (WebSocket)
│  ├─ src/navigation/           Auth → onboarding → onglets + écrans empilés
│  ├─ src/screens/              auth/, onboarding/, main/ (découverte, matchs, chat…), settings/
│  ├─ src/components/           SwipeCard, PhotoGrid, SafetySheet, MatchCelebration…
│  ├─ src/ui/ · src/theme/      Design system (palettes claire/sombre, typographie, primitives)
│  ├─ src/feedback/             Toasts, dialogues de confirmation, bottom sheets
│  ├─ e2e/                      Fixtures, test UI Playwright (web)
│  └─ scripts/                  Smoke test API, lancement sur téléphone
├─ services/api/
│  ├─ cmd/api/                  Point d'entrée (composition) + tests d'intégration HTTP
│  ├─ cmd/seed/                 Données de démonstration
│  └─ internal/
│     ├─ platform/              config, db (migrations), middleware, httpx, media, realtime,
│     │                         ratelimit, mailer, people (âge, distance), logger, handlers
│     └─ features/              auth, users (suppression de compte), profiles, photos,
│                               discovery, matching, chat, safety, notifications, realtime
├─ docker-compose.yml           Postgres + API (infra/docker-compose.yml l'inclut)
├─ scripts/qa.sh · qa.ps1       Porte qualité complète
└─ docs/ARCHITECTURE.md         Détails d'architecture et modèle de données
```

Chaque feature de l'API suit `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`) : pas de SQL dans les handlers, logique métier dans les services, règles d'autorisation vérifiées côté serveur (souvent directement dans la requête SQL). Les features ne s'importent pas entre elles : elles dépendent de petites interfaces (`Notifier`, `URLSigner`, `realtime.Publisher`) câblées dans `cmd/api/main.go`.

Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) pour le modèle de données, l'algorithme de découverte et le temps réel.

---

## Installation

Prérequis : **Node.js 20+**, **Go 1.24+**, **Docker** (ou un PostgreSQL 16 local).

```bash
git clone <ce dépôt> lueur && cd lueur
cp .env.example .env
# Générez un secret fort et collez-le dans JWT_SECRET :
openssl rand -hex 32
```

### Variables d'environnement (API)

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | **oui** | — | Secret HMAC des access tokens (≥ 32 caractères aléatoires, les valeurs d'exemple sont refusées) |
| `DATABASE_URL` | **oui** | — | Chaîne de connexion PostgreSQL |
| `APP_ENV` | non | `development` | `development` ou `production` (mode release, e-mails non journalisés) |
| `PORT` | non | `8080` | Port HTTP |
| `MEDIA_DIR` | non | `./data/media` | Dossier de stockage des photos |
| `MEDIA_SIGNING_SECRET` | non | dérivé de `JWT_SECRET` | Clé dédiée des URLs de photos signées (≥ 32 caractères) |
| `ALLOWED_ORIGINS` | non | vide | Origines CORS autorisées (web), séparées par des virgules |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies dont `X-Forwarded-For` est fiable (**indispensable derrière un proxy**, sinon tous les clients partagent la même limite de débit) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | port `587` | Envoi des codes de réinitialisation. Sans SMTP en développement, le code est affiché dans les logs de l'API |
| `POSTGRES_PASSWORD` | non | `postgres` | Utilisé par Docker Compose |

Côté application : `EXPO_PUBLIC_API_URL` (URL publique de l'API, **jamais de secret** côté frontend). En développement, l'app utilise par défaut `http://localhost:18080` (iOS / web) ou `http://10.0.2.2:18080` (émulateur Android).

---

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et **appliquées automatiquement au démarrage** de l'API (et du seed) :

- chaque fichier est exécuté une seule fois, dans une transaction, et enregistré dans `schema_migrations` ;
- un verrou consultatif PostgreSQL empêche deux instances de migrer en même temps ;
- pour faire évoluer le schéma, ajoutez un fichier `NNN_description.sql` (ne modifiez jamais une migration déjà déployée).

Tables principales : `users`, `sessions`, `password_reset_codes`, `profiles`, `preferences`, `photos`, `interests`, `user_interests`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`.

## Données de démonstration

```bash
cd services/api
export DATABASE_URL='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable'
go run ./cmd/seed                          # 24 profils de démo autour de Paris
go run ./cmd/seed -like vous@exemple.fr    # ces profils vous ont déjà « liké » : matchs immédiats
go run ./cmd/seed -purge                   # supprime tous les profils de démo et leurs photos
```

Les comptes de démonstration sont **impossibles à confondre avec de vrais membres** : drapeau `users.is_demo`, e-mails en `@demo.lueur.invalid` (domaine réservé qui ne peut recevoir aucun e-mail), bio préfixée « [Profil de démonstration] », photos illustrées « DEMO » et badge « Démo » dans l'application. Le seed refuse de s'exécuter si `APP_ENV=production`. Mot de passe des comptes de démo : `DemoPassword1`. Avec Docker : `docker compose exec api ./seed`.

---

## Lancement local

### 1. API + base avec Docker

```bash
export JWT_SECRET="$(openssl rand -hex 32)"
docker compose up --build -d
curl http://localhost:18080/health   # {"status":"ok"}
```

L'API est exposée sur le port hôte **18080** (Postgres sur `127.0.0.1:5432`).

### 1 bis. API sans Docker

```bash
cd services/api
export DATABASE_URL='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export PORT=18080 ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api
```

### 2. Application

```bash
cd apps/mobile
npm ci
npm run web       # navigateur
npm run ios       # simulateur iOS
npm run android   # émulateur Android
```

**Sur un téléphone physique**, n'utilisez jamais `localhost` : l'API doit être joignable via l'IP LAN de votre machine.

```bash
npm run start:phone            # détecte l'IP LAN et définit EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080
# ou manuellement :
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npx expo start
```

Avant de déclarer l'environnement prêt, vérifiez qu'aucun autre service n'occupe le port : `docker compose ps`, `netstat -ano | findstr :18080` (Windows) ou `lsof -i :18080`, puis `curl http://<IP_LAN>:18080/health`. En cas de conflit, changez le port hôte dans `docker-compose.yml` (par exemple `18090:8080`) et alignez `EXPO_PUBLIC_API_URL`, `scripts/start-phone.js` et ce README.

---

## Tests

```bash
./scripts/qa.sh                                    # tout : format, vet, tests Go, build, typecheck, lint, format front
E2E_API_URL=http://localhost:18080 ./scripts/qa.sh # + smoke test API et test UI de bout en bout
# Windows : powershell -ExecutionPolicy Bypass -File .\scripts\qa.ps1
```

Détail :

| Commande | Contenu |
| --- | --- |
| `cd services/api && go test ./...` | Tests unitaires (validation, âge, distance, rate limiter, traitement d'image, URLs signées, tickets temps réel…). Avec `DATABASE_URL` (ou `DATABASE_URL_TEST`) : tests d'intégration HTTP sur une vraie base — auth et révocation, onboarding et photos, découverte mutuelle, like / match / doublons, likes concurrents, chat et permissions, blocage, signalement et modération, confidentialité, isolation multi-tenant, suppression de compte, limitation de débit, WebSocket |
| `cd apps/mobile && npm run typecheck && npm run lint && npm run format:check` | TypeScript strict, ESLint, Prettier |
| `npm run e2e:api` | Parcours critique complet contre une API réelle (inscription → profil → photo → découverte → like → match → chat → notifications → blocage → suppression) |
| `npm run build:web && npm run e2e:web` | Même parcours **dans l'interface**, sur Chromium via Playwright (upload de photo réel, match, chat temps réel, blocage, suppression de compte, mode sombre). Captures dans `apps/mobile/e2e/reports/` |

Le test UI tourne sur la version web : l'application native partage exactement le même code (seules les primitives de plateforme diffèrent). Avant une mise en production mobile, faites un smoke test sur appareil réel : inscription, connexion, découverte, match, chat, notifications.

## Build

- **API** : `docker build -t lueur-api ./services/api` (image Alpine, utilisateur non-root, binaires `api` et `seed`), ou `go build ./cmd/api`.
- **Web** : `EXPO_PUBLIC_API_URL=https://api.exemple.fr npm run build:web` → dossier statique `apps/mobile/dist`.
- **iOS / Android** : via [EAS Build](https://docs.expo.dev/build/introduction/) (`npx eas build`), en définissant `EXPO_PUBLIC_API_URL` dans le profil de build.

## Déploiement

1. PostgreSQL 16 managé (sauvegardes activées) et `DATABASE_URL` avec `sslmode=require`.
2. L'image API derrière un reverse proxy HTTPS (TLS obligatoire : tokens et photos transitent par l'API). Définissez `APP_ENV=production`, `JWT_SECRET`, `MEDIA_SIGNING_SECRET`, `TRUSTED_PROXIES`, `ALLOWED_ORIGINS` et le SMTP.
3. Un **volume persistant** pour `MEDIA_DIR` (ou remplacez `media.Storage` par un stockage objet, voir ci-dessous).
4. Le proxy doit laisser passer les WebSockets sur `/realtime/ws` (en-têtes `Upgrade`/`Connection`).
5. Une seule instance d'API tant que le hub temps réel et le rate limiter sont en mémoire (voir « Évolutions »).

---

## Principales décisions techniques

- **Partir du boilerplate existant** : l'architecture Go par features et la session à refresh tokens étaient saines ; elles ont été conservées et étendues. Les modules hors produit (facturation Stripe — l'app est gratuite —, posts/commentaires/votes, stub `files` sans upload réel, annuaire `/users` qui exposait les e-mails de tous les membres) ont été supprimés, ainsi que Tamagui, inutilisé.
- **Autorisation dans le SQL** : l'accès à une conversation passe par un match actif sans blocage ; un profil n'est visible que s'il est découvrable ou matché, et jamais en cas de blocage. Tout ce qui est interdit répond `404`, pour ne rien révéler.
- **Match sans course** : like, création du match, annulation et blocage d'une même paire sont sérialisés par un verrou consultatif transactionnel ; contraintes d'unicité `(actor, target)` et `(user_low, user_high)` en filet de sécurité. Un swipe n'est jamais répétable (`409`).
- **Recommandation transparente** : filtres stricts et **mutuels** (genre, âge, rayon de distance des deux membres, organisation, pause, modération, photo requise, blocages, déjà vus) puis score pondéré (a déjà liké, activité récente, proximité, intérêts communs). La requête est isolée dans `discovery/repository.go` pour pouvoir évoluer (PostGIS, embeddings…) sans toucher au reste.
- **Vie privée de la localisation** : coordonnées arrondies à ~1 km sur l'appareil **et** sur le serveur, jamais renvoyées à quiconque ; distance affichée arrondie (au km sous 10 km, par 5 km au-delà, minimum 1 km) et masquable ; permission « position approximative » uniquement sur Android.
- **Photos sûres** : type vérifié sur le contenu (pas sur l'extension), protection contre les bombes de décompression, redimensionnement, **ré-encodage JPEG qui supprime les métadonnées EXIF (dont le GPS)** et neutralise les fichiers polyglottes ; noms de fichiers aléatoires, URLs signées HMAC expirant après 1 à 2 h ; compression sur l'appareil avant envoi.
- **Sessions révocables** : access token de 15 min, mais chaque requête vérifie que la session est active, donc déconnexion, réinitialisation du mot de passe et suppression de compte sont effectives immédiatement (WebSockets compris). Comparaison bcrypt factice pour les e-mails inconnus (pas d'énumération par le temps de réponse) ; la demande de réinitialisation répond toujours `202`.
- **Temps réel** : WebSocket en lecture seule (toutes les écritures passent par l'API REST validée), authentifié par un ticket à usage unique valable 30 s (les navigateurs ne peuvent pas envoyer d'en-tête `Authorization` sur un WebSocket). Le client se reconnecte avec backoff exponentiel et repasse en polling tant que la connexion est coupée.
- **Modération** : un signalement bloque automatiquement ; au 3ᵉ signalement distinct ouvert (émis par des comptes de plus de 24 h, pour empêcher des comptes jetables de faire disparaître quelqu'un), le profil est masqué de la découverte en attendant une revue (`reports.status`). Les signalements survivent à la suppression des comptes (`ON DELETE SET NULL`).
- **Suppression de compte** : profil, photos (fichiers compris), likes, matchs, conversations, messages, sessions et notifications sont effacés dans une transaction ; seuls les signalements anonymisés sont conservés.
- **Anti-abus, pas de quota** : limites de débit généreuses par IP (auth) et par membre (swipes 120/min, messages 40/min, uploads 30/h, signalements 10/h, changements de position 30/h, suppression de compte et changement de mot de passe 10/h) uniquement pour bloquer les robots. Au plus 3 codes de réinitialisation par compte et par heure, 5 essais par code (compteur atomique). Au plus 5 WebSockets par membre ; chaque socket revérifie sa session toutes les 25 s et la déconnexion ferme celles de l'appareil.
- **Stockage des tokens** : Keychain / Keystore via `expo-secure-store` sur mobile, `sessionStorage` sur le web (effacé à la fermeture de l'onglet).
- **Swipe sans dépendance native** : `Animated` + `PanResponder`, avec boutons équivalents pour l'accessibilité et le web.
- **Langue** : interface en français ; le code et les commentaires restent en anglais.

### Limites connues

- Un profil découvrable affiche son âge et sa distance arrondie à tout membre de la même organisation qui connaît son identifiant (même hors préférences mutuelles). La grille de ~1 km, l'arrondi de la distance et la limite de changements de position rendent la trilatération grossière, mais elle reste possible à l'échelle du kilomètre ; masquez la distance pour l'éviter.
- Sans position partagée, la découverte n'applique aucun filtre de distance.
- Les listes de matchs (100) et de conversations (50) chargent une première page ; la pagination de l'API est prête pour un défilement infini.

### Évolutions prévues par l'architecture

- Plusieurs instances d'API : remplacer le hub en mémoire (`platform/realtime`) par Postgres `LISTEN/NOTIFY` ou Redis derrière l'interface `Publisher`, et le rate limiter par un stockage partagé.
- Stockage objet (S3, GCS) : implémenter `media.Storage`.
- Notifications push natives (Expo Push) : brancher un second canal dans `notifications.Service.Notify`.
- Outil de modération : les tables `reports` et `profiles.moderation_hidden` sont prêtes pour une interface d'administration.

---

## Référence rapide de l'API

| Méthode et route | Rôle |
| --- | --- |
| `GET /health` | État de l'API et de la base |
| `POST /auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` | Session |
| `POST /auth/password/forgot` · `/auth/password/reset` | Réinitialisation par code |
| `GET /me` · `PUT /me/password` · `DELETE /me` | Compte |
| `GET/PUT /me/profile` · `PUT /me/preferences` · `PUT /me/privacy` · `PUT/DELETE /me/location` · `PUT /me/interests` · `GET /interests` | Profil |
| `GET/POST /me/photos` · `PUT /me/photos/order` · `PUT/DELETE /me/photos/:id` · `GET /media/photos/:id?exp&sig` | Photos |
| `GET /discovery` · `GET /profiles/:userId` | Découverte |
| `POST /swipes` · `GET /matches` · `DELETE /matches/:userId` | Matching |
| `GET /chats` · `GET/POST /chats/:userId/messages` · `POST /chats/:userId/read` | Chat |
| `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` | Sécurité |
| `GET /notifications` · `GET /notifications/unread-count` · `POST /notifications/:id/read` · `POST /notifications/read-all` | Notifications |
| `POST /realtime/ticket` · `GET /realtime/ws?ticket=` | Temps réel |

Les erreurs ont toujours la forme `{"error": "message", "code": "code_stable"}`.

## Documentation complémentaire

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- [CONTRIBUTING.md](CONTRIBUTING.md) · [CHANGELOG.md](CHANGELOG.md) · [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [AGENTS.md](AGENTS.md) (règles pour les agents de développement)
