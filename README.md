# Lueur — une application de rencontre 100 % gratuite

Lueur est une application de rencontre complète : création de compte, profil et photos, découverte
par swipe, matchs, messagerie en temps réel, notifications, blocage, signalement et suppression de
compte. **Tout est gratuit** : aucun abonnement, aucun paywall, aucun « boost », aucun like limité,
aucune publicité.

L'interface est en français (textes centralisés dans `apps/mobile/src/i18n/fr.ts`).

## Fonctionnalités

| Domaine         | Ce qui est livré                                                                                                                                                  |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Compte          | Inscription, connexion, sessions avec refresh token à rotation, récupération par code e-mail, changement de mot de passe, suppression définitive du compte        |
| Profil          | Prénom, âge (18+ vérifié côté serveur), genre, bio, ville, centres d'intérêt, préférences (genres, tranche d'âge, distance), mise en pause du profil              |
| Photos          | 6 photos max, ordre et photo principale, remplacement, suppression, validation du contenu réel, recompression, accès protégé                                      |
| Découverte      | Deck de cartes (swipe ou boutons), galerie, profil complet, filtrage mutuel des préférences, profils déjà traités jamais re-proposés, ré-affichage des « passés » |
| Matching        | Like réciproque = match, jamais de doublon (contraintes SQL + verrou transactionnel), annulation de match                                                         |
| Chat            | Conversation créée au match, messages texte, temps réel (WebSocket), heure, lu/non lu, suppression locale d'une conversation                                      |
| Notifications   | Match et nouveau message, compteur de non lus, « tout marquer comme lu », push temps réel dans l'application                                                      |
| Sécurité        | Blocage (bidirectionnel, supprime le match), signalement avec contexte, limitation de débit, validation stricte, uploads sécurisés                                |
| Géolocalisation | Position approximative uniquement ; les autres ne voient qu'une distance arrondie par paliers de 5 km ; alternative sans GPS (choix d'une ville)                  |

## Stack technique

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256 (access 15 min) + refresh token opaque, bcrypt, SQL explicite (pas d'ORM), `coder/websocket`, `x/image` (recompression des photos).
- **Client** : Expo / React Native (iOS, Android **et web**), TypeScript strict, React Navigation, TanStack Query.
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Architecture

```text
.
├─ apps/mobile/                 # client Expo (iOS / Android / web)
│  ├─ App.tsx
│  ├─ src/api/                  # client HTTP unique (refresh auto), services, types, stockage des jetons
│  ├─ src/auth/                 # contexte d'authentification
│  ├─ src/realtime/             # connexion WebSocket + mise à jour du cache
│  ├─ src/navigation/           # pile principale, onglets, garde d'onboarding
│  ├─ src/screens/              # écrans
│  ├─ src/components/           # composants réutilisables (ui, SwipeDeck, PhotoGrid, …)
│  ├─ src/theme/ src/i18n/ src/lib/
│  └─ e2e/                      # tests Playwright sur le build web
├─ services/api/
│  ├─ cmd/api/                  # serveur (main.go) + câblage des routes (router.go)
│  ├─ cmd/seed/                 # données de démonstration (jamais en production)
│  └─ internal/
│     ├─ platform/              # config, db (+ migrations), middleware, httpx, logger, realtime, storage, mailer
│     └─ features/              # auth, profiles, photos, matching, chat, moderation, notifications
│        └─ <feature>/          # handler.go → service.go → repository.go (+ model.go, routes.go)
├─ docs/ARCHITECTURE.md
└─ docker-compose.yml
```

Règles : `handler → service → repository`, pas de SQL dans les handlers, un seul serveur HTTP (Gin),
le client n'accède au réseau que via `src/api`. Détails dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Démarrage rapide (Docker)

```bash
cp .env.example .env
# Renseignez JWT_SECRET (openssl rand -hex 32) et POSTGRES_PASSWORD dans .env
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# Données de démonstration (comptes @demo.invalid, jamais en production)
docker compose exec api ./seed
```

Compte de démo : `demo@demo.invalid` / `demo-password-123`. Trois profils de démonstration ont déjà
liké ce compte : un like en retour suffit pour obtenir un match. `./seed -reset` supprime les comptes de démo.

## Développement local (sans Docker pour l'API)

Prérequis : Go ≥ 1.24, Node ≥ 20, PostgreSQL ≥ 14.

```bash
# 1. Base de données
createdb app
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/app?sslmode=disable"
export JWT_SECRET="$(openssl rand -hex 32)"
export UPLOAD_DIR=./data/uploads
export LOG_RESET_CODES=true               # affiche les codes de récupération dans le journal

# 2. API (les migrations s'appliquent automatiquement au démarrage)
cd services/api
go run ./cmd/api                          # écoute sur :8080
go run ./cmd/seed                         # optionnel : données de démo

# 3. Client
cd ../../apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://localhost:8080 npx expo start --web     # ou --ios / --android
```

**Téléphone physique** : n'utilisez jamais `localhost`. Mettez `EXPO_PUBLIC_API_URL=http://<IP_LAN>:<PORT>`
(ex. `http://192.168.1.20:18080` avec Docker Compose) et vérifiez `curl http://<IP_LAN>:<PORT>/health` avant de lancer l'app.
Si un service local occupe déjà le port, changez `API_HOST_PORT` dans `.env`.

## Variables d'environnement

Voir [.env.example](.env.example) (aucun secret n'est versionné).

| Variable                                                             | Obligatoire | Rôle                                                                                |
| -------------------------------------------------------------------- | ----------- | ----------------------------------------------------------------------------------- |
| `DATABASE_URL`                                                       | oui         | Connexion PostgreSQL                                                                |
| `JWT_SECRET`                                                         | oui         | ≥ 32 caractères, aléatoire (les valeurs « placeholder » sont refusées au démarrage) |
| `PORT`                                                               | non (8080)  | Port d'écoute de l'API                                                              |
| `UPLOAD_DIR`                                                         | non         | Dossier des photos (à monter sur un volume persistant)                              |
| `ALLOWED_ORIGINS`                                                    | non         | Origines CORS/WebSocket autorisées (client web uniquement)                          |
| `TRUSTED_PROXIES`                                                    | non         | IP/CIDR du reverse proxy dont `X-Forwarded-For` est cru (sinon ignoré)              |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USERNAME` `SMTP_PASSWORD`              | non         | Envoi des e-mails de récupération de compte                                         |
| `MAIL_FROM`                                                          | si SMTP     | Adresse expéditrice                                                                 |
| `LOG_RESET_CODES`                                                    | non         | Développement : écrit les codes de récupération dans le journal si SMTP absent      |
| `APP_ENV`                                                            | non         | `production` : le seed refuse de s'exécuter                                         |
| `EXPO_PUBLIC_API_URL`                                                | client      | URL de l'API, injectée **au build** du client (vide en web = même origine)          |
| `POSTGRES_PASSWORD`, `POSTGRES_USER`, `POSTGRES_DB`, `API_HOST_PORT` | Compose     | Réglages de `docker-compose.yml`                                                    |

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et
appliquées au démarrage, **une seule fois chacune**, dans une transaction (table `schema_migrations`,
verrou consultatif entre instances). Pour en ajouter une : créez `00N_nom.sql` (ordre alphabétique).

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `photos`, `interests`,
`user_interests`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`,
`blocks`, `reports`, `notifications`. Clés étrangères en `ON DELETE CASCADE` (supprimer un compte efface
tout ce qui le concerne), contraintes `CHECK` (âge ≥ 18, genres, bornes), unicité (un match par paire,
une décision par couple), index sur tous les chemins de lecture (voir `AGENTS.md`).

> Cette version remplace l'ancien socle générique (organisations, facturation, posts). Une base créée avec
> l'ancien schéma doit être recréée : `docker compose down -v`.

## Tests et qualité

```bash
# API : format, analyse statique, tests unitaires + intégration (PostgreSQL requis), build
cd services/api
gofmt -l .                                # doit ne rien afficher
go vet ./...
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable" go test ./...
go build ./cmd/api ./cmd/seed

# Client : types, lint, format, tests unitaires, build web
cd apps/mobile
npm ci
npx tsc --noEmit && npm run lint && npm run format:check && npm test && npm run build:web

# E2E (inscription → profil → découverte → like → match → chat temps réel → blocage → suppression)
# Base vide requise ; le build web doit connaître l'URL de l'API.
createdb e2e
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
E2E_DATABASE_URL="postgres://postgres:postgres@localhost:5432/e2e?sslmode=disable" npm run test:e2e
# Chromium déjà installé ? PW_CHROMIUM_PATH=/chemin/vers/chrome  ;  sinon : npx playwright install chromium
```

Sans `TEST_DATABASE_URL`, les tests d'intégration sont ignorés (les tests unitaires tournent toujours).

## Déploiement

1. **API** : image Docker (`services/api/Dockerfile`, utilisateur non-root). Fournir `DATABASE_URL`, `JWT_SECRET`,
   `UPLOAD_DIR` (volume persistant), SMTP, `ALLOWED_ORIGINS`/`TRUSTED_PROXIES` selon votre infra. Placez-la derrière un
   proxy TLS (HSTS est envoyé quand `X-Forwarded-Proto: https`). Le proxy doit autoriser les WebSockets (`/ws`).
2. **Web** : `EXPO_PUBLIC_API_URL=https://api.exemple.fr npm run build:web` puis servir `apps/mobile/dist` (site statique).
3. **Mobile** : `EAS Build` / `expo prebuild` avec `EXPO_PUBLIC_API_URL` défini.
4. Une seule instance d'API est supportée telle quelle (hub WebSocket et limiteurs en mémoire) ; voir ci-dessous pour monter en charge.

## Principales décisions techniques

- **Garder la stack existante** (Go/Gin/pgx, Expo/TS) ; suppression de ce qui contredit le produit (Stripe, organisations, posts, commentaires, votes, ancien chat) et de Tamagui (déclaré mais inutilisé).
- **Match = deux likes**, matérialisé dans `matches(user_a < user_b)` ; un verrou consultatif par paire sérialise deux likes simultanés (testé). La première décision est définitive (`swipes` PK `(actor, target)`), « passer » ne revient que via « Ré-afficher les profils passés ».
- **Recommandation** : interface `Recommender` ; l'implémentation SQL filtre en mutuel (genre, âge, distance des deux côtés), exclut bloqués et déjà traités, puis classe par centres d'intérêt communs, proximité, activité récente et « m'a déjà liké ». Remplaçable sans toucher au reste.
- **Géolocalisation** : coordonnées arrondies à ~1 km au stockage, jamais renvoyées par l'API ; distance affichée par paliers de 5 km ; masquable (`showDistance`) ; sans GPS, choix d'une ville.
- **Photos** : type détecté sur le contenu (pas sur l'extension/MIME client), dimensions bornées avant décodage, ré-encodage JPEG 1280 px (retire EXIF/GPS et charges cachées), noms aléatoires, stockage hors webroot derrière une interface `Storage`, accès uniquement authentifié et soumis aux règles de blocage/visibilité.
- **Chat** : une conversation par match ; un non-participant reçoit 404 (pas d'oracle). Lu/non lu via `last_read_at` par participant (pas une ligne par message). « Supprimer » masque localement ; un nouveau message la fait réapparaître. Bloquer ou annuler le match supprime la conversation ; un signalement conserve un instantané des 20 derniers messages.
- **Temps réel** : WebSocket avec ticket à usage unique (30 s) pour ne jamais mettre le JWT dans une URL ; hub en mémoire derrière l'interface `Publisher`. Le client rafraîchit son cache à la reconnexion.
- **Sessions** : access token court + refresh token opaque haché, rotation à usage unique ; chaque requête vérifie que la session est active (déconnexion et suppression de compte effets immédiats).
- **Abus** : limiteurs par IP/utilisateur/e-mail ; `X-Forwarded-For` ignoré sauf `TRUSTED_PROXIES`.
- **Récupération de compte** : code à 6 chiffres haché, 15 min, 5 essais, réponse identique que le compte existe ou non ; sans SMTP, aucun e-mail n'est envoyé (sauf `LOG_RESET_CODES` en dev).
- **Modération** : les signalements sont stockés (`reports`, statut `open`) ; l'outil de revue (back-office) reste à brancher. Exemple : `SELECT * FROM reports WHERE status = 'open' ORDER BY created_at;`.
- **Âge** : 18 ans minimum imposé par la base (`CHECK`) et le service ; maximum 99.
- **Seed** : domaine réservé `@demo.invalid` (RFC 2606), refuse `APP_ENV=production`, supprimable en une commande.

## Limites connues et pistes

- Instance unique : pour plusieurs instances, remplacer le hub et les limiteurs par Redis/NATS (interfaces prêtes).
- Pas de notifications push OS (APNs/FCM) : les notifications sont stockées et poussées en direct dans l'app ouverte.
- Pas d'orientation EXIF appliquée côté serveur : les sélecteurs d'images mobiles fournissent des images déjà orientées.
- Pas de back-office de modération ni de vérification d'e-mail à l'inscription.

## Licence

Voir le dépôt. Contributions : [CONTRIBUTING.md](CONTRIBUTING.md) · [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
