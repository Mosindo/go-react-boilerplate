# Liane — application de rencontre 100 % gratuite

Liane est une application de rencontre moderne (mobile iOS / Android via Expo) et son API Go.
**Aucun abonnement, aucun paywall, aucun boost payant, aucun like limité, aucune publicité.** Toutes les fonctionnalités sont ouvertes à tous.

| Parcours | Fonctionnalités |
|---|---|
| Compte | inscription, connexion, récupération par code e‑mail, changement de mot de passe, suppression définitive du compte |
| Profil | prénom, âge (18+ vérifié côté serveur), genre, bio, centres d'intérêt, 1 à 6 photos (ordre, photo principale, remplacement, suppression), localisation approximative |
| Découverte | cartes avec swipe ou boutons, carrousel de photos, profil complet, « Qui m'a liké » (gratuit), préférences mutuelles (genre, âge, distance) |
| Matching | like / passer, match automatique si l'intérêt est réciproque, sans doublon, un profil jugé ne revient jamais |
| Chat | conversation ouverte au match, temps réel (WebSocket), horodatage, non‑lus, accusé de lecture, suppression locale, défaire le match |
| Sécurité | blocage, signalement, masquage du profil, notifications, limitation de débit, URLs de photos signées |

## Stack

- **API** : Go 1.24, Gin, PostgreSQL 16 (pgx/pgxpool, pas d'ORM), JWT HS256 + refresh tokens, bcrypt, gorilla/websocket
- **Mobile** : Expo SDK 54, React Native, TypeScript strict, React Navigation, TanStack Query
- **Infra** : Docker, Docker Compose, GitHub Actions

Architecture détaillée : [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) · contrat d'API : [`docs/API.md`](docs/API.md).

## Démarrage rapide (Docker)

Prérequis : Docker, Node 20+.

```bash
cp .env.example .env
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(openssl rand -hex 32)/" .env   # secret aléatoire (≥ 32 caractères)

docker compose up --build -d                       # PostgreSQL + API sur http://localhost:18080
curl http://localhost:18080/health                 # {"status":"ok"}

# données de démonstration (facultatif, voir plus bas)
docker compose exec api ./seed -n 12

# client mobile
cd apps/mobile
cp .env.example .env                               # EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080
npm ci
npx expo start
```

> **Téléphone physique** : n'utilisez jamais `localhost`. Mettez l'IP de votre machine sur le réseau local dans `EXPO_PUBLIC_API_URL`
> (`http://192.168.x.x:18080`). Émulateur Android : `http://10.0.2.2:18080`. Vérifiez `curl http://<IP_LAN>:18080/health` avant de lancer l'app.
> Si le port 18080 est pris, changez le mapping dans `docker-compose.yml` et `EXPO_PUBLIC_API_URL`.

## Variables d'environnement

Voir [`.env.example`](.env.example) (racine) et `apps/mobile/.env.example`. **Ne commitez jamais** de fichier `.env`.

| Variable | Obligatoire | Description |
|---|---|---|
| `JWT_SECRET` | oui | secret ≥ 32 caractères (`openssl rand -hex 32`), refusé s'il ressemble à un exemple |
| `DATABASE_URL` | oui | URL PostgreSQL (fournie par docker compose) |
| `PORT` | non | défaut `8080` |
| `APP_ENV` | non | `development` (défaut) ou `production` (exige `SMTP_*`, désactive le seed) |
| `UPLOAD_DIR` | non | dossier des photos (défaut `./data/uploads`, volume Docker `uploads`) |
| `ALLOWED_ORIGINS` | non | origines navigateur autorisées (CORS). Inutile pour une app native |
| `TRUSTED_PROXIES` | non | IP/CIDR du reverse‑proxy dont `X-Forwarded-For` est fiable (sinon ignoré) |
| `SMTP_ADDR`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | prod | envoi des codes de récupération. Sans `SMTP_ADDR`, les codes sont **écrits dans les logs** (développement seulement) |
| `EXPO_PUBLIC_API_URL` | mobile | URL publique de l'API. Les variables `EXPO_PUBLIC_*` sont embarquées dans l'app : aucun secret ici |

## Installation locale sans Docker

```bash
# PostgreSQL 16 local
createuser -P app && createdb -O app app          # mot de passe : app
export DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export PORT=18080
cd services/api && go run ./cmd/api
```

## Base de données et migrations

Les migrations SQL (`services/api/internal/platform/db/migrations/*.sql`) sont embarquées dans le binaire et appliquées
**automatiquement au démarrage**, dans l'ordre, chacune dans une transaction, avec suivi dans la table `schema_migrations`
(rejouer ne fait rien ; un verrou évite les courses entre instances). Pour évoluer : ajouter `002_xxx.sql`, ne jamais modifier une migration déjà publiée.

Tables : `users, sessions, password_resets, profiles, preferences, photos, interests, user_interests, swipes, matches, conversations,
conversation_participants, messages, blocks, reports, notifications` (contraintes, clés étrangères en `ON DELETE CASCADE`, index sur tous les chemins chauds).

## Données de démonstration (seed)

```bash
cd services/api
DATABASE_URL=... go run ./cmd/seed -n 12                    # crée demo-1@demo.invalid … (mot de passe DemoPassw0rd, SEED_PASSWORD pour changer)
DATABASE_URL=... go run ./cmd/seed -like moi@example.com    # les profils démo ont déjà liké ce compte → match immédiat en vous likant
DATABASE_URL=... go run ./cmd/seed -purge                   # supprime tous les @demo.invalid
```

Les comptes démo utilisent le domaine réservé `demo.invalid` (impossible à confondre avec une vraie adresse), des photos de dégradés générées
(aucune vraie personne) et la commande **refuse de tourner avec `APP_ENV=production`**.

## Tests

```bash
# API (les tests d'intégration EFFACENT la base indiquée ; ils sont ignorés si la variable est absente)
cd services/api
export TEST_DATABASE_URL='postgres://app:app@localhost:5432/app_test?sslmode=disable'
gofmt -l . && go vet ./... && go test -race ./... && go build ./cmd/api ./cmd/seed

# Smoke test HTTP réel sur la stack lancée
bash scripts/smoke-api.sh http://localhost:18080

# Mobile
cd apps/mobile && npm ci && npx tsc --noEmit && npm run lint && npm test
```

Les tests d'intégration couvrent notamment : inscription → profil → découverte → like → match → conversation, likes simultanés
(exactement un match), permissions du chat (un tiers ne lit jamais une conversation), blocage/signalement, uploads dangereux, suppression de compte en cascade, WebSocket.
Un parcours UI Maestro est fourni dans `apps/mobile/e2e/maestro` (nécessite un émulateur/appareil, non exécuté en CI).

## Build et déploiement

```bash
docker build -t liane-api services/api      # image minimale, utilisateur non‑root, HEALTHCHECK /health
```

Checklist production :
1. `APP_ENV=production`, `JWT_SECRET` fort et unique, mot de passe PostgreSQL fort (`POSTGRES_PASSWORD`), `SMTP_*` renseignés.
2. Placer l'API derrière un reverse‑proxy **HTTPS** (TLS obligatoire pour l'app et le WebSocket `wss://`) et définir `TRUSTED_PROXIES`.
3. Persister le volume `uploads` (ou remplacer `media.Store` par du stockage objet) et sauvegarder PostgreSQL.
4. Mobile : `EXPO_PUBLIC_API_URL=https://api.votre-domaine.tld`, puis build EAS (`eas build`).
5. Une seule instance d'API pour le temps réel et le rate limiting (état en mémoire) ; pour passer à l'échelle, brancher `realtime.Publisher` sur Redis/NATS et le limiteur sur un store partagé.

## Principales décisions

- **Contrôle d'âge** : date de naissance obligatoire, 18+ refusé sinon (403), non modifiable ensuite ; la date n'est jamais exposée (seul l'âge l'est).
- **Localisation** : coordonnées arrondies à ~1 km avant stockage, distance affichée par paliers de 5 km, aucune coordonnée dans les réponses ; l'app fonctionne aussi sans localisation (ville saisie, pas de filtre de distance).
- **Orientation** : un profil n'apparaît que si chacun correspond aux préférences (genre, âge) de l'autre.
- **Idempotence** : liker deux fois ou passer après un like ne change rien (`alreadyProcessed`). Pas d'« annuler le swipe » : un profil jugé ne revient pas.
- **Blocage** : masque tout dans les deux sens ; les données restent pour la modération des signalements et disparaissent avec les comptes.
- **Suppression de conversation** : « locale » (masquée pour soi, réapparaît à un nouveau message) ; « défaire le match » supprime pour les deux.
- **Notifications** : in‑app (liste + temps réel). Un like annonce « Quelqu'un vous a liké » sans identité ; l'identité est visible dans « Qui m'a liké ».
- **Mots de passe** : 8‑72 caractères, lettre + chiffre, bcrypt ; récupération par code à 8 chiffres (15 min, 5 essais) qui révoque toutes les sessions.
- **Photos** : JPEG/PNG vérifiés sur le contenu, ré‑encodés en JPEG (EXIF/GPS supprimés), accès par URL signée expirante.

## Limites connues

- Pas de notifications push (Expo Push / FCM) : les notifications sont in‑app.
- Pas de back‑office de modération : les signalements sont stockés dans `reports` (`status = 'open'`) pour traitement via SQL ou un futur outil d'admin.
- Pas de vérification d'e‑mail à l'inscription ni de connexion sociale.
- Temps réel, rate limiting et stockage des photos sont conçus pour une instance unique (voir déploiement).
