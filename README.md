# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre complète (API Go + application mobile Expo/React Native) pensée pour des rencontres sincères, sans aucune mécanique payante :

- **aucun abonnement, aucun paywall, aucun boost payant, aucune publicité** ;
- **likes illimités** (seul un anti-abus technique existe, jamais une limite produit) ;
- **« Ils t'ont liké » est gratuit**, comme toutes les autres fonctionnalités ;
- **position exacte jamais enregistrée** : seule une distance approximative est montrée.

Parcours couverts : inscription, connexion, récupération de compte, création du profil (prénom, âge, genre, bio, centres d'intérêt, photos, préférences, localisation approximative), découverte par swipe ou boutons, like / pass, match réciproque, discussion en temps réel avec accusés de lecture, notifications, préférences, confidentialité (pause du profil, suppression de la position), blocage, signalement, annulation de match, suppression locale d'une conversation et suppression du compte.

---

## Sommaire

1. [Stack technique](#stack-technique)
2. [Architecture](#architecture)
3. [Installation et lancement local](#installation-et-lancement-local)
4. [Variables d'environnement](#variables-denvironnement)
5. [Base de données, migrations et seed](#base-de-données-migrations-et-seed)
6. [Tests, qualité et build](#tests-qualité-et-build)
7. [Déploiement](#déploiement)
8. [Référence de l'API](#référence-de-lapi)
9. [Décisions techniques et produit](#décisions-techniques-et-produit)

---

## Stack technique

| Couche | Choix |
| --- | --- |
| API | Go 1.24, Gin, PostgreSQL 16 via `pgxpool` (SQL explicite, pas d'ORM) |
| Auth | JWT HS256 (15 min, `iss`/`typ`/`exp` vérifiés) + refresh tokens rotatifs stockés hachés, bcrypt |
| Temps réel | WebSocket (`gorilla/websocket`) avec tickets d'accès de 60 s |
| Photos | Validation par signature binaire, ré-encodage JPEG (suppression des métadonnées EXIF/GPS), URLs signées et expirantes |
| Mobile | Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 6, TanStack Query 5 |
| Qualité | `go vet`, `gofmt`, tests Go (unitaires + intégration sur PostgreSQL), ESLint, Prettier, `node:test`, E2E API et E2E UI Playwright |
| Infra | Docker (image non-root), Docker Compose, GitHub Actions |

Dépendances ajoutées et pourquoi :
- `github.com/gorilla/websocket` : WebSocket standard et éprouvé (React Native ne supporte pas les flux `fetch`, donc pas de SSE) ;
- `golang.org/x/image` : redimensionnement de qualité et décodage WebP pour le traitement des photos ;
- `expo-image-picker`, `expo-image` (cache disque), `expo-location`, `expo-linear-gradient`, `@react-navigation/native-stack` ;
- `react-native-web`, `react-dom`, `@expo/metro-runtime` : build web utilisé pour les tests E2E Playwright ;
- `eslint`, `eslint-config-expo`, `prettier`, `@types/node` (outillage).

Dépendances retirées : Tamagui (inutilisé), `@types/react-native` (obsolète), tout le module Stripe/billing.

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS, Android, web)
│  ├─ App.tsx                   Providers (thème, React Query, auth) + navigation
│  ├─ src/api/                  Client REST unique (refresh automatique), endpoints typés, WebSocket
│  ├─ src/features/             auth, onboarding, discovery, likes, chat, notifications, profile, safety
│  ├─ src/lib/                  Fonctions pures (validation, formatage) testées avec node:test
│  ├─ src/navigation/           Pile non connectée → onboarding → onglets + écrans
│  ├─ src/providers/            Auth, cache React Query, synchronisation temps réel
│  ├─ src/theme/ et src/ui/     Design system (clair/sombre) et composants réutilisables
│  ├─ scripts/                  E2E API et E2E web, démarrage sur téléphone
│  └─ tests/                    Tests unitaires TypeScript
├─ services/api/                API Go
│  ├─ cmd/api                   Serveur HTTP (arrêt gracieux)
│  ├─ cmd/seed                  Données de démonstration
│  └─ internal/
│     ├─ app/                   Racine de composition + tests d'intégration
│     ├─ features/              auth, profiles, photos, matching, chat, notifications, safety
│     └─ platform/              config, db (migrations), httpx, middleware, authtoken,
│                               realtime, storage, mediaurl, mailer, logger
├─ docs/ARCHITECTURE.md         Détails d'architecture et modèle de données
├─ docker-compose.yml           PostgreSQL + API
└─ .github/workflows/ci.yml     CI complète
```

Chaque feature Go suit `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`) : handlers fins, règles métier dans les services, SQL uniquement dans les repositories. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation et lancement local

### Prérequis

- Docker et Docker Compose (ou PostgreSQL 14+ installé localement)
- Go 1.24+ (pour lancer l'API hors Docker et les tests)
- Node.js 22+ et npm
- Expo Go ou un simulateur iOS / émulateur Android pour l'application

### 1. API + base de données avec Docker

```bash
cp .env.example .env
# Renseigner JWT_SECRET, par exemple :
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env
docker compose up --build -d
curl http://localhost:18080/health      # {"status":"ok"}
```

L'API écoute sur le port hôte **18080** (8080 dans le conteneur). Les photos sont stockées dans le volume `uploads_data`, la base dans `postgres_data`.

> Vous aviez lancé l'ancien boilerplate ? Son schéma est incompatible : l'API refuse de démarrer sur une base qui contient déjà ces tables. Réinitialisez le volume avec `docker compose down -v`.

### 1 bis. API sans Docker

```bash
cd services/api
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/app?sslmode=disable"
export JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/api            # applique les migrations puis écoute sur :8080 (PORT)
```

### 2. Données de démonstration (facultatif)

```bash
# Avec Docker :
docker compose exec api ./seed
# Sans Docker (mêmes variables que l'API) :
cd services/api && go run ./cmd/seed --likes-for ton.email@exemple.fr
```

Voir [Seed](#seed-données-de-démonstration).

### 3. Application mobile

```bash
cd apps/mobile
npm ci
# Simulateur iOS / navigateur : l'URL par défaut http://localhost:18080 suffit.
# Émulateur Android : http://10.0.2.2:18080 est utilisé automatiquement.
# Téléphone physique : utiliser l'IP LAN de la machine, jamais localhost.
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npx expo start
# ou : npm run start:phone   (détecte l'IP LAN automatiquement)
```

Pour tester dans un navigateur : `npm run web` et ajouter `ALLOWED_ORIGINS=http://localhost:8081` côté API.

Avant de déclarer l'application prête sur un téléphone, vérifier qu'aucun service local ne capture le port (`docker compose ps`, `curl http://<IP_LAN>:18080/health`).

## Variables d'environnement

Toutes sont documentées dans [.env.example](.env.example). Aucun secret n'est commité ; le frontend ne contient aucun secret.

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | oui | — | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées). Sert aussi à dériver la clé des URLs de photos. |
| `DATABASE_URL` | oui | — | Connexion PostgreSQL (un seul pool partagé). |
| `PORT` | non | `8080` | Port HTTP de l'API. |
| `APP_ENV` | non | `development` | `development`, `test` (limites de débit ×20 pour l'E2E) ou `production` (mode release, codes de réinitialisation jamais journalisés, seed refusé). |
| `UPLOAD_DIR` | non | `./data/uploads` | Dossier de stockage des photos (volume en Docker). |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS). Inutile pour les apps natives. |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies dont `X-Forwarded-For` est fiable (limitation de débit par IP). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | port 587 | Envoi des codes de récupération. Sans SMTP, le code est écrit dans les logs de l'API **hors production uniquement**. |
| `SEED_DEMO_PASSWORD` | non | `Demo12345` | Mot de passe des comptes de démonstration. |
| `EXPO_PUBLIC_API_URL` | conseillé | `http://localhost:18080` en dev | URL publique de l'API pour l'application (non secrète). |

## Base de données, migrations et seed

### Migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées **au démarrage**, chacune **une seule fois** dans sa propre transaction, suivies dans `schema_migrations` et protégées par un verrou consultatif (plusieurs instances peuvent démarrer en même temps). Pour faire évoluer le schéma : ajouter un fichier `00N_description.sql`, ne jamais modifier une migration déjà appliquée.

Tables : `users`, `sessions`, `password_reset_tokens`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (likes et pass), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Contraintes (âge ≥ 18 ans, genres, bornes de préférences, paire de match canonique `user_a < user_b`, unicité des swipes, ordre des photos), clés étrangères `ON DELETE CASCADE` et index sur tous les chemins chauds sont détaillés dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

### Seed (données de démonstration)

`cmd/seed` crée 22 profils de démonstration autour de Paris (paramétrable avec `--lat` / `--lng`) :
- comptes marqués `users.is_demo = true` et affichés avec un badge **« Profil démo »** dans l'app ;
- e-mails sur le domaine réservé `demo.lueur.invalid` (`demo01@…` à `demo22@…`) ;
- photos abstraites générées (aucun visage réel) ;
- **refusé quand `APP_ENV=production`** ; relancer le seed remplace les anciens comptes démo.

Option `--likes-for <email>` : la moitié des profils démo likent ce compte, pour obtenir des matchs immédiatement en likant en retour.

## Tests, qualité et build

### API (`services/api`)

```bash
gofmt -l .                      # doit être vide
go vet ./...
# Les tests d'intégration ont besoin d'une base dédiée (elle est vidée à chaque test) :
createdb app_test
DATABASE_URL_TEST="postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable" go test -race ./...
go build ./cmd/api ./cmd/seed
```

Sans `DATABASE_URL_TEST`, les tests d'intégration sont ignorés et seuls les tests unitaires s'exécutent.

Couverture des tests d'intégration (`internal/app`) : santé ; cycle d'authentification (normalisation e-mail, rotation et réutilisation des refresh tokens, révocation immédiate au logout, ticket temps réel refusé comme bearer) ; récupération de compte (code à usage unique, verrouillage après 5 essais) et changement de mot de passe ; validation du profil (mineur refusé, date de naissance verrouillée, coordonnées arrondies en base) ; photos (fichier non-image, taille minimale, limite de 6, réordonnancement, remplacement, URL signée et falsifiée, suppression de la dernière photo refusée, photo d'autrui) ; règles de découverte (préférences mutuelles, âge, distance, profils déjà vus, pause, profil incomplet, pagination invalide) ; like → match, doublons, « ils t'ont liké », conversations, permissions d'un tiers, accusés de lecture, pagination, notifications regroupées, suppression locale, unmatch ; **likes simultanés → un seul match** ; blocage et signalement (seuil de modération) ; suppression de compte (données et fichiers effacés) ; livraison temps réel WebSocket ; limitation de débit.

### Mobile (`apps/mobile`)

```bash
npm run typecheck
npm run lint
npm run format:check
npm test                        # tests unitaires (node:test)
npm run build:web               # export web (vérifie le bundling)
```

### Tests de bout en bout

Avec une API lancée (`APP_ENV=test` conseillé, `ALLOWED_ORIGINS=http://localhost:8081`) :

```bash
cd apps/mobile
npm run e2e:api                                   # parcours critique via l'API
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
python3 -m http.server 8081 --directory dist &     # ou tout serveur statique
npm install --no-save playwright && npx playwright install chromium
E2E_SCREENSHOTS=./e2e-shots npm run e2e:web       # parcours UI complet dans Chromium
```

Le test UI déroule : inscription → refus d'une date de naissance mineure → onboarding complet avec upload de photo → découverte → like → match → message → réponse reçue en temps réel → liste des matchs → suppression du compte.

### CI

`.github/workflows/ci.yml` exécute à chaque push : gofmt, vet, tests Go avec `-race` sur PostgreSQL 16, build ; typecheck, lint, format, tests unitaires et build web du mobile ; E2E API + UI Playwright (captures d'écran en artefacts) ; build de l'image Docker et sonde `/health`.

## Déploiement

1. **Base de données** : PostgreSQL 14+ managé, avec sauvegardes. L'extension `pgcrypto` est créée par la première migration (droits nécessaires).
2. **API** : construire `services/api/Dockerfile` (image Alpine, utilisateur non-root, binaires `api` et `seed`). Variables minimales : `APP_ENV=production`, `DATABASE_URL`, `JWT_SECRET` (`openssl rand -hex 32`), `SMTP_*`, `TRUSTED_PROXIES`. Monter un volume persistant sur `/data/uploads`.
3. **TLS** : placer l'API derrière un reverse proxy HTTPS (le WebSocket passe par `wss://` automatiquement). HSTS est envoyé lorsque la requête arrive en HTTPS.
4. **Mise à l'échelle** : le hub temps réel, la limitation de débit et le stockage local sont **par instance**. Pour plusieurs instances : stockage objet (implémenter `storage.Store`, par ex. S3), diffusion des événements via PostgreSQL `LISTEN/NOTIFY` ou Redis (interface `realtime.Publisher`) et limitation de débit partagée.
5. **Application** : `eas build` (Expo Application Services) avec `EXPO_PUBLIC_API_URL=https://api.votre-domaine`. Les identifiants `app.lueur.mobile` et les textes de permission sont dans `app.json`.
6. **Modération** : les signalements sont dans la table `reports` (`status = 'open'`). Un compte signalé par 3 personnes différentes en 30 jours passe en `moderation_status = 'under_review'` (masqué de la découverte). Pour suspendre : `UPDATE users SET moderation_status = 'suspended' WHERE id = …` (toutes ses sessions deviennent invalides immédiatement).

## Référence de l'API

Toutes les routes (sauf indication) exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "message", "code": "code_machine"}`. Listes paginées par `limit`/`offset` validés (400 si invalides), messages par curseur `before`.

| Domaine | Routes |
| --- | --- |
| Santé | `GET /health` (public) |
| Auth (publiques, limitées par IP) | `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `POST /auth/password/forgot`, `POST /auth/password/reset` |
| Compte | `GET /me`, `POST /me/password`, `DELETE /me` (mot de passe requis), `POST /realtime/ticket`, `GET /realtime?ticket=` (WebSocket) |
| Profil | `GET /interests`, `GET /profile`, `PUT /profile`, `PUT /profile/preferences`, `PUT /profile/location`, `DELETE /profile/location`, `PUT /profile/visibility`, `GET /profiles/:userId` |
| Photos | `GET /profile/photos`, `POST /profile/photos` (multipart `photo`), `PUT /profile/photos/:id` (remplacer), `DELETE /profile/photos/:id`, `PUT /profile/photos/order`, `GET /media/photos/:id?exp=&sig=` (URL signée, publique) |
| Matching | `GET /discovery`, `POST /swipes` (`like`/`pass`), `GET /likes/received`, `DELETE /matches/:matchId` |
| Chat | `GET /conversations?filter=all|new|active`, `GET /conversations/:id`, `GET /conversations/:id/messages?before=`, `POST /conversations/:id/messages`, `POST /conversations/:id/read`, `DELETE /conversations/:id` (effacer pour moi) |
| Notifications | `GET /notifications`, `POST /notifications/:id/read`, `POST /notifications/read-all` |
| Sécurité | `GET /blocks`, `POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` |

Événements WebSocket : `message.created`, `conversation.read`, `match.created`, `match.removed`, `notification.created`.

## Décisions techniques et produit

- **Construire sur l'existant** : le boilerplate Go/Gin/pgx + Expo était sain ; il a été conservé et spécialisé. Les modules génériques hors produit (billing Stripe, organisations multi-tenant, posts, commentaires, fichiers génériques) ont été supprimés : une application gratuite n'a pas besoin de facturation, et le multi-tenant compliquait chaque requête sans bénéfice.
- **Âge** : 18 ans minimum, vérifié côté client, côté service et par une contrainte SQL. La date de naissance n'est plus modifiable après création (empêche de contourner la limite) ; seul l'âge est exposé.
- **Localisation** : demandée uniquement sur action explicite, en précision faible ; arrondie sur l'appareil puis **alignée sur une grille d'environ 1 km côté serveur** avant stockage. Les autres membres ne voient qu'une distance arrondie (≥ 1 km, par paliers de 5 km au-delà de 10 km). Sans position, pas de filtre de distance. Les préférences d'âge, de genre et de distance sont **mutuelles**.
- **Découverte** : toutes les règles d'éligibilité sont en SQL (préférences mutuelles, bounding box + haversine, exclusion des profils déjà vus et des blocages). Le classement est une interface `Ranker` (heuristique actuelle : activité récente, proximité, intérêts communs, réciprocité) remplaçable par un modèle plus avancé sans toucher au reste.
- **Matching** : une décision par paire (clé primaire), paire de match canonique et unique, verrou consultatif par paire : deux likes simultanés créent exactement un match (testé).
- **Chat** : une conversation n'existe que pour un match vivant (annuler le match, bloquer ou supprimer un compte efface la conversation). Chaque requête est filtrée par la ligne de participation de l'appelant : impossible de lire une conversation tierce. « Effacer la conversation » masque l'historique pour soi uniquement.
- **Temps réel** : WebSocket authentifié par un ticket JWT dédié de 60 s (`typ=realtime`, refusé comme bearer) pour ne jamais exposer de jeton longue durée dans une URL ; repli automatique sur du polling si la connexion tombe.
- **Sessions** : chaque requête vérifie côté serveur que la session n'est pas révoquée : déconnexion, réinitialisation de mot de passe, suspension ou suppression de compte prennent effet immédiatement.
- **Photos** : type détecté par signature binaire, protection contre les bombes de décompression, dimensions minimales, ré-encodage systématique (supprime EXIF/GPS et neutralise les fichiers polyglottes), clés de stockage aléatoires, URLs signées HMAC expirantes (stables par fenêtre de 6 h pour le cache). Une photo remplacée change d'identifiant.
- **Notifications** : in-app et temps réel ; les messages non lus d'une même conversation sont regroupés. Les push notifications natives (Expo) ne sont pas branchées : elles nécessitent un build de développement et un projet EAS (évolution prévue via `notifications.Service`).
- **Récupération de compte** : code de 10 caractères (≈ 50 bits), haché, valable 30 min, usage unique, 5 essais maximum, réponse identique que l'e-mail existe ou non, toutes les sessions révoquées après réinitialisation.
- **Signalement** : bloque automatiquement la personne signalée ; au-delà de 3 signalants distincts en 30 jours, le compte est masqué en attendant une revue humaine.
- **Suppression de compte** : confirmation par mot de passe, suppression en cascade de toutes les données et des fichiers photo, fermeture des connexions temps réel.
- **Limitation de débit** : en mémoire par instance (identification, swipes, messages, uploads, signalements, global par IP). Elle protège contre les robots sans jamais limiter les likes d'un usage humain.
- **Design** : identité originale « Lueur » (accent braise, ivoire chaud, nuit aubergine), titres en serif système, mode clair/sombre automatique, composants accessibles (rôles, libellés, alternatives aux gestes de swipe).
