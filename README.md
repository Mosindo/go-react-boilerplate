# Lien — application de rencontre gratuite

**Lien** est une application de rencontre complète et **100 % gratuite** : pas d'abonnement, pas de paywall, pas de boost payant, pas de limite de likes, pas de publicité.

Fonctionnalités : création de compte et récupération par e-mail, profil (prénom, âge, bio, centres d'intérêt, jusqu'à 6 photos), préférences de rencontre (genres, âge, distance), découverte par cartes (swipe, boutons ou flèches du clavier), match réciproque, messagerie temps réel (non-lu, accusés de lecture), notifications, blocage, signalement, pause du profil, suppression du compte et de toutes les données.

## Stack

| Couche | Technologies |
| --- | --- |
| API | Go 1.24, Gin, PostgreSQL 16 (`pgxpool`, SQL explicite, pas d'ORM), JWT HS256 + refresh tokens rotatifs, WebSocket (`gorilla/websocket`) |
| App | Expo 54 / React Native 0.81 / React Native Web, TypeScript strict, React Navigation, TanStack Query, `expo-image`, `expo-image-picker`, `expo-location` |
| Infra | Docker, Docker Compose, GitHub Actions |

## Architecture

```
services/api/
  cmd/api            serveur HTTP
  cmd/seed           comptes de démonstration (@demo.invalid), -purge pour les retirer
  internal/
    platform/        config, db (migrations versionnées), middleware (auth, CORS, rate limit),
                     realtime (hub WebSocket), storage, mailer, pagination, errors, logger
    features/        auth · profiles · photos · discovery (découverte, likes, matchs)
                     · chat · notifications · safety (blocages, signalements)
                     chacune : handler → service → repository (+ model, routes)
    server/          assemblage de l'application (utilisé par main et par les tests)
    testutil/        schéma Postgres isolé par test
apps/mobile/
  src/api            client HTTP (refresh automatique), types, endpoints
  src/hooks          auth, données (React Query), temps réel, deck de swipe
  src/screens        écrans · src/components : deck, cartes, sélecteurs, photos
  src/shared         design system (thème clair/sombre) et mise en page
  e2e/web            tests navigateur Playwright · e2e/maestro : parcours natif
```

Voir aussi [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Installation et lancement local

Prérequis : Go 1.24+, Node 22+, Docker (ou un PostgreSQL 14+).

```bash
cp .env.example .env            # puis renseigner JWT_SECRET : openssl rand -hex 32
docker compose up --build -d    # PostgreSQL + API sur http://localhost:18080
curl http://localhost:18080/health

# comptes de démonstration (facultatif)
docker compose exec api ./seed -yes       # mot de passe : Demo-Password-1
cd apps/mobile && npm ci
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run web      # navigateur
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npm start         # téléphone (Expo Go) : jamais localhost
```

Sans Docker : démarrer PostgreSQL, puis `cd services/api && DATABASE_URL=… JWT_SECRET=… go run ./cmd/api`. Les migrations s'appliquent automatiquement au démarrage.

> Téléphone physique : utilisez l'IP LAN de la machine. Vérifiez qu'aucun service local ne détourne le port (`docker compose ps`, `curl http://<IP_LAN>:18080/health`). En cas de conflit, changez le port hôte dans `docker-compose.yml` et `EXPO_PUBLIC_API_URL`.

## Variables d'environnement

Voir [`.env.example`](.env.example) (aucun secret n'est versionné).

| Variable | Rôle |
| --- | --- |
| `JWT_SECRET` (obligatoire) | Secret HS256, 32 caractères minimum, non factice |
| `DATABASE_URL` (obligatoire) | Chaîne de connexion PostgreSQL |
| `PORT` | Port de l'API (défaut `8080`) |
| `UPLOAD_DIR` | Stockage des photos (défaut `./data/uploads`, volume Docker `/data/uploads`) |
| `ALLOWED_ORIGINS` | Origines navigateur autorisées (CORS + WebSocket) |
| `TRUSTED_PROXIES` | Proxys dont `X-Forwarded-For` est cru (vide = aucun) |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | E-mails de récupération. Sans `SMTP_HOST`, le code est écrit dans les logs (développement uniquement) |
| `APP_ENV=production` | Le seed refuse de s'exécuter |
| `TEST_DATABASE_URL` | Base utilisée par `go test` |
| `EXPO_PUBLIC_API_URL` | URL publique de l'API pour l'app (non secrète) |

## Base de données et migrations

Les fichiers `services/api/internal/platform/db/schema/*.sql` sont embarqués dans le binaire et appliqués dans l'ordre, une seule fois chacun (table `schema_migrations`, verrou consultatif, une transaction par fichier). Pour ajouter une évolution : créer `005_xxx.sql`, ne jamais modifier un fichier déjà appliqué.

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Clés étrangères en `ON DELETE CASCADE` (supprimer un utilisateur efface tout ce qui le concerne ; ses signalements reçus sont conservés sans lien), contraintes `CHECK`/`UNIQUE` (paire de match ordonnée, un swipe par paire, un signalement ouvert par paire) et index sur les chemins chauds.

## Tests, qualité, build

```bash
# API (PostgreSQL requis : chaque test crée puis supprime son propre schéma)
cd services/api
export TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable
gofmt -l . && go vet ./cmd/... ./internal/server/... ./internal/platform/... ./internal/features/{auth,profiles,photos,discovery,chat,notifications,safety}/...
go test ./cmd/... ./internal/server/... ./internal/platform/... ./internal/features/{auth,profiles,photos,discovery,chat,notifications,safety}/...
go build ./cmd/api ./cmd/seed

# App
cd apps/mobile
npm ci && npx tsc --noEmit && npm run lint && npm run format:check && npm test

# Parcours complet dans un navigateur (API lancée avec ALLOWED_ORIGINS=http://localhost:8081, base vide)
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run web:export
npm run e2e:smoke            # smoke API : inscription → profil → photo → like → match → message
npm run e2e:web              # Playwright : deux utilisateurs, match, chat temps réel, accusé de lecture
```

Les tests d'intégration Go couvrent authentification (rotation des refresh tokens, récupération, verrouillage), validation des profils, photos (contenu réel, EXIF, limites), filtres de découverte et classement, règles de match (idempotence, likes simultanés), chat (permissions, pagination, non-lu), WebSocket, blocage, signalement et suppression de compte. Le parcours natif Maestro (`e2e/maestro`) demande un émulateur et n'est pas exécuté en CI.

## Déploiement

1. Construire l'image : `docker build -t lien-api services/api` (binaires `api` et `seed`, utilisateur non-root).
2. Fournir `DATABASE_URL`, `JWT_SECRET` (secret fort), `ALLOWED_ORIGINS` si une app web est servie, `SMTP_*`, et un **volume persistant** sur `/data/uploads`.
3. Placer l'API derrière un proxy TLS ; renseigner `TRUSTED_PROXIES` avec son IP pour que le rate limiting voie l'IP client réelle.
4. Application : `eas build` (mobile) ou `npm run web:export` (site statique) avec `EXPO_PUBLIC_API_URL` pointant vers l'API publique.

## Décisions techniques

- **Gratuit par conception** : aucun modèle de paiement ; Stripe, abonnements et organisations du boilerplate d'origine ont été retirés du chemin d'exécution.
- **Compatibilité mutuelle** : un profil n'est montré que si chacun correspond aux critères (genre, âge, distance) de l'autre. Un swipe est revalidé côté serveur : forger un identifiant ne contourne pas les préférences.
- **Matching** : verrou consultatif par paire dans la transaction → deux likes simultanés produisent exactement un match. Pass et like sont définitifs et ne réapparaissent jamais. Classement : intérêts communs, personnes qui vous ont déjà liké, activité récente (une seule expression SQL à faire évoluer : `discovery/repository.go`).
- **Confidentialité de la position** : les coordonnées sont arrondies à ~1 km avant stockage, jamais renvoyées ; seules des distances par paliers de 5 km sont affichées, l'affichage est désactivable, la position est facultative et ne se demande que sur action explicite.
- **Photos** : validées par leur contenu (JPEG/PNG), taille et dimensions bornées, EXIF appliqué puis supprimé, ré-encodées en JPEG (1280 px + miniature 360 px), noms générés côté serveur, servies uniquement à des utilisateurs authentifiés autorisés (non bloqués, profil visible ou match), cache privé immuable.
- **Temps réel** : WebSocket authentifié par ticket à usage unique de 30 s (le JWT ne transite jamais dans une URL). Hub en mémoire : pour plusieurs instances, remplacer `Publish` par un bus (ex. LISTEN/NOTIFY) sans toucher aux appelants. L'app resynchronise les listes à chaque reconnexion.
- **Sécurité** : JWT HS256 strict avec expiration (15 min) + refresh tokens hachés et rotatifs ; bcrypt ; rate limiting (par IP, par compte, par utilisateur) en mémoire ; limites de corps de requête ; en-têtes de sécurité ; messages d'erreur uniformes (pas d'énumération de comptes à la connexion/récupération) ; accès aux conversations d'autrui = 404.
- **Vérification de l'e-mail** : non incluse (choix de simplicité) ; la récupération de compte prouve néanmoins la possession de l'adresse.
- **Modération** : les signalements sont stockés (`reports`, statut `open`) ; leur traitement est manuel (SQL) et `users.status = 'suspended'` bloque la connexion. Aucune suspension automatique, pour éviter les signalements abusifs coordonnés.
- **Notifications** : in-app et temps réel (match). Pas de notifications push (nécessite FCM/APNs).

## Limitations connues

- Rate limiting et hub temps réel en mémoire : mono-instance.
- Stockage de photos sur disque local (interface `storage.Storage` prête pour S3).
- Pas de console d'administration de modération.
