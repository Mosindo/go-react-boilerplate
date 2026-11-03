# Braise

Braise est une application de rencontre **100 % gratuite** : pas d'abonnement, pas de paywall, pas de boost payant, pas de limite de likes, pas de publicité. Deux personnes peuvent créer un profil, se découvrir, se liker, matcher et discuter en temps réel.

- **API** : Go, Gin, PostgreSQL (`pgx`), sans ORM, JWT HS256 + refresh tokens rotatifs, WebSocket.
- **Client** : Expo / React Native + TypeScript (iOS, Android, web), React Navigation, TanStack Query.
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Fonctionnalités

| Domaine | Détail |
| --- | --- |
| Compte | inscription, connexion, refresh rotatif, récupération de mot de passe, suppression de compte (efface données et fichiers) |
| Profil | prénom, date de naissance (18+ vérifié côté serveur), genre, bio, ville, centres d'intérêt, préférences (genres, âge, distance) |
| Photos | jusqu'à 6, ordre, photo principale, remplacement, suppression ; JPEG/PNG/WebP ≤ 8 Mo, ré-encodées en JPEG ≤ 1280 px |
| Découverte | paquet de profils filtrés (préférences mutuelles, âge, distance, blocs, déjà traités), swipe ou boutons, fiche complète |
| Matching | like / pass définitifs, match réciproque atomique et unique, « Ils vous ont liké » gratuit, défaire un match |
| Chat | conversation créée au match, temps réel (WebSocket), heure, lu / non lu, suppression locale, pagination |
| Notifications | notification de match persistée + événements temps réel, compteur de non lues |
| Sécurité | blocage (bilatéral), signalement (6 motifs), rate limiting, uploads protégés |
| Vie privée | distance approximative par tranches de 5 km, position arrondie à ~1 km, masquage possible, profil non découvrable |

## Architecture

```text
apps/mobile/            client Expo (src/api, auth, realtime, features, screens, navigation, ui)
services/api/
  cmd/api/              serveur HTTP unique (Gin) + tests d'intégration
  cmd/seed/             données de démonstration (refuse APP_ENV=production)
  internal/platform/    config, db (+ migrations SQL embarquées), middleware, ratelimit, realtime, httpx, errors, logger
  internal/features/    auth, profiles, photos, matching, chat, notifications, safety
infra/docker-compose.yml
docs/ARCHITECTURE.md    détails et décisions
```

Chaque feature suit `handler → service → repository` (+ `model.go`, `routes.go`). Pas de SQL dans les handlers. Détails dans [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Démarrage rapide (Docker)

```bash
cp .env.example .env
# renseigner JWT_SECRET :
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=$(openssl rand -hex 32)/" .env
docker compose -f infra/docker-compose.yml --env-file .env up --build -d
curl http://localhost:18080/health            # {"status":"ok"}
# données de démo (facultatif) :
docker compose -f infra/docker-compose.yml --env-file .env exec api ./seed
```

Les migrations s'appliquent automatiquement au démarrage de l'API (table `schema_migrations`, verrou consultatif : plusieurs instances peuvent démarrer en même temps).

### Sans Docker

Prérequis : Go ≥ 1.24, PostgreSQL ≥ 14, Node ≥ 22.

```bash
createdb app    # ou utiliser le Postgres de Docker : docker compose -f infra/docker-compose.yml up -d postgres
cd services/api
export DATABASE_URL='postgresql://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export PORT=18080
go run ./cmd/seed     # facultatif : 12 profils de démonstration
go run ./cmd/api
```

## Lancer l'application mobile

```bash
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start    # ou: npm run start:phone
```

- **Téléphone physique** : n'utilisez jamais `localhost`. Utilisez l'IP LAN de la machine (`npm run start:phone` la détecte).
- Émulateur Android : `http://10.0.2.2:18080` (valeur par défaut). Simulateur iOS / web : `http://localhost:18080`.
- Vérifiez qu'aucun autre service n'occupe le port : `docker compose ps`, `curl http://<IP_LAN>:18080/health`. En cas de conflit, changez le port hôte (`18080:8080`) dans `infra/docker-compose.yml`, `EXPO_PUBLIC_API_URL` et `src/config.ts`.
- Web (CORS) : ajoutez l'origine du client à `ALLOWED_ORIGINS`.

Comptes de démonstration (après `seed`) : `camille@demo.invalid`, `hugo@demo.invalid`, … mot de passe `DemoPassword123`. Domaine `demo.invalid` réservé (RFC 2606) : aucun risque de collision avec de vrais comptes. `go run ./cmd/seed -purge` les supprime.

## Variables d'environnement

Voir [`.env.example`](.env.example).

| Variable | Obligatoire | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | ≥ 32 caractères aléatoires (`openssl rand -hex 32`). Les valeurs « placeholder » sont refusées. |
| `DATABASE_URL` | oui | DSN PostgreSQL. |
| `PORT` | non (8080) | Port d'écoute de l'API. |
| `APP_ENV` | non | `production` désactive le seed et signale l'absence de mailer réel. |
| `UPLOAD_DIR` | non | Dossier des photos (monter un volume persistant). |
| `ALLOWED_ORIGINS` | non | Origines navigateur autorisées (CORS + WebSocket). |
| `TRUSTED_PROXIES` | non | Proxys dont `X-Forwarded-For` est cru (rate limiting par IP). Vide = personne. |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API vue par le client. Ne contient aucun secret. |

## Tests et qualité

```bash
# API (les tests d'intégration utilisent DATABASE_URL_TEST ou DATABASE_URL, et sont ignorés sans base)
cd services/api
gofmt -l .                       # doit ne rien afficher
go vet ./...
DATABASE_URL=postgresql://app:app@localhost:5432/app?sslmode=disable go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test
npm run build:web                # export web (EXPO_PUBLIC_API_URL à définir)
```

**E2E navigateur** (inscription → onboarding → découverte → like → match → conversation, avec réponse temps réel) : `apps/mobile/e2e/web-flow.mjs`.

```bash
# API lancée avec ALLOWED_ORIGINS=http://localhost:8099 et le seed appliqué
cd apps/mobile && EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
(cd dist && python3 -m http.server 8099 &)
npm i --no-save playwright && npx playwright install chromium   # une fois
node e2e/web-flow.mjs
```

## API (résumé)

Toutes les routes, sauf `/health`, `/auth/*` et `/ws`, exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "...", "code": "..."}`. Listes paginées par curseur (`limit`, `before`).

| Route | Rôle |
| --- | --- |
| `POST /auth/register` `login` `refresh` `logout` `forgot-password` `reset-password` | session et récupération |
| `GET /me`, `DELETE /me` | compte (suppression : mot de passe requis) |
| `GET/PUT /me/profile`, `PUT /me/preferences`, `PUT /me/privacy`, `PUT/DELETE /me/location`, `GET /interests`, `GET /profiles/:id` | profil |
| `POST /me/photos`, `POST /me/photos/:id/replace`, `DELETE /me/photos/:id`, `PUT /me/photos/order`, `GET /photos/:id` | photos |
| `GET /discover`, `POST /swipes`, `GET /likes/received`, `GET /matches`, `DELETE /matches/:id` | découverte et matching |
| `GET /conversations`, `GET/POST /conversations/:id/messages`, `POST /conversations/:id/read`, `DELETE /conversations/:id` | chat |
| `GET /ws` | WebSocket : premier message `{"type":"auth","token":"…"}` ; événements `message`, `read`, `match`, `unmatch` |
| `GET /notifications`, `POST /notifications/read`, `POST /notifications/:id/read` | notifications |
| `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` | sécurité |

## Décisions techniques

- **Base existante conservée** : Go/Gin/pgx, sessions à refresh rotatif, middleware, logger. Supprimés car contraires au produit ou inutiles : Stripe/abonnements, organisations, posts, commentaires, anciens modules génériques, Tamagui, scripts PowerShell.
- **Migrations** : réinitialisées et suivies (`schema_migrations`) ; l'ancien runner rejouait tous les fichiers à chaque démarrage.
- **Matching** : verrou consultatif par paire dans une transaction → un seul match même si deux likes arrivent simultanément. Les likes/pass sont définitifs (le profil ne revient jamais) ; seul « défaire le match » est possible, et la paire ne réapparaît pas.
- **Recommandation** : filtres durs en SQL + score remplaçable (intérêts communs, personnes qui vous ont déjà liké, proximité) dans `matching/repository.go`.
- **Distance** : calcul haversine en SQL sans PostGIS. La limite de distance ne s'applique que si les deux personnes ont partagé une position (étape facultative, personne n'est masqué pour l'avoir ignorée). Pour de gros volumes, ajouter PostGIS ou un pré-filtre par boîte englobante.
- **Photos privées** : servies par `GET /photos/:id` après authentification, refus si blocage ou profil non visible. Le client envoie le jeton dans l'en-tête (blob + URL objet sur le web).
- **Temps réel** : hub WebSocket en mémoire (une instance). Pour plusieurs instances, ajouter un pont pub/sub (LISTEN/NOTIFY, Redis). Le client se reconnecte avec backoff et recharge ses listes à chaque reconnexion.
- **Notifications** : en base + temps réel. Pas de push APNs/FCM (nécessite des comptes développeur).
- **E-mail de récupération** : interface `Mailer` ; l'implémentation par défaut écrit le code dans les journaux (développement). Brancher SMTP/API avant la production (l'API le signale au démarrage en `APP_ENV=production`).
- **Modération** : les signalements sont stockés (`reports`, statut `open`) ; il n'y a pas d'interface d'administration.
- **Suppression de compte** : suppression en cascade (profil, photos, swipes, matchs, messages, signalements) + effacement des fichiers.
- **Rate limiting** : en mémoire, par IP (auth) et par utilisateur (swipes, messages, uploads, signalements).
- **Jetons d'accès** : 15 min ; un jeton volé reste valable jusqu'à expiration même après déconnexion (compromis assumé, pas de vérification de session à chaque requête).
