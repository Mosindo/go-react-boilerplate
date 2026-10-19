# Ember — application de rencontre 100 % gratuite

Ember est une application de rencontre mobile (iOS / Android) complète, **sans abonnement, sans paywall, sans « boost » payant, sans limite artificielle de likes et sans publicité**. Tout ce qui existe dans l'application est accessible à tout le monde.

Ce dépôt contient l'application mobile (Expo / React Native), l'API (Go / Gin / PostgreSQL) et l'infrastructure Docker. Il est issu du boilerplate `go-react-saas`, dont l'architecture (`handler → service → repository`) a été conservée.

## Fonctionnalités

| Domaine | Ce que fait l'application |
| --- | --- |
| Compte | Inscription (18 ans et + confirmé), connexion, sessions avec refresh token rotatif, **récupération de compte par code e-mail**, suppression de compte définitive |
| Profil | Prénom, date de naissance (seul l'âge est visible), genre, bio, ville, 10 centres d'intérêt maximum, masquage temporaire du profil |
| Photos | Jusqu'à 6 photos, upload validé et **réencodé côté serveur**, photo principale, réordonnancement, remplacement, suppression |
| Découverte | Pile de profils (swipe ou boutons), galerie de photos, profil complet, « J'aime » / « Passer », jamais deux fois le même profil |
| Matching | Like réciproque ⇒ match unique (sans doublon, même en cas de likes simultanés), liste des personnes qui vous ont liké (gratuit), suppression de match |
| Chat | Conversation créée au match, temps réel (WebSocket), heure d'envoi, accusés de lecture (✓ / ✓✓), compteurs de non-lus, suppression locale d'une conversation |
| Notifications | Nouveau match, nouveau message (regroupés), marquage lu / tout lu, mises à jour en direct |
| Préférences | Genres recherchés, tranche d'âge, distance maximale, affichage (ou non) de sa distance |
| Sécurité | Blocage (supprime match + conversation, invisibilité mutuelle), signalement avec capture des derniers messages comme preuve, limitation de débit |
| UX | Thème clair / sombre automatique, états de chargement / vides / erreurs, retours (toasts), libellés d'accessibilité, zones tactiles ≥ 44 px |

## Pile technique

- **Mobile** : Expo SDK 54, React Native 0.81, TypeScript strict, React Navigation, TanStack Query, `expo-secure-store`, `expo-image-picker`, `expo-image-manipulator`, `expo-location`
- **API** : Go 1.24, Gin, PostgreSQL 16, `pgx/v5` (pool unique, SQL explicite, pas d'ORM), JWT HS256, bcrypt, `coder/websocket`
- **Infra** : Docker, Docker Compose, GitHub Actions

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (TypeScript)
│  └─ src/
│     ├─ api/                   Client HTTP (refresh de session mutualisé) + modules d'appels typés
│     ├─ auth/                  Session (SecureStore) et AuthProvider
│     ├─ realtime/              WebSocket (reconnexion, resynchronisation, mise à jour du cache)
│     ├─ hooks/ lib/ theme/     Requêtes, utilitaires purs, jetons de design clair / sombre
│     ├─ ui/ components/        Composants réutilisables
│     ├─ screens/ navigation/   Écrans et navigation
│     └─ __tests__/             Tests unitaires et UI (Jest + Testing Library)
├─ services/api/
│  ├─ cmd/api/                  Point d'entrée du serveur
│  ├─ cmd/seed/                 Données de démonstration (jamais en production)
│  └─ internal/
│     ├─ server/                Câblage du routeur + tests d'intégration HTTP
│     ├─ platform/              config, db (+ migrations), middleware, logger, realtime, storage, pagination
│     └─ features/              auth · profiles · matching · chat · notifications · safety
├─ docker-compose.yml           PostgreSQL + API
├─ docs/ARCHITECTURE.md         Détails d'architecture et modèle de données
└─ .env.example
```

Chaque feature suit `handler.go → service.go → repository.go` (+ `model.go`, `routes.go`) : handlers fins, règles métier dans les services, SQL uniquement dans les repositories.

## Installation

Prérequis : Docker, Go 1.24+, Node 20.19+ (22 recommandé), npm.

```bash
git clone <ce dépôt> && cd go-react-boilerplate
cp .env.example .env
# Éditez .env : définissez POSTGRES_PASSWORD et JWT_SECRET (openssl rand -hex 32)
```

### Variables d'environnement

| Variable | Obligatoire | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256, 32 caractères minimum, refusé s'il ressemble à un exemple (`openssl rand -hex 32`) |
| `DATABASE_URL` | oui (hors Docker) | URL PostgreSQL. Avec Docker Compose elle est construite à partir de `POSTGRES_*` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Compose | Identifiants de la base (le mot de passe est obligatoire) |
| `PORT` | non | Port d'écoute de l'API (défaut `8080`) |
| `APP_ENV` | non | `development` (défaut) ou `production` |
| `UPLOAD_DIR` | non | Dossier des photos (défaut `./data/uploads`, volume Docker `/data/uploads`) |
| `ALLOWED_ORIGINS` | non | Origines navigateur autorisées (CORS / WebSocket). Inutile pour une app native |
| `TRUSTED_PROXIES` | non | IP/CIDR des reverse proxies dont `X-Forwarded-For` est cru (vide = aucun) |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USERNAME` `SMTP_PASSWORD` `SMTP_FROM` | non | Envoi des codes de récupération. Sans SMTP : le code est écrit dans le journal de l'API en développement, rien n'est envoyé en production |
| `API_HOST_PORT` | non | Port publié sur la machine (défaut `18080`) |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API vue depuis le téléphone. **Aucun secret dans les variables `EXPO_PUBLIC_*`** |

## Lancement local

```bash
# 1. API + PostgreSQL (les migrations s'appliquent au démarrage)
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# 2. Données de démonstration (optionnel, développement uniquement)
docker compose exec api ./seed
#    compte : demo@demo.invalid / demo-password-123   (SEED_PASSWORD pour changer)
#    suppression : docker compose exec api ./seed -purge

# 3. Application mobile
cd apps/mobile
cp .env.example .env                       # puis réglez EXPO_PUBLIC_API_URL
npm ci
npm start                                  # ou : npm run start:phone (détecte l'IP du réseau local)
```

> **Téléphone physique** : n'utilisez jamais `localhost`. Mettez `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`. Émulateur Android : `http://10.0.2.2:18080`. Avant de dire que l'app est « prête », vérifiez qu'aucun autre service n'occupe le port (`docker compose ps`, `curl http://<IP_LAN>:18080/health`), puis déroulez le parcours : inscription → profil → photo → découverte → like → match → message → notification.

Sans Docker : `createdb`, exportez `DATABASE_URL`, `JWT_SECRET` puis `cd services/api && go run ./cmd/api`.

### Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`internal/platform/db/migrations`) et appliquées dans l'ordre au démarrage, chacune dans une transaction, avec un verrou consultatif (plusieurs instances peuvent démarrer ensemble) et un suivi dans `schema_migrations`. Pour ajouter une évolution : créer `0NN_description.sql` (ne jamais modifier une migration déjà appliquée).

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `conversations`, `conversation_members`, `messages`, `blocks`, `reports`, `notifications`. Contraintes d'unicité, clés étrangères `ON DELETE CASCADE`, `CHECK` métier et index sur tous les chemins chauds (voir `docs/ARCHITECTURE.md`).

## Tests et qualité

```bash
# API (les tests d'intégration ont besoin d'une base dédiée, ils sont ignorés sinon)
createdb app_test
cd services/api
export DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test
gofmt -l . && go vet ./... && go test -race -count=1 ./... && go build ./cmd/api

# Mobile
cd apps/mobile
npm ci && npx tsc --noEmit && npm run lint && npm run format:check && npm test

# Parcours complet contre une API qui tourne (inscription → match → chat temps réel → suppression)
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
```

- **Unitaires** : âge, validation des profils et préférences, traitement d'image, distances approximatives, curseurs, limiteur de débit, client HTTP (refresh unique), helpers d'interface.
- **Intégration HTTP + PostgreSQL** (`internal/server`) : authentification et validation des jetons (alg `none`, expiré, mauvais secret…), récupération de compte, profils, photos (fichiers dangereux), règles de découverte, likes / matchs / doublons / **concurrence**, chat et permissions, WebSocket, blocage, signalement, notifications, suppression de compte et de fichiers.
- **UI** (Testing Library) : connexion, inscription, découverte → like → match → conversation, envoi de message.
- **E2E** : `e2e:smoke` (API réelle + WebSocket) en CI ; `apps/mobile/e2e/maestro/demo-flow.yaml` pour un appareil (Maestro).

## Build et déploiement

```bash
docker build -t ember-api services/api            # image Alpine, utilisateur non-root, healthcheck
cd apps/mobile && npx expo export --platform android   # vérifie le bundle ; builds store via EAS
```

Pour la production : `APP_ENV=production`, un `JWT_SECRET` fort, HTTPS devant l'API (reverse proxy, puis renseignez `TRUSTED_PROXIES`), SMTP configuré, sauvegardes de la base **et du volume `/data/uploads`**, et `EXPO_PUBLIC_API_URL` pointant vers l'URL publique au moment du build mobile.

## Décisions techniques

- **Âge minimal 18 ans**, vérifié côté serveur à chaque écriture de profil ; la date de naissance n'est jamais renvoyée aux autres utilisateurs.
- **Géolocalisation minimale** : les coordonnées sont arrondies à ~1 km avant stockage, jamais renvoyées par l'API ; les autres voient une distance arrondie au multiple de 5 km supérieur (≤ 1 km → « moins de 1 km »). Chacun peut masquer sa distance.
- **Photos privées** : stockées hors base sous un identifiant aléatoire, servies uniquement via `GET /photos/:id` (authentification + vérification de blocage). Le serveur valide le type réel, plafonne poids (8 Mo) et dimensions, décode puis **réencode en JPEG ≤ 1280 px** : métadonnées EXIF/GPS et charges cachées disparaissent.
- **Matching** : un verrou consultatif par paire d'utilisateurs sérialise deux likes simultanés ; `UNIQUE(user_a, user_b)` avec `user_a < user_b` garantit un seul match. Une décision (like ou passe) est définitive et le profil traité n'est plus jamais reproposé. L'ordre des candidats (qui m'a déjà liké → intérêts communs → proximité) est isolé dans `Discover` pour faire évoluer l'algorithme.
- **Blocage = rupture complète** : suppression du match, de la conversation et des messages, invisibilité réciproque (découverte, profil, photos, likes). Le signalement conserve un instantané des 20 derniers messages comme preuve, même si la conversation est supprimée ensuite.
- **Temps réel** : un hub WebSocket en mémoire (authentification par premier message, jamais par URL). Pour plusieurs instances d'API, remplacer le hub par un pub/sub (Redis, `LISTEN/NOTIFY`) derrière l'interface `realtime.Publisher`. En cas de coupure, le mobile se reconnecte avec backoff et resynchronise via REST.
- **Rate limiting** en mémoire par IP (authentification, récupération) et par utilisateur (messages, photos, signalements, découverte). Les likes ne sont limités que par une garde anti-abus (300/min) qu'un humain ne peut pas atteindre.
- **Récupération de compte** : code à 10 caractères (≈ 50 bits), valable 1 h, usage unique, 5 essais, haché en base ; la réponse est identique que l'e-mail existe ou non ; toutes les sessions sont révoquées après réinitialisation.
- **Suppression de compte** : confirmée par mot de passe ; les contraintes `ON DELETE CASCADE` effacent profil, likes, matchs, messages, blocages, signalements, notifications, sessions ; les fichiers photo sont supprimés du disque.
- **Modération** : les signalements sont stockés (`reports`, statut `open`) ; aucun back-office n'est fourni — voir `SELECT * FROM reports WHERE status = 'open'` et traitez-les selon votre politique.
- **Aucune dépendance de paiement** : l'ancien module de facturation Stripe du boilerplate a été supprimé.

## Limites connues

- Pas de notifications push hors application (nécessite des comptes Apple / Google : à brancher sur `notifications` + `expo-notifications`).
- Le stockage des photos est local (disque / volume) ; pour du multi-instances, implémenter `storage.Store` pour un stockage objet (S3, GCS).
- La vérification de l'e-mail à l'inscription n'est pas imposée.
- Pas de back-office de modération.
