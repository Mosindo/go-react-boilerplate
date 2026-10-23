# Cléo — application de rencontre 100 % gratuite

Application de rencontre fullstack (API Go + application mobile Expo/React Native), construite sur le boilerplate `go-react-saas`.

**Gratuite pour de vrai** : aucun abonnement, aucun paywall, aucun « boost », aucune limite artificielle de likes, aucune publicité. Il n'existe aucun code de facturation dans l'application.

## Ce que fait l'application

1. Création de compte, connexion, récupération de compte (code par e-mail).
2. Onboarding : prénom, date de naissance (18 ans minimum, vérifié côté serveur), genre, préférences de rencontre, ville (+ position approximative optionnelle), bio, centres d'intérêt, photos.
3. Découverte : cartes de profils avec photos, centres d'intérêt, distance approximative ; swipe ou boutons *like* / *passer* ; profil complet.
4. Match quand l'intérêt est réciproque (création atomique, sans doublon, sans match incohérent). Un profil déjà traité ne réapparaît jamais.
5. Chat en temps réel (WebSocket) entre matchs : heure d'envoi, lu/non lu, liste des conversations, masquage local d'une conversation.
6. Notifications (nouveau match, nouveau message) en temps réel et en liste.
7. Préférences et confidentialité : tranche d'âge, distance, genres recherchés, visibilité dans la découverte, affichage de la distance.
8. Blocage, signalement (suspension automatique après 5 signalements distincts), suppression définitive du compte et de toutes les données associées.

## Stack

| Couche | Technologies |
| --- | --- |
| API | Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256 + refresh tokens rotatifs, bcrypt, gorilla/websocket, x/image. Pas d'ORM. |
| Mobile | Expo 54, React Native 0.81, TypeScript strict, React Navigation, TanStack Query |
| Infra | Docker, Docker Compose, GitHub Actions |

## Architecture

```text
.
├─ apps/mobile/                 # application Expo (src/api, src/features, src/screens, src/realtime, src/ui, src/theme, src/i18n)
├─ services/api/
│  ├─ cmd/api/                  # serveur HTTP (un seul serveur Gin) + tests d'intégration
│  ├─ cmd/seed/                 # données de démonstration (développement uniquement)
│  └─ internal/
│     ├─ platform/              # config, db (migrations), middleware, realtime (hub WebSocket), mailer, httpx, pagination, errors
│     └─ features/              # auth, profiles, photos, discovery, matching, chat, moderation, notifications
│        └─ <feature>/          # handler.go -> service.go -> repository.go (+ model.go, routes.go)
├─ docs/API.md                  # contrat d'API (source de vérité client/serveur)
├─ docs/ARCHITECTURE.md
├─ docker-compose.yml           # postgres + api (port hôte 18080)
└─ scripts/                     # smoke / QA
```

Règles : handlers fins, logique métier dans `service.go`, SQL uniquement dans `repository.go`, préoccupations transverses dans `platform`. Le mobile n'accède au réseau que via `src/api` (et `src/realtime` pour la WebSocket).

## Installation

Prérequis : Go 1.24+, Node 20.19+ (22 recommandé), Docker (ou un PostgreSQL 15+), un téléphone avec Expo Go ou un émulateur.

```bash
git clone <repo> && cd go-react-boilerplate
cp .env.example .env
# éditer .env : JWT_SECRET est obligatoire (openssl rand -hex 32)
```

### Variables d'environnement

Toutes documentées dans [`.env.example`](.env.example).

| Variable | Obligatoire | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées) |
| `DATABASE_URL` | oui (déjà fournie par Compose) | Connexion PostgreSQL |
| `PORT` | non (8080) | Port d'écoute de l'API |
| `ALLOWED_ORIGINS` | non | Origines navigateur autorisées (CORS + WebSocket) |
| `TRUSTED_PROXIES` | non | Reverse proxies de confiance (pour l'IP client et le rate limiting) |
| `AUTH_RATE_PER_MINUTE` / `API_RATE_PER_MINUTE` | non (20 / 600) | Limites par IP (auth) et par utilisateur (API) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | E-mails de récupération. Sans `SMTP_HOST`, le code est écrit dans le log de l'API (**développement uniquement**) |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API vue par le téléphone |
| `ALLOW_DEMO_SEED`, `DEMO_PASSWORD` | seed | Garde-fou et mot de passe des comptes de démo |

Aucun secret ne doit se trouver dans l'application mobile (`EXPO_PUBLIC_*` est public par construction).

## Base de données et migrations

Les migrations sont des fichiers SQL embarqués dans le binaire (`services/api/internal/platform/db/schema/NNN_*.sql`). Elles sont appliquées automatiquement au démarrage de l'API, dans l'ordre, chacune dans une transaction, et enregistrées dans `schema_migrations` (verrou consultatif : plusieurs instances peuvent démarrer ensemble). Pour ajouter une migration : créer le fichier suivant (`006_….sql`) ; ne jamais modifier une migration déjà publiée.

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `photo_blobs`, `swipes`, `matches`, `messages`, `blocks`, `reports`, `notifications`. Toutes les relations sont en `ON DELETE CASCADE` (suppression de compte complète), avec contraintes `CHECK`/`UNIQUE` et les index nécessaires aux chemins chauds (`messages(match_id, created_at)`, `matches(user, last_activity_at)`, `swipes(target_id, action)`, `notifications(user_id, created_at)`, …).

## Lancement local

```bash
# 1. API + base (Docker)
docker compose up --build -d
curl http://localhost:18080/health            # {"status":"ok"}

# 2. Données de démonstration (facultatif, voir plus bas)
docker compose run --rm -e ALLOW_DEMO_SEED=true api ./seed

# 3. Application mobile
cd apps/mobile
npm ci
# Téléphone physique : JAMAIS localhost, utiliser l'IP LAN de la machine
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start
```

Vérifications avant de déclarer l'environnement prêt (conflits de port fréquents) :

```bash
docker compose ps
curl http://<IP_LAN>:18080/health
```

Si un autre service occupe le port 18080, changer le port hôte dans `docker-compose.yml` (et `infra/docker-compose.yml`), `EXPO_PUBLIC_API_URL` et les scripts.

Sans Docker : démarrer PostgreSQL, puis `cd services/api && DATABASE_URL=… JWT_SECRET=… PORT=18080 go run ./cmd/api`.

### Seed

`cmd/seed` crée 10 comptes **clairement fictifs** : adresses `demoNN@demo.invalid` (domaine réservé, non routable), bios marquées « compte de démonstration », photos générées (dégradés, jamais de vraies personnes). Il refuse de s'exécuter sans `ALLOW_DEMO_SEED=true`, est idempotent, et se retire avec :

```sql
DELETE FROM users WHERE email LIKE '%@demo.invalid';
```

Connexion : `demo01@demo.invalid` / `DemoPass123` (ou `DEMO_PASSWORD`). Ne jamais lancer le seed en production.

## Tests et qualité

```bash
# Backend (les tests d'intégration utilisent DATABASE_URL_TEST ou DATABASE_URL et sont ignorés sans base)
cd services/api
gofmt -l .                       # doit être vide
go vet ./...
DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable go test -race ./...
go build ./cmd/api

# Mobile
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test

# Parcours critique de bout en bout contre une API qui tourne
npm run e2e:smoke                # inscription -> profil -> photo -> découverte -> match -> chat -> WebSocket -> blocage -> suppression
```

Les tests d'intégration backend couvrent : authentification (rotation/refus de refresh, logout immédiat, récupération de mot de passe, verrouillage après 5 codes faux, rate limiting), profils et validation (18+), photos (types, taille, EXIF, limites, contrôle d'accès), découverte (préférences mutuelles, distance approximative, classement), likes/matchs (idempotence, concurrence, décisions définitives), chat (confidentialité, pagination, lu/non lu, masquage, unmatch), blocage/signalement/suspension automatique, notifications, WebSocket (auth, isolation entre utilisateurs, origines), suppression de compte.

CI : `.github/workflows/ci.yml` (Postgres de service, gofmt, `go test -race`, build, typecheck/lint/format/tests mobiles, image Docker + `/health`) et `qa-lite.yml` (stack Compose + smoke).

## Build et déploiement

- API : `docker build -t cleo-api services/api` (image Alpine, utilisateur non root, binaires `api` et `seed`). Fournir `DATABASE_URL`, `JWT_SECRET` (≥ 32 caractères aléatoires), `SMTP_*`, `ALLOWED_ORIGINS` si un client web existe, `TRUSTED_PROXIES` derrière un reverse proxy. Terminer TLS devant l'API (HTTPS/WSS obligatoires en production).
- Mobile : builds EAS / `expo prebuild` avec `EXPO_PUBLIC_API_URL=https://api.exemple.tld`.
- Passage à l'échelle : le hub WebSocket et le rate limiter sont **en mémoire** (une instance). Pour plusieurs instances : router les sockets avec `LISTEN/NOTIFY` PostgreSQL ou un broker et partager les compteurs de limites.

## Principales décisions techniques

- **Conservé** : Go/Gin/pgx sans ORM, architecture `handler → service → repository`, sessions JWT (15 min) + refresh tokens rotatifs (SHA-256 en base, rotation atomique, un seul refresh concurrent gagne). **Retiré** : Stripe/abonnements, organisations multi-tenant, posts, commentaires, fichiers génériques, Tamagui (inutilisé).
- **Session vérifiée en base à chaque requête** : logout, changement de mot de passe, suspension et suppression de compte prennent effet immédiatement, pas à l'expiration du JWT. JWT strict : HS256 uniquement, `exp` et `iss` obligatoires.
- **Match = conversation** : une ligne `matches` (paire ordonnée `user_a < user_b`, unique) porte la conversation ; pas de table `conversations` redondante. Un verrou consultatif par paire rend le « like réciproque » exempt de course.
- **Décisions de swipe définitives** : pas de « défaire un like » (un profil traité n'est jamais réaffiché) ; `unmatch` et blocage ferment la conversation sans permettre de re-match.
- **Découverte** : une requête SQL unique filtre (préférences **mutuelles** : genre, âge, distance ; actif, visible, onboardé ; sans swipe ni blocage) et classe (intérêts communs, personnes qui vous ont déjà liké, activité récente, proximité). `Recommender` isole le classement pour le remplacer plus tard sans toucher aux handlers. Distance par Haversine SQL ; pour de très gros volumes, passer à PostGIS ou un index géohash.
- **Vie privée de la localisation** : coordonnées arrondies à ~2 km avant stockage, jamais renvoyées par l'API ; distance affichée par paliers de 5 km (anti-trilatération), masquable par l'utilisateur ; position facultative et effaçable (`clearLocation`).
- **Photos** : validées par le contenu (JPEG/PNG/WebP, ≤ 8 Mo, 64–8000 px, refus des bombes de décompression), réencodées en JPEG (métadonnées EXIF/GPS supprimées, orientation corrigée), deux variantes (1080 px / 320 px), stockées en base (`bytea` dans une table séparée : la suppression de compte est atomique et il n'y a pas de fichier orphelin). Servies uniquement par `GET /photos/:id` authentifié (ni propriétaire ni visible → 404), cache privé + ETag. Au-delà de quelques milliers d'utilisateurs : stockage objet derrière la même interface de service.
- **Temps réel** : WebSocket serveur → client uniquement ; l'envoi passe par REST (validation, limites, idempotence au même endroit). Le jeton est envoyé dans la première trame, jamais dans l'URL ; buffer borné par client (client lent déconnecté puis resynchronisé via REST).
- **Notifications** : en base + temps réel. Les messages non lus d'une conversation sont fusionnés en une seule notification ; les push OS (APNs/FCM) ne sont pas inclus (nécessitent des comptes développeur) — le point d'extension est `notifications.Service`.
- **Abus** : rate limiting par IP (auth) et par utilisateur (API, envoi de messages, uploads) sans bornes artificielles sur les likes ; en-têtes de sécurité ; corps JSON limité à 1 Mo ; `TRUSTED_PROXIES` vide par défaut (IP non falsifiable via `X-Forwarded-For`).
- **Récupération de compte** : code de 10 caractères (≈ 50 bits), haché en HMAC, valable 30 min, 5 essais, usage unique, révoque toutes les sessions ; la réponse ne révèle jamais si l'e-mail existe.
- **Modération** : un signalement par couple (rejouer ne gonfle pas le compteur) ; 5 signalants distincts ouverts → suspension automatique (sessions révoquées, invisible, connexion refusée). La revue est manuelle :

  ```sql
  SELECT r.reported_id, count(*) FROM reports r WHERE r.status = 'open' GROUP BY 1 ORDER BY 2 DESC;
  UPDATE users SET status = 'active' WHERE id = '<id>';          -- lever une suspension
  UPDATE reports SET status = 'dismissed' WHERE reported_id = '<id>';
  ```
  Supprimer un compte supprime aussi les signalements qu'il avait émis ou reçus.
- **Interface** : français par défaut (textes centralisés dans `apps/mobile/src/i18n/fr.ts`), thème clair/sombre automatique, identité visuelle originale (prune/terracotta/ambre).

## Limites connues

- Pas de notifications push hors application, pas de vérification d'e-mail à l'inscription, pas de vérification de photo (selfie), pas d'outil d'administration (requêtes SQL ci-dessus).
- Sur le web (`expo start --web`), les photos privées ne s'affichent pas : React Native Web ignore l'en-tête `Authorization` des `<Image>` ; l'application cible iOS/Android.
- L'interface n'a pas pu être testée sur appareil physique dans l'environnement de développement automatisé : le parcours critique est vérifié par les tests d'intégration backend, `npm run e2e:smoke` (HTTP + WebSocket réels) et les tests unitaires mobiles ; le flux Maestro (`apps/mobile/e2e/maestro`) est fourni pour valider l'UI sur émulateur.

## Legacy code pending removal

Le boilerplate d'origine contenait des modules sans rapport avec l'application. Ils ne sont **plus branchés** (ni dans le routeur, ni dans la
migration embarquée, ni dans le binaire), mais leurs sources sont encore dans le dépôt : `AGENTS.md` exige une validation avant toute
suppression de fichier et l'exécution automatisée n'a pas pu l'obtenir. Pour finir le nettoyage :

```bash
git rm -r services/api/internal/features/{billing,comments,posts,files,users} \
          services/api/internal/platform/logger \
          services/api/internal/platform/db/migrations \
          docs/agents/payments-agent.md .codex/config.toml
```

(`platform/logger` n'est utilisé que par ces modules ; `db/migrations` est l'ancien schéma, remplacé par `db/schema`.) Rien d'autre ne les référence ;
`go build ./... && go test ./...` doit rester vert après suppression.
