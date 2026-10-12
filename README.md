# Mellow — application de rencontre 100 % gratuite

Mellow est une application de rencontre mobile (iOS / Android / web) : créer un profil, découvrir des personnes proches, liker ou passer, matcher quand l'intérêt est réciproque, puis discuter en temps réel.

**Gratuit pour de vrai** : aucun abonnement, aucun paywall, aucun « boost », aucun like limité, aucune publicité. Les seules limites techniques (rate limiting) servent à bloquer les robots, pas les humains.

Le dépôt est un monorepo : une API Go + PostgreSQL et un client React Native (Expo). Le nom affiché est configurable (`EXPO_PUBLIC_APP_NAME`).

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | inscription, connexion, sessions à refresh token rotatif, récupération de mot de passe, suppression définitive du compte (avec mot de passe) |
| Profil | prénom, âge (calculé depuis la date de naissance, **18 ans minimum**, date verrouillée après création), genre, bio, métier, ville, centres d'intérêt, préférences (genres recherchés, tranche d'âge, distance) |
| Photos | jusqu'à 6 photos : ajout, remplacement, suppression, réordonnancement, photo principale ; validation du contenu réel, recompression JPEG, miniatures, EXIF/GPS supprimés, URLs signées et expirantes |
| Découverte | pile de profils filtrée par préférences **mutuelles**, âge **mutuel**, distance, blocages ; swipe ou boutons ; un profil déjà traité ne revient jamais |
| Matching | like / pass idempotents, match atomique (même en cas de likes simultanés), unmatch |
| Chat | conversation ouverte après un match, messages texte, temps réel (WebSocket), heure, lu / non lu, liste des conversations, effacement local, pagination |
| Notifications | nouveaux matchs et messages, badge, « tout lire » |
| Sécurité | blocage, signalement (avec capture des derniers messages comme preuve), masquage du profil, distance masquable |
| Localisation | demandée seulement sur action de l'utilisateur ; stockée **arrondie à ~1 km** ; seule une distance approximative (« à moins de 25 km ») est affichée |
| UX | thème clair/sombre, états de chargement / vide / erreur, retours par toasts, accessibilité (rôles, libellés, zones tactiles) |

## Stack

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256, bcrypt, gorilla/websocket, `golang.org/x/image` — pas d'ORM, SQL explicite.
- **Mobile** : Expo SDK 54, React Native 0.81, TypeScript strict, React Navigation, TanStack Query, expo-image, expo-image-picker, expo-location.
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Architecture

```text
services/api
├─ cmd/api            serveur HTTP unique (Gin) + graceful shutdown
├─ cmd/seed           données de démonstration (refuse APP_ENV=production)
└─ internal
   ├─ platform        config, db (migrations suivies + verrou), middleware (auth, rate limit, CORS…),
   │                  realtime (hub WebSocket), mediaurl (URLs signées), httpx, logger
   └─ features        auth · profiles · photos · discovery · matching · messaging · safety · alerts
                      chaque feature : handler → service → repository (+ model, routes)
apps/mobile
├─ App.tsx            providers (thème, toasts, auth, temps réel)
└─ src                api/ (client + endpoints typés) · hooks/ · navigation/ · screens/ · components/ · ui/ · theme/ · lib/
```

Règles respectées : handlers fins, logique métier dans les services, SQL uniquement dans les repositories, aucun accès réseau direct depuis l'UI (tout passe par `src/api`).

### Décisions techniques principales

- **Un match *est* la conversation** : table `matches` (paire canonique `user_a < user_b`) + `match_members` (état lu / effacement par participant) + `chat_messages`. Un utilisateur ne peut lire que les messages d'un match dont il est membre ; un non-membre reçoit `404` (l'existence n'est pas révélée).
- **Match atomique** : un verrou consultatif PostgreSQL sur la paire sérialise deux likes simultanés → exactement un match (test dédié).
- **Distance** : calcul haversine en SQL sur des coordonnées déjà arrondies, restitué par paliers (1, 2, 5, 10, 25, 50, 100, 250, 500 km).
- **Photos privées** : jamais servies publiquement. Chaque URL est signée (HMAC dérivé de `JWT_SECRET`) et expire ; elle n'est fournie qu'à ceux qui peuvent voir le profil. Stockage derrière une interface `Storage` (disque local par défaut ; S3/GCS possibles).
- **Sessions vérifiées à chaque requête** : déconnexion, changement de mot de passe, désactivation ou suppression du compte invalident immédiatement les access tokens (15 min) existants.
- **Temps réel** : WebSocket authentifié par une première trame (pas de token dans l'URL). Hub en mémoire → une seule instance d'API ; l'interface `realtime.Publisher` permet de brancher Redis / LISTEN-NOTIFY pour monter en charge. Après reconnexion, le client resynchronise via REST.
- **Recommandation** : le classement est isolé dans le `ORDER BY` de `discovery/repository.go` (déjà : ceux qui vous ont liké d'abord, puis activité récente, proximité, mélange quotidien). Il se remplace sans toucher aux filtres.
- **Anti-abus** : rate limiting par IP (auth) et par utilisateur (swipes, messages, uploads, signalements). `TRUSTED_PROXIES` doit être renseigné derrière un reverse proxy, sinon l'IP cliente n'est pas fiable.
- **Mot de passe oublié** : jeton à usage unique de 1 h, stocké haché. L'envoi d'e-mail passe par l'interface `auth.Mailer` : fournissez une implémentation SMTP/API en production (`Service.WithMailer`). En développement, `MAIL_LOG_TOKENS=true` journalise le jeton.
- **Pas de vérification d'e-mail ni de téléphone** (choix de simplicité) : à ajouter via le `Mailer` avant un lancement public.
- **Modération** : pas d'interface d'administration. La file de signalements est la table `reports` ; voir ci-dessous.

## Installation

Prérequis : Go 1.24+, Node 20+, Docker (ou un PostgreSQL 14+).

```bash
git clone <repo> && cd <repo>
cp .env.example .env            # puis renseignez JWT_SECRET (openssl rand -hex 32)
```

### Lancer l'API avec Docker (recommandé)

```bash
export JWT_SECRET=$(openssl rand -hex 32)
docker compose up --build        # PostgreSQL + API sur http://localhost:18080
curl http://localhost:18080/health
```

Le port hôte **18080** évite les conflits avec d'autres services locaux. Si le port est pris, changez-le dans `docker-compose.yml`, `infra/docker-compose.yml`, `apps/mobile/src/api/client.ts` (valeur de repli), `apps/mobile/scripts/start-phone.js` et ce README.

### Lancer l'API sans Docker

```bash
cd services/api
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
go run ./cmd/api                 # applique les migrations puis écoute sur :8080
```

### Variables d'environnement

| Variable | Obligatoire | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | secret ≥ 32 caractères (aussi utilisé pour signer les URLs de photos). Les valeurs « placeholder » sont refusées |
| `DATABASE_URL` | oui | chaîne de connexion PostgreSQL |
| `PORT` | non (8080) | port d'écoute |
| `UPLOAD_DIR` | non (`./data/uploads`) | dossier des photos — à monter sur un volume persistant |
| `ALLOWED_ORIGINS` | non | origines web autorisées (CORS + WebSocket), séparées par des virgules |
| `TRUSTED_PROXIES` | non | IP/CIDR des reverse proxies de confiance |
| `MAIL_LOG_TOKENS` | non | **dev uniquement** : journalise les jetons de réinitialisation |
| `APP_ENV` | non | `production` désactive le seed |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API vue par l'appareil |
| `EXPO_PUBLIC_APP_NAME` | mobile | nom affiché |

Aucun secret ne doit aller dans le frontend : les variables `EXPO_PUBLIC_*` sont publiques.

### Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`internal/platform/db/migrations`) et appliquées au démarrage, **une seule fois chacune** (table `schema_migrations`), sous verrou consultatif (plusieurs instances peuvent démarrer ensemble). Schéma principal : `users`, `sessions`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `match_members`, `chat_messages`, `blocks`, `reports`, `user_notifications`, `password_reset_tokens` — avec clés étrangères `ON DELETE CASCADE` (supprimer un utilisateur efface toutes ses données), contraintes `CHECK` / `UNIQUE` et index sur les chemins chauds.

### Données de démonstration

```bash
cd services/api
DATABASE_URL=… JWT_SECRET=… UPLOAD_DIR=./data/uploads go run ./cmd/seed
```

Crée 18 comptes `demo.NN@demo.invalid` (domaine réservé RFC 2606 : jamais de vrai utilisateur), mot de passe `DemoPass123!` (ou `DEMO_PASSWORD`), avec des photos abstraites générées. Connectez-vous avec `demo.01@demo.invalid` : plusieurs comptes l'ont déjà liké, un like en retour donne un match immédiat. Idempotent ; suppression : `DELETE FROM users WHERE email LIKE '%@demo.invalid';` (les fichiers du dossier d'upload sont à nettoyer à part). Le seed refuse de tourner avec `APP_ENV=production`.

### Lancer l'application mobile

```bash
cd apps/mobile
npm ci
# téléphone physique : JAMAIS localhost, utilisez l'IP LAN de la machine qui héberge l'API
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start
# ou : npm run start:phone   (détecte l'IP LAN automatiquement)
```

Avant de déclarer l'app « prête » sur un téléphone, vérifiez qu'aucun autre service ne détourne le port : `docker compose ps`, `curl http://<IP_LAN>:18080/health`, puis faites le parcours réel inscription → profil → découverte → like → chat → notifications.

Aperçu web : `npx expo start --web` (ajoutez l'origine à `ALLOWED_ORIGINS`).

## Tests et qualité

```bash
node scripts/qa.js               # gofmt, go vet, go test, go build, tsc, eslint, prettier, jest
```

| Quoi | Commande |
| --- | --- |
| API (unitaires + intégration, nécessite PostgreSQL) | `cd services/api && DATABASE_URL=… go test ./...` |
| Mobile typecheck / lint / format / tests | `cd apps/mobile && npm run typecheck && npm run lint && npm run format:check && npm test` |
| Smoke API sur un serveur lancé | `SMOKE_API_URL=http://localhost:18080 npm run e2e:smoke` (dans `apps/mobile`) |
| E2E navigateur (Playwright) : inscription → profil → découverte → like → match → chat temps réel | `E2E_API_URL=http://localhost:18080 npm run e2e:web` (API lancée avec `ALLOWED_ORIGINS=http://localhost:19006`) |

Les tests d'intégration Go couvrent : authentification (rotation du refresh token, révocation immédiate, reset de mot de passe), validation des profils et verrouillage de l'âge, fuite de localisation, découverte (préférences mutuelles, distance), likes (doublons, idempotence), match (y compris concurrent), permissions du chat, WebSocket, photos (types dangereux, limites, ordre), blocage, signalement, suppression de compte (données **et** fichiers).

## Build et déploiement

```bash
cd services/api && go build -o api ./cmd/api          # binaire statique ; ou docker build services/api
```

Production : reverse proxy TLS devant l'API (renseignez `TRUSTED_PROXIES`), volume persistant pour `UPLOAD_DIR`, PostgreSQL géré avec sauvegardes, `JWT_SECRET` fort via un gestionnaire de secrets, un `Mailer` réel, `ALLOWED_ORIGINS` si vous publiez un client web. Application mobile : `eas build` (identifiants `app.mellow.dating` à adapter dans `apps/mobile/app.json`). Une seule instance d'API tant que le hub WebSocket est en mémoire et que les photos sont sur disque local.

### Traiter les signalements

```sql
SELECT r.id, r.created_at, r.reason, r.details, r.context, ru.email AS reported
FROM reports r JOIN users ru ON ru.id = r.reported_id
WHERE r.status = 'open' ORDER BY r.created_at;

UPDATE users SET disabled_at = NOW() WHERE id = '<uuid>';   -- suspend : connexion et sessions invalidées
UPDATE reports SET status = 'reviewed' WHERE id = '<uuid>';
```

## Limites connues

- Hub temps réel et stockage photo mono-instance (voir ci-dessus).
- Pas de notifications push natives (les notifications sont intégrées à l'app et poussées par WebSocket quand elle est ouverte).
- Pas de vérification d'e-mail, pas d'interface d'administration.
