# Lien — application de rencontre gratuite

Lien est une application de rencontre complète, **100 % gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de limite artificielle de likes, pas de publicité.

Deux personnes peuvent créer un compte et un profil, ajouter des photos, découvrir des profils proches, liker ou passer, obtenir un match en cas d'intérêt réciproque, discuter en temps réel, recevoir des notifications, régler leurs préférences et leur confidentialité, bloquer / signaler, et supprimer leur compte.

## Stack

| Couche | Technologies |
| --- | --- |
| API | Go 1.24, Gin, PostgreSQL 16, `pgxpool` (SQL explicite, sans ORM), JWT HS256 + refresh tokens rotatifs, WebSocket (`gorilla/websocket`) |
| Client | Expo SDK 54 / React Native 0.81 (iOS, Android, web), TypeScript strict, React Navigation 7, TanStack Query |
| Infra | Docker, Docker Compose, GitHub Actions |
| Tests | `go test` (unitaires + intégration sur PostgreSQL), Jest + Testing Library, Playwright (parcours navigateur) |

## Architecture

```text
services/api
├─ cmd/api            serveur HTTP unique (Gin) + intégration des modules
├─ cmd/seed           données de démonstration (domaine réservé demo.invalid)
└─ internal
   ├─ platform        config, db (+ migrations suivies), middleware (auth, rate limit, CORS…),
   │                  realtime (WebSocket), mailer, signing (URLs signées), logger
   └─ features        handler -> service -> repository, par module :
      auth · profiles · photos · matching · chat · notifications · safety
apps/mobile           client Expo (src/api, auth, realtime, components, screens, navigation)
e2e                   scénario Playwright : inscription → match → chat → blocage
```

Principales règles : handlers fins, logique métier dans les services, SQL uniquement dans les repositories, un seul pool de connexions, aucun appel réseau direct depuis l'UI (tout passe par `src/api`).

Détails et décisions : [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation et lancement local

Prérequis : Docker (ou PostgreSQL 14+), Go 1.24+, Node 20+.

```bash
cp .env.example .env
# renseigner JWT_SECRET (au moins 32 caractères) : openssl rand -hex 32

# 1. API + base de données (l'API applique les migrations au démarrage)
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# 2. Données de démonstration (facultatif, jamais en production)
docker compose run --rm -e DATABASE_URL=postgresql://app:app@postgres:5432/app?sslmode=disable api ./seed
# comptes : alex@demo.invalid, camille@demo.invalid… mot de passe DemoPass123
# suppression : ... api ./seed -reset

# 3. Application
cd apps/mobile
cp .env.example .env     # EXPO_PUBLIC_API_URL
npm ci
npm run web              # ou npm start puis Expo Go (iOS / Android)
```

Sans Docker : démarrer PostgreSQL, puis

```bash
cd services/api
export DATABASE_URL='postgresql://app:app@localhost:5432/app?sslmode=disable' JWT_SECRET="$(openssl rand -hex 32)" PORT=18080
go run ./cmd/api          # migrations appliquées automatiquement
go run ./cmd/seed         # facultatif
```

### Téléphone physique

Ne jamais utiliser `localhost` : `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`. Vérifier qu'aucun autre service n'occupe le port (`docker compose ps`, `curl http://<IP_LAN>:18080/health`). Pour l'émulateur Android l'URL par défaut est `http://10.0.2.2:18080`. Pour le web, ajouter l'origine de l'application à `ALLOWED_ORIGINS` (ex. `http://localhost:8081`).

## Variables d'environnement

| Variable | Rôle |
| --- | --- |
| `JWT_SECRET` | **Obligatoire**, ≥ 32 caractères, valeurs de type « change-me » refusées |
| `DATABASE_URL` | **Obligatoire** (fournie par Docker Compose) |
| `PORT` | Port de l'API (défaut `8080`, publié sur `18080` par Compose) |
| `APP_ENV` | `development` (défaut) ou `production` (exige SMTP) |
| `ALLOWED_ORIGINS` | Origines navigateur autorisées (CORS, WebSocket) |
| `TRUSTED_PROXIES` | Proxies dont `X-Forwarded-For` est cru (rate limiting par IP) |
| `AUTH_RATE_LIMIT` | Appels login/inscription/reset par IP et 10 min (défaut 30) |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | Envoi des codes de récupération. Sans `SMTP_HOST` (dev), le code est écrit dans le log de l'API |
| `EXPO_PUBLIC_API_URL` | (client) URL publique de l'API — jamais de secret côté client |

## Base de données et migrations

Les migrations SQL sont dans `services/api/internal/platform/db/migrations`, embarquées dans le binaire et appliquées par ordre, une transaction chacune, au démarrage (table `schema_migrations`, verrou consultatif pour les démarrages concurrents). Pour ajouter une évolution : créer `00N_nom.sql` (ne jamais modifier une migration déjà appliquée).

Tables : `users`, `sessions`, `password_resets`, `profiles`, `user_preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `messages`, `conversation_states`, `blocks`, `reports`, `notifications`. Contraintes, clés étrangères en cascade et index (découverte, messages par match, notifications non lues…) sont définis dans les migrations.

## Tests et qualité

```bash
# API (PostgreSQL requis pour les tests d'intégration, isolés dans un schéma temporaire)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'
gofmt -l . && go vet ./... && go test ./... && go build ./cmd/api ./cmd/seed

# Client
cd apps/mobile
npm run typecheck && npm run lint && npm run format:check && npm test && npm run build

# Parcours navigateur (API + build web servis, voir en-tête de e2e/web-flow.mjs)
cd e2e && npm install && npx playwright install chromium && npm test
```

Les tests d'intégration couvrent : santé, authentification (jetons forgés/expirés, rotation, récupération de compte), profils et règles d'âge, photos (uploads dangereux), découverte et préférences, likes, match (y compris likes simultanés), chat et autorisations, temps réel, notifications, blocage, signalement, suppression de compte en cascade, limites de débit.

## Déploiement

1. Construire l'image : `docker build -t lien-api services/api` (binaires `api` et `seed`, utilisateur non-root, healthcheck).
2. Fournir `DATABASE_URL` (PostgreSQL géré), `JWT_SECRET`, `APP_ENV=production`, SMTP, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`.
3. Placer l'API derrière un proxy TLS qui autorise les WebSocket (`/ws`). L'API est sans état hors temps réel : pour plusieurs instances, remplacer le hub en mémoire (`platform/realtime`) par PostgreSQL `LISTEN/NOTIFY` ou un broker, et ajouter une limite de débit en bordure.
4. Client : `npm run build` (web statique dans `dist/`) ou EAS Build pour les stores, avec `EXPO_PUBLIC_API_URL` pointant sur l'API HTTPS.

## Décisions techniques

- **Conservé** : Go/Gin/pgx, sessions à refresh token rotatif, structure feature-first. **Supprimé** : Stripe/abonnements, organisations multi-tenant, posts/commentaires/votes, fichiers génériques, Tamagui (inutilisé).
- **Match** : paire non ordonnée `user_a < user_b` unique ; verrou consultatif par paire pour qu'un like réciproque simultané crée exactement un match. L'identifiant du match est l'identifiant de la conversation. Un swipe est définitif (pas de doublon, jamais re-proposé).
- **Éligibilité côté serveur** : le même prédicat SQL sert à proposer les profils et à autoriser un swipe (genre, âge et distance **réciproques**, visibilité, photo, blocages). Le classement place d'abord ceux qui vous ont liké, puis l'activité récente : point d'extension de l'algorithme de recommandation (`matching/repository.go`).
- **Localisation** : coordonnées arrondies à ~1 km côté serveur, jamais exposées ; distance affichée par paliers de 5 km ; mise à jour limitée en fréquence ; option de masquer la distance.
- **Âge** : ≥ 18 ans vérifié côté serveur ; la date de naissance n'est plus modifiable après création.
- **Photos** : JPEG/PNG détectés par contenu, dimensions vérifiées avant décodage, orientation EXIF appliquée, ré-encodage JPEG ≤ 1080 px (métadonnées supprimées), 6 photos max, stockées en base (cohérence avec sauvegardes et suppression de compte), servies via URL signée à durée limitée (le composant image n'a pas besoin d'en-tête d'authentification).
- **Chat** : réservé aux matchs actifs ; suppression « locale » = historique masqué pour soi seulement ; retirer un match ou bloquer ferme la conversation pour les deux.
- **Temps réel** : WebSocket, authentification par première trame (jamais de jeton dans l'URL), REST pour toute écriture ; reconnexion avec backoff et resynchronisation.
- **Modération** : les signalements sont stockés (`reports`, statut `open`) ; leur traitement et la suspension (`users.status = 'suspended'`) se font en base pour l'instant — pas d'interface d'administration.
- **Récupération de compte** : code à 8 chiffres par email (30 min, 5 essais, 1 par minute et par compte), toutes les sessions sont révoquées après réinitialisation.
- **Langue** : interface en français (textes dans les écrans).

## Limites connues

- Hub temps réel en mémoire (mono-instance), limites de débit par processus.
- Notifications = liste dans l'application + WebSocket ; pas de notifications push système.
- L'image Docker et le scénario Playwright en CI n'ont pas pu être exécutés dans l'environnement de développement automatisé (pas de démon Docker) ; le scénario Playwright a été exécuté localement contre l'API et le build web.
