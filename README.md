# Lumo — rencontres gratuites, pour de vrai

Lumo est une application de rencontre **100 % gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de likes limités artificiellement, aucune fonctionnalité essentielle verrouillée, aucune publicité obligatoire. Le code ne contient d'ailleurs aucun module de paiement.

Deux personnes peuvent : créer un compte, composer leur profil (photos, bio, centres d'intérêt, préférences), découvrir des profils compatibles, liker ou passer, obtenir un **match** quand l'intérêt est réciproque, discuter **en temps réel**, recevoir des notifications, gérer leur confidentialité, bloquer / signaler quelqu'un et supprimer leur compte (et toutes leurs données).

Le dépôt est un monorepo : une API Go et un client Expo (iOS, Android **et web**).

```text
apps/mobile/     Client Expo / React Native / TypeScript (aussi exportable en web)
services/api/    API Go (Gin, pgxpool, PostgreSQL, sans ORM)
infra/           Variante de docker-compose à lancer depuis infra/
scripts/         smoke-api.mjs : test de bout en bout d'une API en marche
docs/            ARCHITECTURE.md, API.md (contrat HTTP complet)
```

## Stack

| Couche | Choix |
| --- | --- |
| API | Go 1.24, Gin, pgx/pgxpool (SQL explicite), JWT HS256 + refresh tokens rotatifs, bcrypt |
| Base | PostgreSQL 16 (migrations SQL embarquées, suivies dans `schema_migrations`) |
| Temps réel | WebSocket (`gorilla/websocket`) avec tickets à usage unique |
| Images | `golang.org/x/image` (décodage WebP) ; toutes les photos sont ré-encodées en JPEG |
| Client | Expo 54, React Native 0.81, React Navigation 7, TanStack Query, TypeScript strict |
| Tests | `go test` (intégration sur vrai PostgreSQL), Jest + Testing Library, Playwright (E2E web) |
| Infra | Docker, Docker Compose, GitHub Actions |

Architecture détaillée et décisions : [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Contrat de l'API : [`docs/API.md`](docs/API.md).

## Démarrage rapide (Docker)

Prérequis : Docker, Node 22+ (pour le client).

```bash
cp .env.example .env
# renseigner JWT_SECRET (>= 32 caractères aléatoires) :
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(openssl rand -hex 32)/" .env

docker compose up --build -d           # PostgreSQL + API sur http://localhost:18080
curl http://localhost:18080/health      # {"status":"ok"}

# données de démonstration (optionnel, jamais en production)
docker compose exec api ./seed          # crée 18 comptes demo01..demo18@seed.lumo.invalid

cd apps/mobile
cp .env.example .env                    # EXPO_PUBLIC_API_URL=http://localhost:18080
npm ci
npm run web                             # ou: npm start puis i / a / QR code Expo Go
```

Compte de démonstration : `demo01@seed.lumo.invalid` / `DemoPass123!` (mot de passe modifiable via `SEED_PASSWORD`). Les comptes de démo ont `users.is_demo = TRUE` et un domaine réservé (`.invalid`) qui ne peut jamais recevoir d'e-mail ; `docker compose exec api ./seed -reset` les supprime (photos comprises). Le seed refuse de s'exécuter si `APP_ENV=production`.

> **Téléphone physique** : `localhost` désigne le téléphone. Utiliser l'IP LAN de la machine :
> `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080` dans `apps/mobile/.env`, puis vérifier avec
> `curl http://<IP_LAN>:18080/health` et `docker compose ps`. Si le port 18080 est déjà pris, changer le port hôte
> dans `docker-compose.yml` / `infra/docker-compose.yml` **et** `EXPO_PUBLIC_API_URL`.

## Variables d'environnement

Toutes sont documentées dans [`.env.example`](.env.example). Aucun secret n'est jamais versionné ni envoyé au client.

| Variable | Obligatoire | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | Clé HS256, ≥ 32 caractères, refusée si elle ressemble à un exemple |
| `DATABASE_URL` | oui | URL PostgreSQL |
| `PORT` | non (8080) | Port HTTP |
| `APP_ENV` | non (`production`) | `development` : CORS permissif, codes de reset journalisés, seed autorisé |
| `ALLOWED_ORIGINS` | non | Origines navigateur autorisées (CSV). Vide en production = aucune |
| `TRUSTED_PROXIES` | non | Proxys dont `X-Forwarded-For` est cru (utile pour le rate limiting derrière un LB) |
| `UPLOAD_DIR` | non | Dossier des photos (volume persistant en production) |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | non | E-mails de récupération de compte. Sans SMTP, aucun envoi en production |
| `SEED_PASSWORD` | non | Mot de passe des comptes de démo |
| `EXPO_PUBLIC_API_URL` | client | URL de l'API vue **depuis l'appareil** (`apps/mobile/.env`) |

## Développement local sans Docker

```bash
# PostgreSQL local (exemple)
docker run -d --name lumo-pg -e POSTGRES_USER=app -e POSTGRES_PASSWORD=app -e POSTGRES_DB=app -p 5432:5432 postgres:16-alpine

cd services/api
export DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)" APP_ENV=development PORT=18080
go run ./cmd/api            # applique les migrations au démarrage
go run ./cmd/seed           # données de démo (optionnel) ; go run ./cmd/seed -reset pour les retirer
```

### Migrations

Les fichiers `services/api/internal/platform/db/migrations/NNN_*.sql` sont embarqués dans le binaire et appliqués **automatiquement au démarrage**, dans l'ordre, chacun dans une transaction, avec un verrou consultatif (plusieurs instances peuvent démarrer en même temps). Les migrations appliquées sont consignées dans `schema_migrations` ; ne modifiez jamais une migration publiée, ajoutez-en une nouvelle. Les migrations `002`–`013` sont l'historique du boilerplate d'origine, `014` en supprime les tables SaaS.

## Tests et qualité

```bash
# API : une base PostgreSQL vide suffit (les tests créent et nettoient leurs propres données)
cd services/api
export DATABASE_URL_TEST='postgres://app:app@localhost:5432/app?sslmode=disable'
gofmt -l .                    # ne doit rien afficher
go vet ./...
go test -race ./...           # ~3 min avec -race (bcrypt), ~15 s sans
go build ./cmd/api ./cmd/seed

# API en marche : parcours complet inscription → onboarding → like → match → chat temps réel → blocage
API_URL=http://localhost:18080 node scripts/smoke-api.mjs

# Client
cd apps/mobile
npm ci
npm run typecheck             # tsc --noEmit
npm run lint                  # ESLint + Prettier (npm run format pour corriger)
npm test                      # Jest
E2E_API_URL=http://localhost:18080 npm run e2e:web   # Playwright : deux navigateurs, inscription → match → chat
```

Les tests d'intégration sont ignorés (skip) si `DATABASE_URL_TEST`/`DATABASE_URL` n'est pas défini ; la CI les exécute contre un vrai PostgreSQL. Ils couvrent notamment : authentification et rotation des tokens, récupération de mot de passe (y compris le verrouillage après 5 essais), validation des profils (majorité, coordonnées arrondies), pipeline photo (formats, bombes de décompression, EXIF, permissions), préférences mutuelles de découverte, likes simultanés (un seul match), idempotence des swipes, désappairage, permissions du chat (un tiers reçoit toujours 404), pagination, suppression locale, blocage, signalement, temps réel (tickets à usage unique, événements ciblés), suppression de compte (données et fichiers).

## Build et déploiement

```bash
docker build -t lumo-api services/api          # image alpine non-root, HEALTHCHECK /health
cd apps/mobile && npm run build:web            # export web statique (EXPO_PUBLIC_API_URL est figée à la compilation)
```

En production : `APP_ENV=production`, un `JWT_SECRET` aléatoire, `DATABASE_URL` pointant vers un PostgreSQL géré, **un volume persistant monté sur `UPLOAD_DIR`**, TLS devant l'API (le WebSocket passe en `wss://` automatiquement), `ALLOWED_ORIGINS` = origine du client web, `TRUSTED_PROXIES` = IP de votre load balancer, et un serveur SMTP. Applications natives : `eas build` (Expo) avec `EXPO_PUBLIC_API_URL` en HTTPS.

Limites assumées (voir `docs/ARCHITECTURE.md`) : le hub temps réel, les tickets et les limiteurs de débit vivent en mémoire d'une instance ; les photos sont sur disque derrière l'interface `storage.Store`. Pour plusieurs instances, remplacer ces briques (Redis / LISTEN-NOTIFY, stockage S3) — les interfaces sont déjà en place.

## Principales décisions produit et techniques

* **Gratuit par construction** : aucun module de facturation, aucune limite de swipes ; seuls des plafonds anti-script existent (240 swipes/min, 60 messages/min…).
* **Âge** : date de naissance obligatoire, 18–99 ans vérifiés côté serveur (et par contraintes SQL sur les préférences). Pas de vérification d'e-mail ni d'identité en V1.
* **Confidentialité de la position** : seules des coordonnées arrondies à 0,01° (~1 km) sont stockées ; les autres membres voient une distance par paliers (1, 2, 5, 10, 20, 50, 100… km) que l'on peut masquer ; jamais d'adresse ni de GPS exact.
* **Découverte** : compatibilité **mutuelle** (genre, âge, distance), profils déjà traités / bloqués / masqués / sans photo exclus ; les personnes qui vous ont déjà liké passent en premier. L'algorithme est isolé derrière l'interface `Ranker`.
* **« Qui m'a liké » est gratuit** (`/likes/received`), contrairement à de nombreuses applications.
* **Match** : verrou par paire → exactement un match et une conversation même si deux personnes se likent au même instant ; un couple désapparié ne se re-matche jamais.
* **Chat** : une conversation par match, accusés de lecture par « filigrane » de lecture, suppression locale (l'historique est masqué pour vous seulement et la conversation réapparaît à un nouveau message).
* **Notifications** : in-app (liste + compteur + temps réel). Pas de push natif (Expo Push/APNs/FCM) en V1 : cela exige des identifiants propres à chaque déploiement.
* **Photos** : 6 maximum, ordre et photo principale modifiables, ré-encodage JPEG côté serveur (supprime métadonnées et charges cachées), fichiers servis uniquement via un endpoint authentifié qui applique les règles de visibilité / blocage.
* **Suppression de compte** : mot de passe redemandé ; photos effacées du disque, sockets fermées, cascade SQL sur toutes les données personnelles. Les signalements sont conservés (anonymisés) pour la modération.
* **Modération** : les signalements arrivent dans la table `reports` (`status = 'open'`) ; il n'y a pas encore d'interface d'administration (`SELECT * FROM reports WHERE status='open' ORDER BY created_at`).
* **Interface en français**, chaînes centralisées dans `apps/mobile/src/i18n/fr.ts`.

## Contribuer

Voir [`CONTRIBUTING.md`](CONTRIBUTING.md) et [`AGENTS.md`](AGENTS.md) (règles d'architecture : `handler → service → repository`, pas de SQL dans les handlers, un seul serveur HTTP).
