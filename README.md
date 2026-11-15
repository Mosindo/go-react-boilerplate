# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre complète (mobile iOS/Android via Expo, et web) adossée à une API Go.
Deux personnes peuvent créer un compte, construire leur profil avec photos, découvrir des profils compatibles,
liker ou passer, obtenir un match quand l'intérêt est réciproque, discuter en temps réel, recevoir des
notifications, gérer leurs préférences et leur confidentialité, bloquer ou signaler, et supprimer leur compte.

**Entièrement gratuite** : aucun abonnement, aucun paywall, aucun « boost », likes illimités, la liste
« qui m'a liké » est ouverte à tous, aucune publicité.

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Démarrage rapide (Docker)](#démarrage-rapide-docker)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données et migrations](#base-de-données-et-migrations)
7. [Données de démonstration (seed)](#données-de-démonstration-seed)
8. [Lancement local sans Docker](#lancement-local-sans-docker)
9. [Application mobile](#application-mobile)
10. [Tests et qualité](#tests-et-qualité)
11. [Build et déploiement](#build-et-déploiement)
12. [API](#api)
13. [Décisions techniques et produit](#décisions-techniques-et-produit)

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription (confirmation 18+), connexion, sessions à jetons rotatifs, récupération par code e-mail à 6 chiffres, changement de mot de passe, déconnexion, suppression définitive du compte |
| Onboarding | Prénom → date de naissance → genre → qui rencontrer → photos → bio et centres d'intérêt → localisation approximative |
| Profil | Prénom, âge (calculé), genre, bio, métier, taille, ville, 32 centres d'intérêt (10 max), 1 à 6 photos ordonnables |
| Photos | Upload, recadrage 3:4 et compression sur l'appareil, ordre, photo principale, remplacement, suppression ; côté serveur : détection du type réel, limites de taille/dimensions, ré-encodage JPEG (supprime EXIF/GPS), URLs signées et expirantes |
| Découverte | Pile de cartes à swiper (ou boutons), navigation entre photos, intérêts communs mis en avant, profil complet, filtres réciproques âge/genre/distance, classement pluggable |
| Matching | Like / passer, match à la réciprocité, aucun doublon (contraintes + verrou consultatif), un profil traité ne réapparaît jamais, liste « likes reçus » gratuite, annulation de match |
| Chat | Conversations après match uniquement, temps réel (WebSocket), heure d'envoi, lu/non lu, compteur de non-lus, pagination, envoi optimiste avec relance, effacement local |
| Notifications | Match, message (regroupées par conversation), nouveaux likes ; en direct dans l'app, badges d'onglets, tout marquer comme lu |
| Confidentialité | Mise en pause du profil, masquage de la distance, position arrondie (~1 km) jamais exposée, e-mail jamais visible |
| Sécurité / modération | Blocage (supprime le match et la conversation), signalement motivé (bloque aussi), rate limiting, contrôle d'accès systématique côté serveur |
| UI/UX | Identité visuelle originale, thèmes sombre et clair, animations légères, états de chargement / vide / erreur, toasts, accessibilité (rôles, libellés) |

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, SQL explicite (pas d'ORM), JWT HS256 + refresh tokens
  hashés et rotatifs, bcrypt, WebSocket (`gorilla/websocket`), traitement d'images (`golang.org/x/image`).
- **Mobile / web** : Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 6,
  TanStack Query 5, `expo-image`, `expo-image-picker`, `expo-image-manipulator`, `expo-location`,
  `expo-secure-store`. Le même code tourne sur iOS, Android et le web (`react-native-web`).
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Architecture

```text
.
├─ services/api/                     API Go
│  ├─ cmd/api/                       point d'entrée HTTP (arrêt propre)
│  ├─ cmd/seed/                      données de démonstration (dev uniquement)
│  └─ internal/
│     ├─ app/                        composition (routes, middlewares, dépendances) + tests d'intégration E2E
│     ├─ platform/                   transverse : config, db (migrations), middleware (auth, CORS, logs),
│     │                              apperr, httpx, ratelimit, realtime (hub WebSocket), storage, mailer, textx
│     └─ features/                   domaine : chaque module = handler → service → repository (+ model, routes)
│        ├─ auth/                    comptes, sessions, mot de passe, suppression
│        ├─ profiles/                profil, préférences, confidentialité, intérêts, cartes publiques
│        ├─ photos/                  upload, traitement d'image, ordre, médias signés
│        ├─ matching/                découverte, classement, swipes, matchs, likes reçus
│        ├─ chat/                    conversations, messages, lu/non lu
│        ├─ safety/                  blocages, signalements
│        └─ notifications/           notifications serveur, badges
├─ apps/mobile/                      application Expo
│  ├─ App.tsx                        providers (thème, cache, session, temps réel)
│  ├─ scripts/                       e2e-smoke.js (API), e2e-web.js (navigateur), start-phone.js
│  └─ src/
│     ├─ api/                        client REST typé (refresh automatique), client WebSocket, DTO
│     ├─ session/                    stockage sécurisé des jetons, état de session
│     ├─ realtime/                   synchronisation des événements temps réel avec le cache
│     ├─ navigation/                 pile d'auth, onboarding, onglets + écrans modaux
│     ├─ features/                   auth, onboarding, discovery, likes, chat, notifications, profile, safety
│     ├─ ui/ · theme/                design system (composants, palette sombre/claire, typographie)
│     ├─ lib/                        formatage, validation, messages d'erreur
│     └─ __tests__/                  tests Jest
├─ docker-compose.yml                Postgres + API (+ volume des photos)
└─ docs/ARCHITECTURE.md              détails (modèle de données, flux, sécurité)
```

Règles : handlers fins, logique métier dans les services, SQL uniquement dans les repositories, aucune
requête N+1 (les cartes de profil d'une page sont chargées en 3 requêtes quel que soit leur nombre),
l'interface n'accède jamais au réseau directement (toujours via `src/api`).

## Démarrage rapide (Docker)

Prérequis : Docker, Node.js 20+, npm.

```bash
cp .env.example .env
# Renseignez JWT_SECRET (32+ caractères aléatoires) :
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env

docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# (optionnel) profils de démonstration
docker compose exec api ./seed
```

Puis l'application :

```bash
cd apps/mobile
npm ci
npm run web          # navigateur (ajoutez ALLOWED_ORIGINS=http://localhost:8081 dans .env puis relancez l'API)
npm start            # Expo Go / émulateurs
npm run start:phone  # téléphone physique : détecte l'IP LAN et la passe à EXPO_PUBLIC_API_URL
```

> Téléphone physique : n'utilisez jamais `localhost`. L'API doit être joignable via
> `http://<IP_LAN>:18080` (vérifiez avec `curl http://<IP_LAN>:18080/health`). Si le port 18080 est déjà
> pris, changez le mapping dans `docker-compose.yml` et la valeur de `EXPO_PUBLIC_API_URL`.

> Une ancienne base issue du boilerplate (schéma différent) doit être réinitialisée :
> `docker compose down -v`.

## Variables d'environnement

Toutes sont documentées dans [`.env.example`](.env.example). Aucun secret n'est committé ni embarqué
dans l'application.

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | oui | — | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées). Dérive aussi la clé de signature des URLs de photos. |
| `DATABASE_URL` | oui (hors Compose) | — | Chaîne de connexion PostgreSQL. |
| `APP_ENV` | non | `development` | `development`, `test` ou `production` (mode release Gin, pas de mail en clair dans les logs, seed refusé). |
| `PORT` | non | `8080` | Port HTTP de l'API. |
| `UPLOAD_DIR` | non | `./data/uploads` | Répertoire des photos (volume `/data/uploads` en Docker). |
| `ALLOWED_ORIGINS` | non | vide | Origines CORS autorisées (uniquement utile pour la version web). |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies dont `X-Forwarded-For` est fiable (rate limiting par IP). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | — | Envoi des codes de récupération. Sans SMTP : codes affichés dans les logs en dev, non envoyés en production. |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | non | `postgres`/`postgres`/`app` | Identifiants de la base Compose (dev). |
| `SEED_PASSWORD` | non | `DemoPass123` | Mot de passe des comptes de démonstration. |
| `EXPO_PUBLIC_API_URL` | conseillé | `localhost:18080` (émulateurs) | URL de l'API pour l'application (publique, jamais un secret). |

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et
appliquées au démarrage de l'API : chacune une seule fois, dans une transaction, sous verrou consultatif
PostgreSQL (plusieurs instances peuvent démarrer simultanément), et tracée dans `schema_migrations`.

Tables : `users`, `sessions`, `password_reset_codes`, `profiles`, `preferences`, `interests`,
`user_interests`, `photos`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`,
`blocks`, `reports`, `notifications` — avec clés étrangères, `CHECK`, contraintes d'unicité, timestamps et
index sur tous les chemins chauds. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

Ajouter une migration : créer `002_description.sql` (ordre lexical) ; ne jamais modifier une migration
déjà appliquée.

## Données de démonstration (seed)

```bash
# Docker
docker compose exec api ./seed
docker compose exec api ./seed -like-email vous@exemple.com   # ces profils vous ont déjà liké
docker compose exec api ./seed -reset                         # supprime tous les profils démo

# Hors Docker
cd services/api && go run ./cmd/seed [-lat 45.76 -lng 4.84]
```

Le seed crée 18 profils fictifs autour de Paris (ou des coordonnées fournies) avec des images abstraites
générées (aucun visage réel). Ils sont **clairement séparés** des vrais comptes : drapeau `users.is_demo`,
e-mails sur le domaine réservé `.invalid`, badge « Profil démo » dans l'application, récupération de mot de
passe désactivée, et la commande refuse de s'exécuter si `APP_ENV=production`.

## Lancement local sans Docker

```bash
# PostgreSQL 16 accessible localement, puis :
cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export PORT=18080 ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api
```

Les codes de récupération de mot de passe apparaissent dans les logs (`"event":"dev_mail"`) tant que SMTP
n'est pas configuré.

## Application mobile

| Commande (dans `apps/mobile`) | Rôle |
| --- | --- |
| `npm start` / `npm run android` / `npm run ios` | Expo (Expo Go, émulateurs) |
| `npm run start:phone` | Expo avec `EXPO_PUBLIC_API_URL` réglé sur l'IP LAN |
| `npm run web` | Version web en développement |
| `npm run build:web` | Export web statique dans `dist/` |

Parcours : Bienvenue → Inscription → Onboarding → onglets **Découvrir · Likes · Messages · Profil**
(+ Notifications, Préférences, Réglages, Personnes bloquées, Mot de passe, Suppression du compte).

## Tests et qualité

| Commande | Contenu |
| --- | --- |
| `cd services/api && gofmt -l . && go vet ./... && go test ./...` | Tests unitaires (validation, âge, distance, classement, images, JWT, rate limit, URLs signées, curseurs…) et, si `DATABASE_URL_TEST` (ou `DATABASE_URL`) est défini, **tests d'intégration HTTP de bout en bout** : inscription → profil → photo → découverte → like → match → chat temps réel → permissions → blocage/signalement → reset → suppression, plus une course de likes simultanés (un seul match). |
| `cd services/api && go build ./cmd/api ./cmd/seed` | Build |
| `cd apps/mobile && npm run typecheck` | TypeScript strict |
| `npm run lint` · `npm run format:check` | ESLint (config Expo, `any` interdit) · Prettier |
| `npm test` | Jest : validation, formatage, client API (refresh unique, déconnexion), composants UI |
| `npm run e2e:api` | Smoke test API contre un serveur lancé (`E2E_API_URL`) |
| `npm run e2e:web` | Parcours complet dans Chromium (Playwright) sur l'export web : inscription, onboarding avec upload photo et géolocalisation, likes reçus, match, chat temps réel avec accusé de lecture, notifications, réglages, suppression du compte (`E2E_WEB_URL`, `E2E_API_URL`, `E2E_SCREENSHOTS`) |

La CI GitHub Actions (`.github/workflows/ci.yml`) exécute tout cela : API (Postgres en service), mobile
(types, lint, format, Jest, build web), E2E (API + navigateur, captures d'écran en artefacts) et Docker
Compose (`/health`).

## Build et déploiement

- **API** : image Docker multi-étapes (`services/api/Dockerfile`), binaire statique, utilisateur non-root,
  volume `/data/uploads`. Contient aussi `./seed`. En production : `APP_ENV=production`, `JWT_SECRET`
  fort, PostgreSQL managé, HTTPS terminé par un reverse proxy (renseigner `TRUSTED_PROXIES`), SMTP
  configuré, volume persistant (ou stockage objet via l'interface `storage.Store`).
- **Mobile** : `eas build` (Expo Application Services) avec `EXPO_PUBLIC_API_URL` pointant vers l'API
  HTTPS. Les permissions demandées sont minimales (photos, localisation approximative « pendant
  l'utilisation » ; localisation précise et arrière-plan bloquées sur Android).
- **Web** : `npm run build:web` puis servir `dist/` en statique ; ajouter l'origine dans `ALLOWED_ORIGINS`.

Limites assumées de cette version (une seule instance) : le hub temps réel et le rate limiting sont en
mémoire. Pour plusieurs instances, remplacer `realtime.Publisher` par un broker (Postgres LISTEN/NOTIFY
ou Redis) et le limiteur par un stockage partagé ; les interfaces sont prévues pour.

## API

Réponses JSON ; erreurs au format `{"error": "...", "code": "..."}` ; listes paginées par curseur
(`nextCursor`). Toutes les routes sauf auth, `/health`, `/interests` et `/media` exigent
`Authorization: Bearer <accessToken>`.

| Méthode & route | Rôle |
| --- | --- |
| `POST /auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` | Comptes et sessions |
| `POST /auth/password/forgot` · `/auth/password/reset` | Récupération par code |
| `GET /me` · `PUT /me/password` · `DELETE /me` | Compte courant, mot de passe, suppression |
| `GET/PUT /me/profile` · `PUT/DELETE /me/location` · `PUT /me/preferences` · `PUT /me/privacy` | Profil |
| `GET /interests` | Catalogue des centres d'intérêt |
| `GET /me/photos` · `POST /me/photos` · `PUT /me/photos/order` · `PUT/DELETE /me/photos/:id` | Photos |
| `GET /media/:key?exp&sig` | Fichier photo (URL signée) |
| `GET /profiles/:userId` | Profil public (si visible) |
| `GET /discovery` · `POST /swipes` · `GET /likes/received` · `DELETE /matches/:id` | Matching |
| `GET /conversations` · `GET/DELETE /conversations/:id` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` | Chat |
| `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports` | Sécurité |
| `GET /notifications` · `POST /notifications/:id/read` · `POST /notifications/read-all` · `GET /me/badges` | Notifications |
| `GET /ws` | WebSocket : premier message `{"type":"auth","token":"..."}` puis événements `message.new`, `conversation.read`, `match.new`, `match.removed`, `notification.new` |

## Décisions techniques et produit

- **Construire sur le boilerplate** : l'architecture Go (handler → service → repository, SQL explicite,
  pgxpool) et le client Expo étaient sains ; ils sont conservés. Les modules sans rapport (posts,
  commentaires, facturation Stripe — incompatible avec la gratuité —, organisations) ont été retirés.
- **Matching** : un like réciproque crée un match, protégé par un verrou consultatif par paire et des
  contraintes d'unicité ; les swipes sont conservés pour ne jamais remontrer un profil traité (même après
  annulation du match). Les critères sont **réciproques** (genre, âge, distance des deux côtés). Le
  classement (`matching.Ranker`) combine activité, intérêts communs, réciprocité, proximité et qualité du
  profil ; il peut être remplacé sans toucher au SQL.
- **Âge** : 18 ans minimum vérifié côté serveur ; la date de naissance n'est plus modifiable une fois
  définie ; seul l'âge est exposé.
- **Localisation** : demandée une seule fois, au premier plan, en précision « équilibrée » ; arrondie à
  ~1 km avant stockage ; seules des distances entières (minimum « moins de 2 km ») sont montrées, et
  chacun peut masquer la sienne. La ville peut être saisie à la main sans géolocalisation.
- **Photos** : jamais servies publiquement ; URLs signées HMAC expirant après quelques heures (alignées
  pour rester cacheables) ; ré-encodage systématique qui neutralise métadonnées et fichiers piégés.
- **Sessions** : jeton d'accès 15 min dont la session est vérifiée à chaque requête (déconnexion, reset et
  suppression effectifs immédiatement) ; refresh token rotatif, haché en base ; un refresh token déjà utilisé est refusé.
- **Blocage / signalement** : le blocage supprime le match et la conversation sans prévenir l'autre
  personne ; un signalement bloque aussi. Les signalements survivent à la suppression des comptes
  (anonymisés) pour la modération ; il n'y a pas encore d'interface de modération (requête SQL sur
  `reports` en attendant).
- **Suppression de compte** : effacement immédiat et en cascade de toutes les données personnelles et des
  fichiers photos.
- **Temps réel** : WebSocket authentifié par le premier message (le jeton n'apparaît jamais dans une URL ni
  dans les logs), reconnexion exponentielle, et repli automatique sur du polling.
- **Notifications** : en direct dans l'application. Les notifications push système (APNs/FCM) ne sont pas
  activées : elles exigent des comptes développeur et des clés propres au déploiement.
- **Pas de nouvelle dépendance de gestes/animations** : le swipe utilise `PanResponder` + `Animated`.
