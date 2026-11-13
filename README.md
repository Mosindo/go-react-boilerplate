# Aimant

**Aimant** est une application de rencontre moderne, **100 % gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de likes limités, pas de publicité. Toutes les fonctionnalités sont accessibles à tous les membres.

Deux personnes peuvent : créer un compte, compléter leur profil (photos, âge, genre, préférences, bio, centres d'intérêt), découvrir des profils proches, liker ou passer, obtenir un match lorsque l'intérêt est réciproque, discuter en temps réel, recevoir des notifications, gérer leur confidentialité, bloquer ou signaler quelqu'un et supprimer leur compte.

Le dépôt contient l'API Go (`services/api`) et l'application Expo / React Native (`apps/mobile`, iOS, Android et web).

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Démarrage rapide (Docker)](#démarrage-rapide-docker)
5. [Lancement local sans Docker](#lancement-local-sans-docker)
6. [Variables d'environnement](#variables-denvironnement)
7. [Base de données et migrations](#base-de-données-et-migrations)
8. [Données de démonstration](#données-de-démonstration)
9. [Application mobile](#application-mobile)
10. [Tests et qualité](#tests-et-qualité)
11. [Build et déploiement](#build-et-déploiement)
12. [Référence de l'API](#référence-de-lapi)
13. [Sécurité et vie privée](#sécurité-et-vie-privée)
14. [Décisions techniques](#décisions-techniques)
15. [Limites connues et évolutions](#limites-connues-et-évolutions)

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation des refresh tokens, récupération par code à 6 chiffres envoyé par e-mail, déconnexion, suppression définitive du compte (mot de passe requis) |
| Onboarding | Assistant en 5 étapes : prénom + date de naissance (18 ans minimum, vérifié côté serveur), genre et personnes recherchées, photos, localisation, bio / objectif / centres d'intérêt. Reprise automatique à la première étape manquante |
| Profil | Prénom, âge, jusqu'à 6 photos ordonnées (photo principale), bio, métier, taille, objectif de rencontre, ville, 42 centres d'intérêt (10 max) |
| Photos | Upload, compression côté appareil, validation et ré-encodage côté serveur (suppression EXIF/GPS), réordonnancement, photo principale, remplacement, suppression, URLs signées à durée limitée |
| Découverte | Deck de cartes avec swipe (et boutons accessibles), navigation entre photos, profil complet, distance approximative, intérêts en commun |
| Matching | Like / pass, match uniquement sur like réciproque, aucun doublon, aucune réapparition de profils déjà traités, annulation de match |
| Chat | Conversations après match, temps réel (WebSocket), heure d'envoi, accusés de lecture (« Vu »), compteurs non lus, pagination, envoi optimiste, effacement local de la conversation |
| Notifications | Notifications in-app (match, nouveaux messages regroupés par conversation) poussées en temps réel, badges, « tout marquer comme lu » |
| Préférences | Genres recherchés, tranche d'âge, distance maximale |
| Confidentialité | Mise en pause du profil (invisible et sans nouveaux matchs, conversations conservées), masquage de la distance, position arrondie à ~1 km, jamais d'e-mail / date de naissance / coordonnées exposés |
| Sécurité | Blocage / déblocage, signalement (bloque automatiquement), masquage en attente de vérification après 2 signalements « mineur » distincts ou 3 signalements distincts ; un profil masqué ne peut plus découvrir, matcher ni écrire |
| UI | Identité visuelle originale (terracotta, crème, prune), mode sombre automatique, états de chargement / vide / erreur, toasts, accessibilité (labels, rôles, réduction des animations) |

## Stack technique

**API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool` (pool unique), SQL explicite sans ORM, JWT HS256 (access 15 min) + refresh tokens opaques stockés hachés, bcrypt, `gorilla/websocket`, `golang.org/x/image` (redimensionnement / WebP).

**Mobile / web** : Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 6 (stack + onglets), TanStack Query 5, `expo-image` (cache mémoire + disque), `expo-image-picker` / `expo-image-manipulator`, `expo-location`, `expo-secure-store`.

**Outillage** : Docker / Docker Compose, GitHub Actions, ESLint (eslint-config-expo) + Prettier, tests Go (unitaires + intégration sur Postgres réel), Playwright (E2E navigateur sur le build web).

## Architecture

```text
.
├─ apps/mobile/                  Application Expo (iOS, Android, web)
│  ├─ App.tsx                    Providers : thème, toasts, session, temps réel, navigation
│  ├─ e2e/                       Smoke test API + E2E Playwright du parcours critique
│  └─ src/
│     ├─ api/                    Client HTTP (refresh auto), endpoints typés, WebSocket
│     ├─ state/                  Session, React Query, pont temps réel -> cache
│     ├─ navigation/             Pile auth / onboarding / application + onglets
│     ├─ features/               auth, onboarding, discover, chat, notifications, profile, settings
│     ├─ ui/                     Design system (Text, Button, TextField, Chip, Sheet, Photo…)
│     ├─ theme/                  Palettes clair / sombre
│     └─ utils/                  Erreurs traduites, formats, préparation des photos
├─ services/api/
│  ├─ cmd/api/                   Point d'entrée + tests d'intégration HTTP
│  ├─ cmd/seed/                  Générateur de profils de démonstration
│  └─ internal/
│     ├─ platform/               config, db (migrations), middleware, httpx, ratelimit,
│     │                          storage, media (URLs signées), realtime (hub), mailer, logger
│     └─ features/               auth, profiles, photos, matching, chat, notifications,
│                                safety, realtime
├─ scripts/e2e-local.sh          Suite complète en une commande
├─ docker-compose.yml
└─ docs/ARCHITECTURE.md          Détails d'architecture et modèle de données
```

Chaque feature backend suit `handler.go -> service.go -> repository.go` (+ `model.go`, `routes.go`) : handlers fins, règles métier dans les services, SQL uniquement dans les repositories. Côté mobile, aucun écran n'appelle `fetch` : tout passe par `src/api`.

Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) pour le modèle de données, le flux de matching et le temps réel.

## Démarrage rapide (Docker)

Prérequis : Docker, Node.js 20.19+ et npm.

```bash
cp .env.example .env
# Générez un secret fort et collez-le dans JWT_SECRET :
openssl rand -base64 48

docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# (optionnel) profils de démonstration autour de Paris
docker compose exec api ./seed -count 24
```

Puis lancez l'application :

```bash
cd apps/mobile
npm ci
npm run web          # navigateur : http://localhost:8081
npm run start        # Expo Go / émulateurs
```

En développement, le code de réinitialisation de mot de passe est affiché dans les logs de l'API (`docker compose logs api`).

## Lancement local sans Docker

```bash
# PostgreSQL 16 doit tourner localement
createdb app

cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export PORT=18080 ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api
```

## Variables d'environnement

Le fichier [`.env.example`](.env.example) liste toutes les variables. Aucun secret n'est commité ; `.env` est ignoré par git.

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `DATABASE_URL` | oui | — | Connexion PostgreSQL |
| `JWT_SECRET` | oui | — | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées). Sert aussi à dériver la clé des URLs média |
| `APP_ENV` | non | `development` | `development`, `test` ou `production` (SMTP obligatoire en production) |
| `PORT` | non | `8080` | Port HTTP de l'API |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS + WebSocket), séparées par des virgules |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies dont `X-Forwarded-For` est fiable (rate limiting) |
| `UPLOAD_DIR` | non | `./data/uploads` | Stockage des photos |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `MAIL_FROM` | en production | — | Envoi des codes de récupération |
| `EXPO_PUBLIC_API_URL` | build mobile | émulateur | URL publique de l'API embarquée dans l'app (jamais de secret) |

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées **au démarrage** de l'API (et du seed). Elles sont suivies dans `schema_migrations`, exécutées chacune dans une transaction, et protégées par un verrou consultatif PostgreSQL : plusieurs instances peuvent démarrer en même temps sans conflit.

Tables principales : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (likes/passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`.

Contraintes notables : unicité `(swiper, cible)` sur les swipes, une seule ligne de match par paire (`user_a_id < user_b_id`), unicité `(user_id, position)` des photos (différée pour le réordonnancement), CHECK sur âge / distance / genres / longueurs, suppression en cascade à la suppression du compte. Les index couvrent la découverte, les conversations, les messages, les notifications et les signalements.

Pour ajouter une migration : créez `NNN_description.sql` (numéro supérieur au dernier), elle sera appliquée au prochain démarrage.

## Données de démonstration

```bash
cd services/api
DATABASE_URL=... go run ./cmd/seed -count 24 -lat 48.8566 -lon 2.3522 -city Paris
# Faire liker un vrai compte par tous les profils démo (pour tester les matchs seul) :
DATABASE_URL=... go run ./cmd/seed -like-user vous@exemple.fr
# Supprimer toutes les données de démonstration :
DATABASE_URL=... go run ./cmd/seed -purge
```

Les comptes démo sont clairement séparés : e-mails en `.invalid` (TLD réservé), mot de passe aléatoire inconnu (impossible de s'y connecter), colonne `users.is_demo = TRUE`, badge « Profil de démonstration » dans l'app, photos abstraites sans visage. Le seed refuse de s'exécuter si `APP_ENV=production`.

## Application mobile

```bash
cd apps/mobile
cp .env.example .env     # optionnel
npm ci
npm run start            # puis i (iOS), a (Android), w (web)
```

- Émulateur Android : l'app utilise `http://10.0.2.2:18080` par défaut ; simulateur iOS / web : `http://localhost:18080`.
- **Téléphone physique : n'utilisez jamais `localhost`.** Définissez `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080` ou lancez `npm run start:phone` qui détecte l'IP automatiquement.
- Avant de déclarer l'environnement prêt : `docker compose ps`, vérifiez que rien d'autre n'écoute sur 18080, puis `curl http://<IP_LAN>:18080/health`. En cas de conflit, changez le port publié dans `docker-compose.yml` et `EXPO_PUBLIC_API_URL`.
- Pour le web, l'origine (`http://localhost:8081`) doit figurer dans `ALLOWED_ORIGINS`.

## Tests et qualité

**Tout lancer en une commande** (Go, typecheck, lint, format, smoke API, E2E navigateur) sur une base jetable :

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/app_e2e?sslmode=disable' ./scripts/e2e-local.sh
```

Détail :

```bash
# API
cd services/api
gofmt -l .                                   # doit être vide
go vet ./...
DATABASE_URL_TEST=postgres://... go test ./... # unitaires + intégration (ignorés sans base)
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm run typecheck
npm run lint
npm run format:check
npm run test:e2e:api                         # smoke API (stack démarrée)
npm run test:e2e:web                         # Playwright (API + web servis, voir e2e-local.sh)
```

Couverture :

- **Unitaires Go** : validation des profils et de l'âge, arrondi de position et distance approximative, classement de la découverte, traitement d'image (types dangereux, tailles, re-encodage), URLs signées, JWT (alg, expiration, `none`), rate limiting, stockage (traversée de chemin), hub temps réel, e-mail, codes de récupération, messages, signalements.
- **Intégration Go (HTTP + Postgres)** : cycle d'authentification complet (refresh, logout, reset, suppression), rate limiting, onboarding et photos, découverte avec filtres mutuels, like / match / doublons, chat et permissions, pagination, accusés de lecture, effacement local, annulation de match, blocage, signalement, pause, likes concurrents (un seul match), WebSocket.
- **E2E navigateur (Playwright)** : inscription → onboarding complet avec upload photo et géolocalisation → découverte → like → match → conversation (envoi + réception temps réel) → liste des messages → notifications → suppression du compte.

La CI GitHub Actions (`.github/workflows/ci.yml`) exécute tout cela, ainsi que l'export des bundles web / Android / iOS et un test de santé de l'image Docker.

## Build et déploiement

**API** : l'image Docker (`services/api/Dockerfile`) est multi-étapes, tourne en utilisateur non-root et contient les binaires `api` et `seed`.

```bash
docker build -t aimant-api services/api
```

Recommandations de production :

- `APP_ENV=production`, `JWT_SECRET` long et aléatoire, SMTP configuré.
- Placer l'API derrière un reverse proxy TLS (HSTS est envoyé quand la requête est en HTTPS) et renseigner `TRUSTED_PROXIES`.
- Proxy WebSocket : transmettre les en-têtes `Upgrade`/`Connection` sur `/realtime`.
- Persister `UPLOAD_DIR` sur un volume sauvegardé.
- PostgreSQL managé avec sauvegardes ; l'API applique les migrations au démarrage.
- `ALLOWED_ORIGINS` limité à vos domaines web.

**Mobile** : `npm run build:web` produit un site statique dans `dist/` (servi par n'importe quel CDN). Pour les stores, utilisez EAS Build (`npx eas build`) avec `EXPO_PUBLIC_API_URL` pointant vers l'API HTTPS de production.

## Référence de l'API

Toutes les routes (sauf mention) exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "...", "code": "..."}`.

| Méthode | Route | Description |
| --- | --- | --- |
| GET | `/health` | Santé (public) |
| POST | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Authentification (public, rate-limité) |
| POST | `/auth/password/forgot`, `/auth/password/reset` | Récupération par code (public, rate-limité) |
| GET / DELETE | `/me` | Compte courant / suppression (body `{password}`) |
| GET / PATCH | `/me/profile` | Profil complet / mise à jour partielle (inclut pause et affichage de la distance) |
| PUT | `/me/location` | Position (arrondie à 2 décimales) + ville |
| PATCH | `/me/preferences` | Genres recherchés, âge, distance |
| GET | `/interests` | Catalogue des centres d'intérêt |
| GET / POST | `/me/photos` | Liste / upload multipart (`photo`) |
| PUT / DELETE | `/me/photos/:photoId` | Remplacer / supprimer |
| PUT | `/me/photos/order` | Réordonner (`{photoIds}`) |
| GET | `/media/photos/:photoId?exp&sig` | Image (URL signée, sans bearer) |
| GET | `/profiles/:userId` | Profil public (si visible : match actif ou découvrable, sans blocage) |
| GET | `/discovery?limit=` | Profils suivants, classés |
| POST | `/swipes` | `{targetUserId, action: like|pass}` → `{matched, match?}` |
| DELETE | `/matches/:userId` | Annuler un match |
| GET | `/chats` | Matchs actifs avec dernier message et non lus |
| GET / POST | `/chats/:userId/messages` | Messages (`?before=<id>&limit=`) / envoi |
| POST | `/chats/:userId/read` | Accusé de lecture |
| DELETE | `/chats/:userId` | Effacer la conversation pour soi |
| GET | `/notifications` | Notifications + nombre de non lues |
| POST | `/notifications/:id/read`, `/notifications/read-all` | Marquer comme lues |
| GET / POST | `/blocks` | Membres bloqués / bloquer |
| DELETE | `/blocks/:userId` | Débloquer |
| POST | `/reports` | Signaler (`{userId, reason, details}`) |
| GET (WS) | `/realtime` | WebSocket ; premier message `{"type":"auth","token":"…"}` |

Événements temps réel : `ready`, `message.created`, `conversation.read`, `match.created`, `match.removed`, `notification.created`.

## Sécurité et vie privée

- **Authentification** : JWT HS256 vérifiés strictement (algorithme, expiration obligatoire, claims), session contrôlée à chaque requête (un logout, un reset ou une suppression invalide immédiatement les tokens et ferme les WebSockets), refresh tokens opaques stockés en SHA-256, à usage unique (rotation atomique ; rejouer un ancien token révoque toute la session), bcrypt, comparaison factice pour éviter l'énumération d'e-mails par le temps de réponse.
- **Autorisation côté serveur** : chat uniquement entre membres ayant un match actif et sans blocage ; profils visibles seulement s'ils sont découvrables ou matchés ; une ressource d'autrui renvoie 404 sans divulguer son existence.
- **Validation** : toutes les entrées sont normalisées et bornées (e-mail, prénom, âge ≥ 18, longueurs, enums, pagination, UUID) ; corps JSON limités à 64 Ko.
- **Photos** : type détecté par signature binaire (le nom et le Content-Type client sont ignorés), dimensions et taille bornées (anti « decompression bomb »), ré-encodage JPEG qui supprime toutes les métadonnées (dont la position GPS), noms de fichiers aléatoires, accès uniquement par URLs HMAC expirant en 1 à 2 h (un remplacement de photo change son identifiant et invalide les anciens liens), en-têtes `nosniff` + CSP `sandbox`.
- **Localisation** : demandée seulement quand le membre appuie sur le bouton, coordonnées arrondies à ~1 km avant stockage, distance affichée arrondie (par paliers de 5 km au-delà de 10 km), masquable.
- **Abus** : rate limiting (inscription, connexion, récupération, swipes, messages, uploads, signalements), proxies de confiance explicites, codes de récupération HMAC limités à 5 essais (compteur atomique), 15 minutes et 3 codes par heure et par compte ; décodage d'images limité à 16 Mpx et 2 en parallèle.
- **Données** : la suppression du compte efface profil, photos (fichiers compris), swipes, matchs, messages, blocages, notifications et sessions ; les signalements déposés sont conservés anonymement pour la modération.
- **Mobile** : tokens dans le trousseau / keystore (`expo-secure-store`) ; sur le web, en mémoire uniquement (jamais en `localStorage`).

## Décisions techniques

- **Construire sur le boilerplate existant** : Gin + pgx + architecture par feature et Expo + React Query étaient sains ; ils ont été conservés. Les modules sans rapport avec le produit (billing Stripe, posts, commentaires, votes, fichiers « métadonnées seulement », annuaire `/users` qui exposait les e-mails) ont été supprimés, ainsi que Tamagui (configuré mais inutilisé). La migration `014` supprime leurs tables sur les bases existantes.
- **Swipes + matches séparés** : une ligne par décision (contrainte d'unicité) et une ligne par paire matchée. Un verrou consultatif transactionnel sur la paire garantit qu'un double like simultané crée exactement un match.
- **Découverte en deux temps** : filtres durs en SQL (préférences mutuelles, âge, distance, visibilité, swipes, passes reçus, blocages) sur un pool borné, puis classement en Go via l'interface `Ranker` (récence d'activité, proximité, intérêts communs). Remplacer l'algorithme ne touche ni le SQL ni les handlers.
- **Temps réel par WebSocket** : authentification par premier message (le token n'apparaît jamais dans les URLs / logs), fermeture à l'expiration du token, reconnexion avec backoff côté client. Les actions passent toujours par REST pour partager validation, autorisation et rate limiting.
- **Stockage local derrière une interface** : `storage.Store` permet de brancher S3 / GCS sans modifier les features.
- **Notifications in-app** : stockées et poussées en temps réel ; les notifications push natives ne sont pas activées (voir ci-dessous).
- **Signalement = blocage** : option la plus sûre par défaut. Un signalement isolé ne masque personne (anti-abus) ; deux signalements « mineur » distincts, ou trois signalements quelconques, masquent le profil en attente de revue.
- **Langue** : interface en français ; les messages d'erreur de l'API (anglais, stables) sont traduits côté client.

## Limites connues et évolutions

- **Notifications push natives** (APNs/FCM via Expo) : non activées car elles nécessitent un projet EAS et des identifiants de stores. Le service de notifications est le point d'extension naturel.
- **Plusieurs instances API** : le hub temps réel et le rate limiter sont en mémoire. Pour scaler horizontalement, brancher Postgres `LISTEN/NOTIFY` ou Redis derrière les interfaces `realtime.Publisher` et `ratelimit`.
- **Modération** : les signalements sont stockés (`reports`, statut `open`) et appliquent des règles automatiques ; il n'existe pas encore d'interface d'administration (revue via SQL).
- **Photos** : l'orientation EXIF est normalisée par l'app avant envoi ; un upload direct d'image pivotée via l'API sera affiché sans rotation.
