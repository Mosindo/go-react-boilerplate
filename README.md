# Tandem — application de rencontre 100 % gratuite

Tandem est une application de rencontre mobile (iOS / Android via Expo) adossée à une API Go.
Deux personnes créent un profil, découvrent des profils compatibles, se likent, obtiennent un **match**
quand l'intérêt est réciproque, puis discutent **en temps réel**.

**Gratuit, vraiment** : aucun abonnement, aucun paywall, aucun boost payant, aucune limite artificielle de likes,
aucune publicité. Le code ne contient aucune intégration de paiement.

## Fonctionnalités

| Domaine | Détail |
| --- | --- |
| Compte | inscription, connexion, récupération de mot de passe par code email, déconnexion, **suppression définitive du compte** |
| Profil | prénom, âge (≥ 18 ans vérifié côté serveur), genre, bio, ville, centres d'intérêt, visibilité, affichage de la distance |
| Photos | jusqu'à 6, upload JPEG/PNG, **ré-encodage serveur** (EXIF/GPS supprimés), photo principale, ordre, remplacement, suppression |
| Découverte | cartes avec swipe (ou boutons), galerie, profil complet ; filtrage mutuel genre / âge / distance |
| Matching | like / passer, match si réciproque, idempotent, sans doublon, même en cas de likes simultanés |
| Chat | conversation créée au match, temps réel (WebSocket), heure, lu / non lu, suppression locale, « annuler le match » |
| Sécurité | blocage, signalement (avec blocage optionnel), liste des personnes bloquées |
| Notifications | matchs et nouveaux messages (centre de notifications + badge + temps réel) |
| Géolocalisation | position arrondie (~1 km) à l'écriture ; les autres ne voient qu'une distance approximative |

## Stack

- **Mobile** : Expo SDK 54 · React Native 0.81 · TypeScript strict · TanStack Query · expo-secure-store · expo-image-picker · expo-location
- **API** : Go 1.24 · Gin · PostgreSQL 16 · pgx/pgxpool · JWT HS256 + refresh tokens · bcrypt · gorilla/websocket
- **Infra** : Docker, Docker Compose

## Architecture

```
apps/mobile/            application Expo (src/api, auth, features, nav, realtime, screens, ui)
services/api/
  cmd/api               serveur HTTP (un seul serveur Gin)
  cmd/seed              données de démonstration (@demo.invalid)
  internal/platform/    config, db (+migrations), middleware, ratelimit, realtime, mailer, geo, access, logger
  internal/features/    auth · profiles · photos · matching · chat · safety · notifications · users
```

Chaque feature suit `handler → service → repository` (pas de SQL dans les handlers). Détails dans
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Prérequis

- Go ≥ 1.24, Node ≥ 20 (testé avec 22), PostgreSQL 16 **ou** Docker
- Un téléphone avec Expo Go, ou un émulateur Android / simulateur iOS

## Installation et lancement local

```bash
git clone <ce dépôt> && cd go-react-boilerplate
cp .env.example .env
# éditez .env : renseignez JWT_SECRET  (openssl rand -hex 32)

# 1. API + base de données
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# 2. (optionnel) comptes de démonstration
docker compose exec api ./seed            # voir « Seed » ci-dessous

# 3. application mobile
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<IP_LAN_DE_VOTRE_MACHINE>:18080 npm start
```

> **Téléphone physique** : `localhost` désigne le téléphone lui-même. Utilisez l'IP LAN de votre machine.
> Émulateur Android : `http://10.0.2.2:18080` (valeur par défaut en dev). Simulateur iOS : `http://localhost:18080`.
> Vérifiez qu'aucun autre service n'occupe le port 18080 (`docker compose ps`, `curl http://<IP>:18080/health`),
> sinon changez le port hôte dans `docker-compose.yml` et `EXPO_PUBLIC_API_URL`.

### Sans Docker

```bash
createdb app
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
cd services/api && go run ./cmd/api        # écoute sur :8080 (PORT pour changer)
```

## Variables d'environnement

| Variable | Obligatoire | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | ≥ 32 caractères aléatoires (les valeurs « placeholder » sont refusées au démarrage) |
| `DATABASE_URL` | oui | URL PostgreSQL |
| `PORT` | non | défaut `8080` |
| `APP_ENV` | non | `development` (défaut) ou `production` |
| `UPLOAD_DIR` | non | dossier **privé** des photos, défaut `./data/uploads` (volume persistant en prod) |
| `ALLOWED_ORIGINS` | non | origines CORS (builds web uniquement) |
| `TRUSTED_PROXIES` | non | IP/CIDR des reverse proxies dont `X-Forwarded-For` est cru (limites de débit par IP). Vide = aucun |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USER` `SMTP_PASS` `SMTP_FROM` | prod | envoi des codes de récupération. Sans `SMTP_HOST` (dev), le code est écrit dans le log de l'API. **Obligatoire si `APP_ENV=production`.** |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API vue depuis l'appareil. Aucune valeur secrète ne doit être mise dans une variable `EXPO_PUBLIC_*` |
| `DATABASE_URL_TEST` | tests | base **jetable** pour les tests d'intégration (les tables sont vidées) |

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées
automatiquement au démarrage, une seule fois chacune, dans une transaction, avec un verrou consultatif
(plusieurs instances peuvent démarrer ensemble). Table de suivi : `schema_migrations`.
Pour ajouter une migration : créer `005_xxx.sql` (ne jamais modifier une migration déjà publiée).

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`,
`swipes` (likes/pass), `matches`, `conversations`, `messages`, `conversation_hidden`, `blocks`, `reports`, `notifications`.
Contraintes : clés étrangères `ON DELETE CASCADE`, unicité (un seul match par paire, un seul swipe par couple),
`CHECK` sur genres / âges / longueurs, index sur les chemins chauds (voir les fichiers de migration).

## Seed (démonstration)

```bash
cd services/api && go run ./cmd/seed         # ou : docker compose exec api ./seed
```

Crée 12 comptes `prenom@demo.invalid` (domaine réservé : impossible de les confondre avec de vrais utilisateurs),
mot de passe `demo-password-123` (ou `SEED_PASSWORD`). Idempotent. **Refuse de s'exécuter si `APP_ENV=production`.**
Exemple : `camille@demo.invalid` / `lucas@demo.invalid`.

## Tests et qualité

```bash
# API
cd services/api
gofmt -l .            # doit être vide
go vet ./...
DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable go test -count=1 ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci
npm run typecheck     # tsc --noEmit
npm run lint          # eslint
npm run format:check  # prettier
npm test              # jest (logique pure)

# Parcours critique de bout en bout contre une API qui tourne
API_URL=http://localhost:18080 npm run e2e:api
```

Sans `DATABASE_URL_TEST`, les tests d'intégration sont ignorés (`skip`) et seuls les tests unitaires s'exécutent.
Les tests d'intégration couvrent : auth (inscription, login, refresh rotatif, récupération), validation et vie privée des
profils, photos (types dangereux, limites, ordre, accès), découverte et filtres, likes / matchs / idempotence / concurrence,
chat et permissions, notifications, blocage / signalement, suppression de compte, WebSocket.
`e2e:api` rejoue inscription → profil → photo → découverte → like → match → chat temps réel → lu → blocage → suppression.

## Build et déploiement

- **API** : `docker build -t tandem-api services/api` (image Alpine, utilisateur non-root, binaires `api` et `seed`).
  Montez un volume persistant sur `/data/uploads`. Placez l'API derrière un reverse proxy TLS (le WebSocket `/ws`
  doit être proxifié avec les en-têtes `Upgrade`). Définissez `APP_ENV=production`, un `JWT_SECRET` fort et le SMTP.
- **Mobile** : `EXPO_PUBLIC_API_URL=https://api.votre-domaine eas build` (ou `npx expo prebuild`). Les permissions
  photo/localisation sont déclarées dans `app.json`.
- Les limites de débit sont en mémoire (par instance) ; avec plusieurs réplicas, ajoutez aussi une limite au proxy.
  Le hub temps réel est également en mémoire : pour plusieurs réplicas, brancher `realtime.Publisher` sur
  Postgres `LISTEN/NOTIFY` ou Redis.

## API (résumé)

Authentification : `Authorization: Bearer <accessToken>` (15 min) ; `POST /auth/refresh` fait tourner le refresh token.

| | |
| --- | --- |
| `POST /auth/register` `login` `refresh` `logout` `forgot-password` `reset-password` | comptes |
| `GET/PUT /me/profile` · `PUT /me/preferences` · `PUT /me/location` · `GET /interests` | profil |
| `GET/POST /me/photos` · `PUT /me/photos/order` · `PUT/DELETE /me/photos/:id` · `PUT /me/photos/:id/primary` · `GET /photos/:id` | photos |
| `GET /discover` · `POST /swipes` · `POST /discover/reset` · `DELETE /matches/:id` | découverte et matching |
| `GET /conversations` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` · `DELETE /conversations/:id` | chat |
| `PUT/DELETE /users/:id/block` · `POST /users/:id/report` · `GET /me/blocks` | sécurité |
| `GET /notifications` · `POST /notifications/read` · `POST /notifications/:id/read` | notifications |
| `DELETE /me` | suppression du compte (mot de passe requis) |
| `GET /ws` | WebSocket : premier message `{"type":"auth","token":"…"}` ; événements `message`, `read`, `match`, `unmatch` |

Erreurs : `{"error": "message"}` avec le code HTTP adapté. Pagination : `limit` borné (≤ 100), curseurs (`before`) pour les
conversations, messages et notifications ; la découverte n'a pas d'offset (les profils déjà traités sont exclus par la requête).

## Principales décisions techniques

- **Garder la base existante** (Go/Gin/pgx, JWT + sessions rotatives, Expo) ; **supprimer** Stripe/abonnements (incompatible
  avec « gratuit »), posts, commentaires, fichiers génériques et organisations multi-tenant (hors sujet pour une app de rencontre).
  Migrations repartent de zéro : le dépôt était un boilerplate sans données de production.
- **Matching** : un `swipe` par couple (clé primaire), une décision n'est jamais écrasée (rejouer renvoie le même résultat).
  La création du match est protégée par un verrou consultatif par paire → deux likes simultanés donnent exactement un match.
  `matches(user_a < user_b)` garantit l'unicité de la paire.
- **Recommandation** : une seule requête SQL (`matching/repository.go`) applique les filtres durs mutuels puis classe par
  centres d'intérêt communs, distance, activité récente. C'est le point d'extension pour un futur algorithme.
- **Photos** : fichiers sur disque privé, nommés aléatoirement, jamais servis statiquement ; chaque lecture passe par
  `GET /photos/:id` qui vérifie blocage / visibilité / match. Images décodées puis ré-encodées en JPEG ≤ 1280 px
  (neutralise polyglottes et métadonnées). L'interface `Storage` permet un backend S3 plus tard.
- **Géolocalisation** : coordonnées arrondies à 2 décimales avant stockage, jamais renvoyées par l'API, distance
  approximative (paliers 1 km puis 5 km). La distance n'est utilisée que si les deux personnes ont partagé une position.
- **Temps réel** : WebSocket authentifié par un premier message (le jeton ne passe pas dans l'URL). Les événements ne
  contiennent que des identifiants ou le message ; REST reste la source de vérité (rattrapage à la reconnexion).
- **Chat** : « supprimer la conversation » est local (masque l'historique pour soi, réapparaît à un nouveau message).
  Bloquer ou annuler un match ferme la conversation pour les deux. L'accès renvoie toujours 404 aux non-participants.
- **Blocage** : bidirectionnel — profils, photos, découverte, conversation et notifications disparaissent des deux côtés.
- **Suppression de compte** : mot de passe requis ; `ON DELETE CASCADE` supprime profil, swipes, matchs, messages
  (y compris ceux envoyés aux autres : la conversation disparaît), notifications, signalements émis ; fichiers photo effacés.
  Les signalements **reçus** sont aussi supprimés avec le compte signalé (cascade) — à revoir si une rétention légale est requise.
- **Récupération de compte** : code aléatoire à usage unique (1 h), stocké haché, révoque toutes les sessions ; la route répond
  toujours 202 (pas d'énumération d'emails).
- **Modération** : les signalements sont stockés dans `reports` ; aucun back-office n'est fourni (consultation via SQL).
- **Web** : le projet cible mobile. Un export web fonctionne pour des tests visuels (les photos y sont chargées par `fetch` + blob),
  mais la session n'y est pas persistée.
