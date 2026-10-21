# Alma — application de rencontre 100 % gratuite

Alma est une application de rencontre complète (mobile iOS / Android / web) : compte, profil, photos, découverte, likes, matchs, messagerie en temps réel, notifications, blocage, signalement et suppression du compte.

**Gratuite pour de vrai** : aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de likes, aucune fonctionnalité verrouillée, aucune publicité. (Le seul plafond est un garde-fou anti-script, très au-dessus de la vitesse humaine : 300 swipes / minute.)

## Stack

| Couche | Technologies |
| --- | --- |
| Mobile / web | Expo (React Native 0.81) + TypeScript strict, React Navigation, TanStack Query, `expo-image`, `expo-image-picker`, `expo-location` |
| API | Go 1.24, Gin, PostgreSQL 16, `pgxpool` (pas d'ORM), JWT HS256 + refresh tokens rotatifs, bcrypt, WebSocket (`gorilla/websocket`) |
| Infra | Docker, Docker Compose |
| Qualité | `go test -race` sur une vraie base PostgreSQL, Vitest, ESLint, Prettier, Playwright (parcours critique de bout en bout) |

## Fonctionnalités

- **Compte** : inscription (âge ≥ 18 ans vérifié côté serveur), connexion, récupération par code envoyé par e-mail, changement de mot de passe, suppression du compte (efface toutes les données et les fichiers).
- **Profil** : prénom, âge (calculé depuis la date de naissance, jamais modifiable), genre, bio, ville, métier, taille, ce que l'on cherche, jusqu'à 10 centres d'intérêt, jusqu'à 6 photos (ordre, photo principale, remplacement, suppression).
- **Découverte** : une carte à la fois, swipe **ou** boutons J'aime / Passer (accessible sans geste), fiche complète avec galerie. Filtrage mutuel (orientation, tranche d'âge, distance), jamais deux fois le même profil, bouton « revoir les profils passés ».
- **Matchs** : like réciproque ⇒ match, sans doublon, même si les deux likes sont simultanés (verrou par paire + contraintes SQL).
- **Chat** : conversation créée au match, temps réel (WebSocket), heure, accusés de lecture, non-lus, suppression locale d'une conversation, annulation du match.
- **Notifications** : nouveaux matchs et messages (liste in-app + temps réel + badges). Pas de push natif (voir « Décisions »).
- **Sécurité & modération** : blocage, signalement (motifs), pause du profil, distance masquable, position approximative supprimable.

## Architecture

```
apps/mobile/                  Expo app (iOS / Android / web)
  src/api/                    client HTTP (refresh de jeton, erreurs typées) + un module par domaine
  src/hooks/                  auth, temps réel (WebSocket → cache TanStack Query), swipe optimiste
  src/screens/                auth · onboarding · discover · messages · profile · safety
  src/components/             SwipeCard, PhotoGrid, PhotoCarousel, formulaires
  src/shared/                 design system (ui), layout, feedback (toasts, vues vides/erreur)
  src/lib/                    logique pure testée (validation, formats, brouillon de profil)
  e2e/                        parcours Playwright : inscription → match → chat → suppression
services/api/
  cmd/api                     point d'entrée du serveur
  cmd/seed                    données de démonstration (développement uniquement)
  internal/app                assemblage du routeur Gin
  internal/platform/          config, db (migrations), middleware, httpx, errors, storage, realtime, mail, geo, logger
  internal/features/          auth · profiles · photos · discovery · matches · chat · notifications · safety
                              (handler → service → repository ; routes.go ; model.go)
  internal/platform/db/sql/   migrations SQL embarquées et suivies (table schema_migrations)
```

Règles : handlers fins, logique métier dans les services, SQL uniquement dans les repositories, un seul serveur HTTP (Gin), un seul pool de connexions, l'UI n'appelle jamais le réseau en direct (tout passe par `src/api`).

### Modèle de données

`users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests` / `user_interests`, `photos`, `swipes`, `matches`, `conversations` / `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`.
Clés étrangères en `ON DELETE CASCADE` depuis `users` (droit à l'effacement), contraintes `CHECK` (genres, âges, longueurs, `user_a < user_b` pour l'unicité d'un match), index sur tous les chemins chauds (`swipes(target_id, swiper_id)`, `messages(conversation_id, created_at)`, `notifications(user_id, created_at)`, …).

### Algorithme de découverte

Une seule requête SQL (`internal/features/discovery/repository.go`) : éligibilité **mutuelle** (genre recherché, tranche d'âge, distance, profil non en pause, au moins une photo), exclusion des profils déjà traités et des blocages dans les deux sens ; tri : *ceux qui vous ont déjà liké* d'abord, puis les plus proches (par tranches de 10 km), puis les plus récemment actifs. Remplacer le `ORDER BY` suffit pour faire évoluer la recommandation.

### API (extrait)

| Domaine | Endpoints |
| --- | --- |
| Auth | `POST /auth/register · login · refresh · logout · forgot-password · reset-password`, `GET /me`, `POST /me/password`, `DELETE /me` |
| Profil | `GET/PUT /profile`, `GET/PUT /preferences`, `GET /interests`, `GET /users/:id` |
| Photos | `GET/POST /photos`, `PUT /photos/order`, `PUT/DELETE /photos/:id`, `GET /media/photos/:id` (URL signée) |
| Découverte | `GET /discover`, `POST /swipes`, `DELETE /swipes/passes`, `GET /matches`, `DELETE /matches/:id` |
| Chat | `GET /conversations`, `GET/POST /conversations/:id/messages`, `POST /conversations/:id/read`, `DELETE /conversations/:id` |
| Notifications | `GET /notifications`, `POST /notifications/read` |
| Sécurité | `GET /blocks`, `PUT/DELETE /blocks/:userId`, `POST /reports` |
| Temps réel | `GET /ws` (premier message `{"type":"auth","token":"…"}`) → `message.new`, `match.new`, `conversation.read`, `conversation.removed`, `notification.new` |

Erreurs : `{"error": "message lisible", "code": "code_stable"}`. Listes paginées (`limit`, `before`).

## Installation

Prérequis : Go 1.24+, Node 22+, Docker (ou PostgreSQL 16 local).

### 1. Configuration

```bash
cp .env.example .env            # puis éditez JWT_SECRET et POSTGRES_PASSWORD
cp apps/mobile/.env.example apps/mobile/.env
```

| Variable | Rôle |
| --- | --- |
| `JWT_SECRET` (**requis**) | signe les jetons d'accès **et** les URLs de photos ; ≥ 32 caractères aléatoires (`openssl rand -base64 48`) |
| `DATABASE_URL` (**requis** hors Docker) | connexion PostgreSQL |
| `POSTGRES_USER/PASSWORD/DB` | identifiants de la base Docker Compose |
| `PORT` | port de l'API (défaut `8080`) ; Docker l'expose sur `API_HOST_PORT` (défaut `18080`) |
| `APP_ENV` | `development` (défaut) ou `production` (exige `SMTP_HOST`, interdit le seed) |
| `UPLOAD_DIR` | dossier des photos (volume Docker `uploads_data`) |
| `ALLOWED_ORIGINS` | origines CORS autorisées (build web) |
| `TRUSTED_PROXIES` | proxys dont `X-Forwarded-For` est cru (vide = aucun) |
| `SMTP_HOST/PORT/USER/PASSWORD/FROM` | e-mails de récupération ; sans `SMTP_HOST` (dev) le code est écrit dans le log de l'API |
| `EXPO_PUBLIC_API_URL` (mobile) | adresse de l'API vue par l'app. **Public** : aucun secret |

### 2. Base de données et API

```bash
docker compose up -d --build      # PostgreSQL + API (migrations appliquées au démarrage)
curl http://localhost:18080/health
```

Les migrations (`services/api/internal/platform/db/sql/*.sql`) sont embarquées dans le binaire, exécutées dans l'ordre, chacune dans une transaction, et enregistrées dans `schema_migrations` (verrou consultatif : plusieurs répliques peuvent démarrer ensemble).

### 3. Données de démonstration (développement uniquement)

```bash
docker compose exec api ./seed -confirm -reset
# ou : cd services/api && go run ./cmd/seed -confirm -reset
```

Crée 10 membres `demoNN@demo.alma.invalid` (mot de passe `DemoPass123!`) marqués `users.is_demo = TRUE`, avec des photos dégradées générées (aucune vraie personne). `-reset` ne supprime que les comptes de démonstration. Le seed **refuse de tourner** si `APP_ENV=production`. Connectez-vous avec `demo01@demo.alma.invalid` : quatre membres l'ont déjà liké, liker l'un d'eux déclenche un match.

### 4. Application

```bash
cd apps/mobile
npm ci
npm run web                       # navigateur : http://localhost:8081
npm start                         # Expo Go / simulateurs
npm run start:phone               # téléphone physique : détecte l'IP du réseau local
```

Sur un **téléphone physique**, n'utilisez jamais `localhost` : `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`. Avant de déclarer l'app prête : `docker compose ps`, `curl http://<IP_LAN>:18080/health`, puis un parcours complet (inscription → profil → découverte → match → chat).

## Tests et qualité

```bash
# API — nécessite une base PostgreSQL de test (elle est migrée automatiquement)
cd services/api
export TEST_DATABASE_URL="postgres://alma:alma@localhost:5432/alma_test?sslmode=disable"
gofmt -l . ; go vet ./... ; go test -race -count=1 ./... ; go build ./cmd/api

# Mobile
cd apps/mobile
npm run typecheck && npm run lint && npm run format:check && npm test

# Parcours de bout en bout dans un vrai navigateur (API lancée sur :18080)
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
npm run e2e:web                   # PLAYWRIGHT_CHROMIUM_PATH=/chemin/chromium si besoin
```

Sans `TEST_DATABASE_URL`, les tests d'intégration sont **sautés explicitement** (message `skipping database integration test`) ; la CI les exécute toujours. Ils couvrent : authentification et rotation des jetons, récupération de compte, validation de profil, upload/ordre/remplacement/suppression de photos et URLs signées, découverte (orientation, distance, doublons, pause), likes concurrents, matchs, conversations et **permissions** (un tiers ne peut ni lire, ni écrire, ni marquer lu), blocage, signalement, révocation de session, suppression complète du compte, WebSocket.
Le test Playwright joue : inscription → onboarding → photo → découverte → like → match → message → réponse reçue **en temps réel** → notifications → blocage → suppression du compte.

## Déploiement

1. Construisez l'image : `docker build -t alma-api services/api` (utilisateur non-root, volume `/data/uploads`).
2. Exécutez-la derrière un reverse proxy **TLS** avec `APP_ENV=production`, un `JWT_SECRET` fort, `DATABASE_URL`, SMTP, `TRUSTED_PROXIES` (IP du proxy) et un volume persistant pour `UPLOAD_DIR`.
3. Mobile : `EXPO_PUBLIC_API_URL=https://api.votre-domaine` puis `eas build` (stores) ou `npm run build:web` (dossier `dist/` statique ; ajoutez son origine à `ALLOWED_ORIGINS`).
4. Sauvegardez PostgreSQL et le volume des photos ensemble.

## Sécurité — ce qui est en place

- JWT HS256 strict (algorithme verrouillé, expiration obligatoire, 15 min) + refresh tokens aléatoires **hachés en base et rotatifs** ; la session est revérifiée à chaque requête (cache 10 s) : déconnexion, changement de mot de passe et suppression de compte coupent immédiatement les jetons.
- bcrypt, comparaison à temps constant (faux hash pour les e-mails inconnus), mots de passe ≤ 72 octets (pas de troncature silencieuse), code de récupération de 80 bits, à usage unique, 30 min, par IP limité à 5 essais / 15 min.
- Autorisation **toujours côté serveur** : l'appartenance à une conversation est la seule porte d'accès au chat ; un profil n'est visible que s'il est découvrable, apparié ou a liké le demandeur, et jamais en cas de blocage.
- Rate limiting (auth, upload, envoi de messages, signalements, swipes), CORS en liste blanche, en-têtes de sécurité, proxys non crus par défaut.
- Photos : types vérifiés (JPEG/PNG/WebP), 8 Mo max, dimensions bornées (anti « decompression bomb »), **réencodage systématique** (métadonnées EXIF/GPS et charges cachées supprimées), stockage hors webroot, **URLs signées HMAC expirantes** délivrées uniquement aux personnes autorisées.
- Localisation : coordonnées arrondies à ~1 km **avant** stockage, jamais renvoyées aux autres ; distance affichée grossière (1 km sous 50 km, pas de 5 km au-delà) ; position effaçable ; distance masquable.
- Suppression de compte : cascade SQL + suppression des fichiers. Aucun secret côté application mobile.

## Décisions techniques principales

- **On construit sur l'existant** : Go/Gin/pgx/JWT et l'app Expo sont conservés ; supprimés car hors produit : facturation Stripe, organisations multi-tenant, posts / commentaires / votes, Tamagui (jamais utilisé par les composants), scripts PowerShell de smoke obsolètes.
- **Migrations** : fichiers SQL embarqués + table `schema_migrations` (le runner précédent rejouait tout à chaque démarrage).
- **Pas de vérification d'e-mail** : le plus simple et le plus sûr sans dépendance à un service de mail ; l'e-mail sert à la connexion et à la récupération. À ajouter avec un fournisseur SMTP si la lutte anti-faux comptes l'exige.
- **Temps réel** : WebSocket en processus (un hub mémoire). Pour plusieurs instances API, relayer via PostgreSQL `LISTEN/NOTIFY` ou Redis (interface `realtime.Publisher` déjà isolée). REST reste la source de vérité : à la reconnexion le client resynchronise.
- **Pas de notifications push natives** (nécessitent des comptes Apple / Google / Expo) : notifications in-app + temps réel. Point d'extension : `notifications` + `realtime.Publisher`.
- **Un swipe est définitif** (idempotent) ; seul « revoir les profils passés » les efface. Un blocage supprime le match et la conversation pour les deux.
- **Pas d'écran de like reçus** (« qui m'a liké ») : c'est le ressort classique des paywalls ; ici, ceux qui vous ont liké sont simplement montrés en premier dans la découverte.
- **Limites** : le seed est de démonstration ; le filtrage de contenu des messages / photos et le tableau de bord de modération (traitement des `reports`) ne sont pas inclus — les signalements sont stockés avec un statut `open | reviewed | dismissed`.

## Licence et contribution

Voir `CONTRIBUTING.md` et `CODE_OF_CONDUCT.md`.
