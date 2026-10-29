# Sillage

Sillage est une application de rencontre **100 % gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de limite artificielle de likes, pas de publicité.

Deux personnes peuvent créer un compte, composer leur profil (photos, bio, centres d'intérêt), découvrir des profils proches, s'aimer mutuellement (« match »), discuter en temps réel, gérer leur confidentialité, bloquer ou signaler quelqu'un et supprimer leur compte.

Le dépôt est un monorepo : une API Go, une base PostgreSQL et une application mobile React Native (Expo).

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation du refresh token, déconnexion, **récupération par code e-mail**, changement de mot de passe, **suppression du compte** (toutes les données associées sont effacées). |
| Profil | Prénom, âge (≥ 18 ans vérifié côté serveur), genre, ville, bio, jusqu'à 10 centres d'intérêt, visibilité et affichage de la distance configurables. |
| Photos | Jusqu'à 6 photos, ordre et photo principale, remplacement, suppression. Chaque image est décodée, **redimensionnée et ré-encodée en JPEG côté serveur** (métadonnées EXIF/GPS supprimées) ; les fichiers ne sont servis qu'aux utilisateurs authentifiés autorisés. |
| Découverte | Pile de profils avec swipe (et boutons accessibles), photos multiples, profil complet. Filtres mutuels : genres recherchés, tranches d'âge, distance. Les profils déjà traités, bloqués ou masqués n'apparaissent jamais. |
| Matching | Like / passer. Like réciproque = match, créé de façon atomique (verrou par paire : un seul match même si les deux personnes likent au même instant). Idempotent, sans doublon. |
| Chat | Conversation créée au match, messages texte temps réel (WebSocket), heure, lu / non lu, accusés « Vu », suppression locale d'une conversation, annulation de match. |
| Notifications | Match et nouveaux messages (regroupés par conversation), pastilles sur les onglets, marquage lu. |
| Sécurité | Blocage, signalement (avec blocage optionnel), limitation de débit, validation stricte, autorisations côté serveur. |
| Géolocalisation | Position approximative demandée uniquement à la demande de l'utilisateur ; coordonnées **arrondies à ~1 km** avant stockage ; seule une **distance approximative** est exposée. |

## Pile technique

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256, bcrypt, SQL explicite (pas d'ORM), WebSocket (`gorilla/websocket`), `golang.org/x/image` (redimensionnement, WebP).
- **Mobile** : Expo SDK 54 / React Native 0.81, TypeScript strict, React Navigation, TanStack Query, `expo-image`, `expo-image-picker`, `expo-location`, `expo-secure-store`.
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS / Android)
│  └─ src/
│     ├─ api/                   Client REST (refresh automatique du token), un module par domaine
│     ├─ state/                 Auth, temps réel (WebSocket), toasts, caches
│     ├─ hooks/                 useDeck (pile de découverte), requêtes, actions asynchrones
│     ├─ components/            Design system (Button, TextField, SwipeCard, PhotoGrid…)
│     ├─ screens/               auth/ · onboarding/ · main/
│     ├─ navigation/            Stack racine + onglets
│     └─ theme/                 Palette claire/sombre « Sillage »
├─ services/api/
│  ├─ cmd/api                   Point d'entrée de l'API
│  ├─ cmd/seed                  Données de démonstration (dev uniquement)
│  └─ internal/
│     ├─ platform/              config, db (migrations versionnées), middleware, realtime, storage, mailer, httpx
│     └─ features/              auth · profiles · photos · discovery · chat · notifications · safety
├─ infra/docker-compose.yml     PostgreSQL + API
├─ scripts/smoke.mjs            Parcours complet de bout en bout contre une API qui tourne
└─ docs/ARCHITECTURE.md         Détails, modèle de données, décisions
```

Chaque feature backend suit `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`). Les handlers sont fins, le SQL ne vit que dans les repositories, les règles métier dans les services. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Démarrage rapide

Prérequis : Docker, Go 1.24+, Node 22+.

```bash
cp .env.example .env
# Renseignez JWT_SECRET (>= 32 caractères aléatoires) :  openssl rand -hex 32
make up        # PostgreSQL + API sur http://localhost:18080 (migrations appliquées au démarrage)
make seed      # 12 comptes de démonstration (facultatif, développement uniquement)
```

Vérifiez l'API : `curl http://localhost:18080/health` → `{"status":"ok"}`.

### Lancer l'application mobile

```bash
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start
```

- **Téléphone physique** : n'utilisez **jamais** `localhost`. Mettez l'IP LAN de votre ordinateur (téléphone et ordinateur sur le même réseau) et vérifiez que rien d'autre n'occupe le port : `curl http://<IP_LAN>:18080/health`.
- Émulateur Android : l'application utilise `10.0.2.2:18080` par défaut en développement ; simulateur iOS : `localhost:18080`.
- Si le port 18080 est pris, changez `API_HOST_PORT` dans `.env` et `EXPO_PUBLIC_API_URL` en conséquence.

Comptes de démonstration (après `make seed`) : `camille@demo.invalid`, `lucas@demo.invalid`, … avec le mot de passe `SEED_PASSWORD` (`demo-password-123` par défaut). Le domaine `demo.invalid` est réservé et non délivrable : ces comptes ne peuvent pas être confondus avec de vrais utilisateurs. Le seed refuse de s'exécuter si `APP_ENV=production`.

### Sans Docker

```bash
# PostgreSQL local, puis :
export DATABASE_URL=postgresql://app:app@localhost:5432/app?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
cd services/api && go run ./cmd/api      # écoute sur :8080
```

## Variables d'environnement

Toutes les variables sont documentées dans [`.env.example`](.env.example).

| Variable | Obligatoire | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256, ≥ 32 caractères, valeurs « placeholder » refusées. |
| `DATABASE_URL` | oui | Chaîne de connexion PostgreSQL. |
| `PORT` | non (8080) | Port HTTP de l'API. |
| `APP_ENV` | non | `production` impose `SMTP_HOST` et interdit le seed. |
| `UPLOAD_DIR` | non | Dossier des photos (monter un volume en production). |
| `ALLOWED_ORIGINS` | non | Origines navigateur autorisées (CORS + WebSocket). Les apps natives n'en ont pas besoin. |
| `TRUSTED_PROXIES` | non | Proxys dont `X-Forwarded-For` est cru (limitation de débit par IP). |
| `SMTP_*` | prod | Envoi des codes de récupération. En développement sans SMTP, le code est écrit dans les logs de l'API. |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API vue **depuis le téléphone**. |

Aucun secret n'est présent dans le code ni dans le frontend ; `.env` est ignoré par Git.

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées **au démarrage de l'API**, dans l'ordre, chacune dans une transaction et enregistrée dans `schema_migrations` (un verrou consultatif évite les courses entre instances). Pour ajouter une évolution : créer `00N_nom.sql`, ne jamais modifier une migration déjà publiée.

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `photos`, `interests`, `user_interests`, `swipes` (likes et passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Contraintes d'unicité, clés étrangères en `ON DELETE CASCADE` (la suppression d'un compte efface tout), `CHECK` sur les domaines de valeurs, et index sur les chemins chauds (`messages (conversation_id, created_at)`, `notifications (user_id, created_at)`, `blocks`, `swipes (to_user_id, from_user_id)`, etc.).

> Changement incompatible : ce schéma remplace celui de l'ancien boilerplate SaaS. Sur une base de développement existante, recréez la base (`docker compose … down -v`).

## Tests, lint et build

```bash
make test        # API (unitaires + intégration) et mobile (Jest)
make lint        # gofmt, go vet, tsc, eslint, prettier
make build       # binaire API + bundle Android
```

- **API** : `cd services/api && DATABASE_URL_TEST=postgres://… go test ./...`. Les tests d'intégration (auth, profils, photos, découverte, likes/matchs concurrents, chat, permissions, blocage, signalement, WebSocket) **vident la base `DATABASE_URL_TEST`** : utilisez une base dédiée. Sans cette variable ils sont ignorés.
- **Mobile** : `cd apps/mobile && npm ci && npm run typecheck && npm run lint && npm test`.
- **Parcours de bout en bout** : `API_URL=http://localhost:18080 node scripts/smoke.mjs` joue inscription → profil → photo → découverte → like → match → chat temps réel → notifications → blocage/signalement → suppression de compte sur une vraie API.

La CI GitHub Actions exécute tout cela (jobs `api`, `mobile`, `e2e`).

## Déploiement

1. Construire l'image : `docker build -t sillage-api services/api`.
2. Fournir PostgreSQL, `JWT_SECRET`, `DATABASE_URL`, `APP_ENV=production`, `SMTP_*`, et **un volume persistant monté sur `/data/uploads`** (photos).
3. Placer l'API derrière un proxy TLS ; renseigner `TRUSTED_PROXIES` avec l'IP du proxy.
4. Le stockage des photos est un disque local. Pour plusieurs répliques, implémenter l'interface `storage.Storage` (S3…) ; le hub WebSocket et le limiteur de débit sont en mémoire et pensés pour une instance (voir ci-dessous).
5. Application mobile : définir `EXPO_PUBLIC_API_URL` (HTTPS) au moment du build (EAS Build ou `expo prebuild`).

## Décisions techniques principales

- **On construit sur l'existant** : Go/Gin/pgx/JWT/Expo conservés. Supprimés car sans objet : facturation Stripe, organisations multi-tenant, posts, commentaires, fichiers génériques, composants Tamagui inutilisés, scripts PowerShell.
- **Likes et passes dans une seule table** (`swipes`) : l'exclusion des profils déjà vus est un simple `NOT EXISTS` ; le match se décide dans une transaction sous verrou consultatif de la paire.
- **Recommandation isolée** : le tri des candidats est une expression SQL unique (`discovery/ranking.go`) — profils qui vous ont déjà aimé d'abord, puis proximité, centres d'intérêt communs, activité récente, un peu d'aléatoire. Remplaçable sans toucher au reste.
- **Distance sans PostGIS** : fonction SQL haversine + fenêtre de latitude indexée. Suffisant jusqu'à de gros volumes ; PostGIS est la suite logique.
- **Confidentialité de la position** : coordonnées arrondies à ~1 km à l'écriture, jamais renvoyées par l'API, distance arrondie (1 km puis paliers de 5 km).
- **Photos privées** : ré-encodage systématique, clés aléatoires, accès authentifié et filtré (bloqué ou masqué = 404), cache `private, immutable`.
- **Blocage = fin de la relation** : le match et la conversation sont supprimés pour les deux (cascade) ; l'autre personne voit seulement que le match a disparu.
- **Suppression locale d'un message/conversation** : horodatage `cleared_at` par participant ; l'historique n'est effacé que de son côté et la conversation ressurgit à la réception d'un nouveau message.
- **Récupération de compte** : code à 8 caractères valable 30 minutes, haché en base, 5 essais maximum, usage unique, toutes les sessions sont révoquées ; la réponse ne révèle jamais si l'e-mail existe.
- **Sessions révocables** : le jeton d'accès (15 min) est vérifié contre la session en base ; déconnexion ou suppression de compte coupent l'accès immédiatement.
- **Notifications** : in-app et temps réel. Les notifications push (APNs/FCM) ne sont **pas** incluses : elles exigent des comptes développeur et des clés propres à chaque déploiement.
- **Limites connues** : hub WebSocket et limiteur de débit en mémoire (une instance) ; signalements stockés mais sans interface d'administration (consultables en SQL : table `reports`) ; les messages d'une conversation supprimée par blocage ne sont pas conservés comme preuve.

## Licence et contribution

Voir [CONTRIBUTING.md](CONTRIBUTING.md) et [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
