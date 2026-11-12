# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre mobile (iOS, Android, et web pour le développement) **entièrement gratuite** : pas d'abonnement, pas de paywall, pas de « boost », pas de likes limités, pas de publicité. Toutes les fonctionnalités sont ouvertes à tout le monde.

Deux personnes peuvent : créer un compte → créer leur profil (prénom, âge, genre, photos, bio, métier, centres d'intérêt, intention) → découvrir des profils compatibles → liker ou passer → matcher quand l'intérêt est réciproque → discuter en temps réel (accusés de lecture, non-lus) → recevoir des notifications → régler préférences et confidentialité → bloquer, signaler → supprimer leur compte et leurs données.

| Découverte | Match | Conversation | Profil |
| --- | --- | --- | --- |
| Cartes à swiper (ou boutons accessibles), navigation entre photos, distance approximative | Détection serveur, sans doublon, même en cas de likes simultanés | WebSocket temps réel, pagination, « Vu », masquage local | Photos (6 max), préférences réciproques, pause du profil, localisation arrondie |

---

## Sommaire

1. [Stack technique](#stack-technique)
2. [Architecture](#architecture)
3. [Démarrage rapide (Docker)](#démarrage-rapide-docker)
4. [Variables d'environnement](#variables-denvironnement)
5. [Base de données, migrations, seed](#base-de-données-migrations-et-seed)
6. [Lancer l'app mobile](#lancer-lapplication-mobile)
7. [Tests et qualité](#tests-et-qualité)
8. [Build et déploiement](#build-et-déploiement)
9. [API](#api)
10. [Sécurité et vie privée](#sécurité-et-vie-privée)
11. [Décisions techniques](#décisions-techniques)

---

## Stack technique

| Couche | Choix |
| --- | --- |
| Mobile | Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 6, TanStack Query 5, expo-image, expo-image-picker, expo-location, expo-secure-store |
| API | Go 1.24, Gin, PostgreSQL 16 via `pgxpool` (SQL explicite, pas d'ORM), JWT HS256, bcrypt, WebSocket (`coder/websocket`), traitement d'images (`golang.org/x/image`) |
| Infra | Docker, Docker Compose, GitHub Actions (CI + E2E) |
| Qualité | `gofmt`, `go vet`, `go test -race`, ESLint (`eslint-config-expo`), Prettier, `tsc`, tests Node natifs, Playwright (E2E web), Maestro (E2E natif) |

## Architecture

```text
.
├─ apps/mobile/                 # Client Expo (iOS / Android / web)
│  ├─ App.tsx                   # Providers : safe area, thème, auth/React Query, toasts
│  ├─ src/api/                  # SEUL accès réseau : client (refresh auto), endpoints typés, types
│  ├─ src/hooks/                # useAuth (session), queries (React Query), useRealtime (WebSocket)
│  ├─ src/navigation/           # Auth → Onboarding → App (onglets + piles)
│  ├─ src/screens/              # auth, onboarding, discover, matches (chat), profile, me, notifications
│  ├─ src/components/           # PhotoGrid, PreferencesForm, InterestPicker, SafetySheet, localisation
│  ├─ src/ui/ + src/theme/      # Design system (clair/sombre) : Text, Button, TextField, Chip, Photo…
│  └─ e2e/                      # Smoke API, E2E navigateur (Playwright), flux natif (Maestro)
├─ services/api/
│  ├─ cmd/api/                  # Composition root + tests d'intégration HTTP
│  ├─ cmd/seed/                 # Données de démonstration (refusé en production)
│  └─ internal/
│     ├─ platform/              # config, db (migrations suivies), middleware (auth, CORS, headers),
│     │                         # httpx, ratelimit, media (stockage + URLs signées + traitement),
│     │                         # realtime (hub WebSocket), mailer, logger
│     └─ features/              # handler → service → repository (+ model, routes) par domaine
│        ├─ auth                # inscription, connexion, refresh, mot de passe, suppression de compte
│        ├─ profiles            # profil, préférences, centres d'intérêt, localisation, confidentialité
│        ├─ photos              # upload, remplacement, ordre, suppression
│        ├─ matching            # découverte, swipes, matchs, unmatch
│        ├─ chat                # conversations (réservées aux matchs), messages, lu/non-lu
│        ├─ safety              # blocages, signalements
│        ├─ notifications       # notifications in-app + push temps réel
│        └─ users, posts, comments, files   # modules génériques hérités du boilerplate
├─ docker-compose.yml
├─ scripts/qa.sh                # Porte qualité locale (identique à la CI)
└─ docs/ARCHITECTURE.md         # Détails : modèle de données, flux, choix
```

Règles respectées : un seul serveur HTTP (Gin), handlers minces, logique métier dans les services, SQL uniquement dans les repositories, pool PostgreSQL unique, aucune requête N+1 (profils, photos et centres d'intérêt chargés par lots), pagination partout. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Démarrage rapide (Docker)

Prérequis : Docker, Node.js ≥ 22.6, (optionnel) Go 1.24 pour développer l'API hors Docker.

```bash
cp .env.example .env
# Générez un secret fort et collez-le dans JWT_SECRET :
openssl rand -hex 32

docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# Données de démonstration (facultatif, jamais en production) :
docker compose exec api ./seed
```

L'API écoute sur `http://localhost:18080` (port hôte 18080 → 8080 dans le conteneur). Les photos sont stockées dans le volume Docker `uploads`, la base dans `postgres_data`.

Compte de démo après le seed : `demo@demo.lueur.invalid` / `Demo12345`. Plusieurs membres de démo l'ont déjà liké : liker en retour crée des matchs immédiatement.

## Variables d'environnement

API (`.env` à la racine, lu par Docker Compose — voir [.env.example](.env.example)) :

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | **oui** | — | ≥ 32 caractères aléatoires. Signe les jetons d'accès ; des clés dérivées signent les URLs média et les codes de réinitialisation. Les valeurs de type « change-me » sont refusées. |
| `DATABASE_URL` | oui (hors Docker) | — | Chaîne de connexion PostgreSQL. |
| `PORT` | non | `8080` | Port d'écoute de l'API. |
| `APP_ENV` | non | `development` | `development` ou `production`. En dev sans SMTP, les codes de réinitialisation sont écrits dans les logs ; jamais en production. |
| `UPLOAD_DIR` | non | `./data/uploads` | Dossier des photos traitées. |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS + WebSocket), ex. `http://localhost:8081`. Inutile pour les apps natives. |
| `TRUSTED_PROXIES` | non | vide | Proxies autorisés à fournir `X-Forwarded-For` (sinon l'IP réelle est utilisée pour le rate limiting). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | port `587` | Envoi des codes de réinitialisation de mot de passe. |
| `POSTGRES_PASSWORD` | non | `postgres` | Mot de passe Postgres utilisé par Docker Compose. |

Mobile ([apps/mobile/.env.example](apps/mobile/.env.example)) : `EXPO_PUBLIC_API_URL` — URL publique de l'API (aucun secret côté client). Sur un téléphone physique, utilisez **l'IP LAN de votre ordinateur**, jamais `localhost` (`npm run start:phone` la détecte automatiquement).

Ne commitez jamais `.env` (il est ignoré par git).

## Base de données, migrations et seed

- Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées **automatiquement au démarrage**, chacune dans une transaction, suivies dans `schema_migrations` et protégées par un verrou consultatif (plusieurs instances peuvent démarrer en même temps).
- Ajouter une migration : créer `0NN_description.sql` (numéro supérieur au dernier). Une migration appliquée ne doit plus être modifiée.
- Tables principales : `users`, `sessions`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (likes et passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`, `password_resets`.
- Seed de démonstration : `go run ./cmd/seed` (ou `docker compose exec api ./seed`) crée 12 membres fictifs ; `-reset` les supprime. Les comptes de démo utilisent le domaine réservé `.invalid` (ne peut pas exister), leur bio commence par `[Démo]`, leurs photos sont des illustrations abstraites, et l'outil **refuse de s'exécuter si `APP_ENV=production`**.

## Lancer l'application mobile

```bash
cd apps/mobile
npm ci
cp .env.example .env               # puis ajustez EXPO_PUBLIC_API_URL
npm run start:phone                # téléphone physique (Expo Go) : IP LAN détectée automatiquement
npm start                          # émulateurs : http://10.0.2.2:18080 (Android) / localhost (iOS) par défaut
npm run web                        # navigateur (ajoutez ALLOWED_ORIGINS=http://localhost:8081 côté API)
```

Avant de déclarer l'app « prête » sur téléphone : `docker compose ps`, vérifier que le port 18080 n'est pas pris par un autre service, puis `curl http://<IP_LAN>:18080/health`.

## Tests et qualité

Tout en une commande (identique à la CI) :

```bash
DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable ./scripts/qa.sh
API_URL=http://localhost:18080 ./scripts/qa.sh     # + smoke test sur l'API en marche
```

| Niveau | Commande | Couvre |
| --- | --- | --- |
| Unitaires Go | `cd services/api && go test ./internal/...` | règles d'âge, préférences, arrondis de distance/coordonnées, validation, mots de passe, JWT (alg/exp/session), traitement d'image (bombes, EXIF, types), URLs signées, rate limiter, hub temps réel |
| Intégration Go (Postgres) | `DATABASE_URL_TEST=… go test -p 1 ./...` | auth + révocation de session, reset de mot de passe (verrouillage), onboarding, photos, découverte, like/pass, match unique (y compris likes **simultanés**), chat (permissions, pagination, lu/non-lu, masquage), blocage, signalement, annuaire privé, suppression de compte, WebSocket, rate limiting |
| Mobile | `npm run typecheck && npm run lint && npm run format:check && npm test` | TypeScript strict, ESLint (0 warning), Prettier, tests unitaires des utilitaires |
| Smoke API | `npm run e2e:api` | parcours complet contre une API réelle |
| E2E UI (navigateur) | `npm run build:web`, servir `dist/` sur :8081, puis `npm run e2e:web` | Inscription → onboarding → découverte → like → match → conversation temps réel (message entrant + « Vu ») → suppression du compte, avec captures d'écran |
| E2E natif | `maestro test e2e/maestro/critical-flow.yaml` | même parcours sur iOS/Android |

Sans base de données, les tests d'intégration Go sont ignorés (skip) ; en CI ils s'exécutent contre un service PostgreSQL.

## Build et déploiement

- **API** : `docker build -t lueur-api services/api` (image Alpine, utilisateur non-root, binaires `api` et `seed`). Déployez derrière un reverse proxy TLS (HSTS automatique en HTTPS), avec `APP_ENV=production`, un `JWT_SECRET` dédié, un SMTP, `TRUSTED_PROXIES` réglé sur votre proxy, et un volume persistant pour `UPLOAD_DIR`. Le proxy doit laisser passer les WebSockets sur `/ws`.
- **Mise à l'échelle** : le rate limiting et le hub temps réel sont en mémoire (une instance). Pour plusieurs instances, brancher un backend partagé (Redis ou Postgres `LISTEN/NOTIFY`) derrière les interfaces `ratelimit.Limiter` et `realtime.Publisher`, et un stockage objet derrière `media.Storage`.
- **Mobile** : `eas build` (Expo Application Services) avec `EXPO_PUBLIC_API_URL` pointant vers l'API HTTPS de production. Identifiants : `app.lueur.mobile`.
- **Web** (démo/QA) : `npm run build:web` produit un site statique dans `apps/mobile/dist`.

## API

Toutes les routes (sauf `/health`, `/auth/*` publiques, `/media/*` signées) exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "..."}`.

| Domaine | Routes |
| --- | --- |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/password/forgot`, `/auth/password/reset` · `GET /me` · `PUT /me/password` · `DELETE /me` |
| Profil | `GET/PUT /profile` · `PUT /profile/preferences`, `/profile/interests`, `/profile/privacy` · `PUT/DELETE /profile/location` · `GET /interests` · `GET /profiles/:userId` |
| Photos | `GET/POST /profile/photos` · `PUT /profile/photos/order` · `PUT/DELETE /profile/photos/:photoId` · `GET /media/:key?exp&sig` |
| Rencontres | `GET /discover?limit&exclude` · `POST /swipes` · `GET /matches` · `DELETE /matches/:matchId` |
| Chat | `GET /chats`, `/chats/unread` · `GET/POST /chats/:userId/messages` (`?before=<id>`) · `POST /chats/:userId/read` · `DELETE /chats/:userId` |
| Sécurité | `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` |
| Notifications | `GET /notifications`, `/notifications/unread-count` · `POST /notifications/read-all`, `/notifications/:id/read` |
| Temps réel | `GET /ws` (WebSocket ; premier message `{"type":"auth","token":"…"}`) → `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new` |

## Sécurité et vie privée

- **Authentification** : bcrypt, jetons d'accès HS256 de 15 min (algorithme, expiration et claims vérifiés strictement), refresh tokens opaques hachés en base et **rotatifs**, session vérifiée à chaque requête (déconnexion, reset et suppression révoquent immédiatement). Comparaison bcrypt factice si l'email n'existe pas (pas d'énumération par le temps de réponse).
- **Autorisations côté serveur** : un chat n'existe que via un match ; un tiers reçoit `404` (on ne confirme jamais l'existence d'une conversation). Profils visibles seulement si non bloqués et découvrables (ou déjà matchés). Les décisions de match, l'âge et les préférences sont recalculés côté serveur.
- **Données personnelles** : l'email n'est jamais exposé aux autres membres ; la date de naissance n'est jamais renvoyée (seulement l'âge, masquable) ; les coordonnées sont arrondies à ~1 km sur l'appareil **et** côté serveur, jamais renvoyées, la distance est arrondie par paliers, et les mises à jour de position sont limitées (anti-triangulation). Aucune localisation en arrière-plan (permission « approximative » sur Android).
- **Photos** : type réel détecté sur les octets (JPEG/PNG/WebP), 10 Mo max, dimensions contrôlées avant décodage (anti-bombe), orientation EXIF appliquée puis **ré-encodage JPEG** qui supprime toutes les métadonnées (dont le GPS) et tout contenu caché ; noms de fichiers aléatoires, chemins validés, URLs signées HMAC expirantes.
- **Abus** : rate limiting (global par IP, auth, swipes, messages, uploads, signalements, localisation) ; uniquement anti-robots — aucun quota pour un usage humain.
- **Modération** : signaler bloque automatiquement ; un seul signalement ouvert par paire ; les signalements survivent (anonymisés) à la suppression de compte. Une interface de modération n'est pas incluse : les signalements sont consultables dans la table `reports` (statuts `open`, `reviewed`, `dismissed`).
- **Suppression de compte** : immédiate et définitive (profil, photos et fichiers, préférences, swipes, matchs, conversations, sessions, notifications), confirmée par mot de passe.
- En-têtes de sécurité, CORS sur liste blanche, proxies de confiance explicites, conteneur non-root, secrets uniquement par variables d'environnement.

## Décisions techniques

- **Construire sur le boilerplate** plutôt que réécrire : l'architecture Go `handler → service → repository` et le client Expo étaient sains. Supprimé : la facturation Stripe (contraire au « 100 % gratuit »), Tamagui (installé mais inutilisé), le compose dupliqué, les scripts de smoke obsolètes. Conservé : `posts`, `comments`, `files` (modules génériques du boilerplate, testés, non exposés dans l'app).
- **Likes et passes dans une seule table `swipes`** (clé primaire acteur/cible) : impossible de répondre deux fois. **Matchs stockés une fois par paire** avec identifiants ordonnés + contrainte d'unicité, et un **verrou consultatif par paire** sérialise deux likes simultanés : exactement un match.
- **Découverte** : filtres réciproques (genre, âge, distance des deux côtés), exclusion des profils déjà vus, bloqués, en pause ou sans photo ; classement par « vous a déjà liké », centres d'intérêt communs, activité récente. L'algorithme est isolé dans `matching/repository.go#Discover` pour être remplacé facilement.
- **Annuler un match** supprime la conversation pour les deux et transforme le like en « pass » : la paire ne réapparaît jamais. Bloquer fait de même.
- **Temps réel** par WebSocket (natif dans React Native, aucune dépendance client) ; le jeton est envoyé dans le premier message (jamais dans l'URL ni les logs). Si la connexion tombe, le client repasse en polling léger.
- **Notifications** : in-app et temps réel (match, message ; une seule notification non lue par conversation). Les notifications push système (APNs/FCM via Expo) ne sont pas activées : elles nécessitent un projet EAS et des identifiants propres au déploiement.
- **Récupération de compte** par code à 6 chiffres envoyé par email (15 min, 5 essais, 3 demandes/heure, haché HMAC), sans révéler si l'email existe.
- **Date de naissance non modifiable** après création (évite de contourner la majorité), âge minimal 18 ans vérifié côté serveur.
- **Interface en français**, identité visuelle originale « Lueur » (ivoire chaud / braise, mode sombre automatique, titres serif), accessibilité (rôles, libellés, zones tactiles ≥ 44 px, alternatives aux gestes de swipe).

## Licence et contribution

Voir [CONTRIBUTING.md](CONTRIBUTING.md) et [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
