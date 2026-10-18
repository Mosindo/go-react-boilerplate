# Tendre — application de rencontre 100 % gratuite

Tendre est une application de rencontre complète (mobile iOS / Android / web) : créer un profil, ajouter des photos,
découvrir des personnes proches, liker ou passer, matcher, discuter en temps réel, gérer sa confidentialité, bloquer,
signaler et supprimer son compte.

**Gratuite pour de vrai** : aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de
likes, aucune fonctionnalité verrouillée, aucune publicité obligatoire. Le code ne contient aucune intégration de
paiement.

> Le nom « Tendre » est un nom de travail : il tient en une constante (`APP_NAME` dans
> `apps/mobile/src/components/labels.ts`) et dans `apps/mobile/app.json`.

## Sommaire

1. [Stack](#stack)
2. [Architecture](#architecture)
3. [Démarrage rapide](#démarrage-rapide)
4. [Variables d'environnement](#variables-denvironnement)
5. [Base de données, migrations, seed](#base-de-données-migrations-seed)
6. [Tests, lint, build](#tests-lint-build)
7. [Application mobile](#application-mobile)
8. [API](#api)
9. [Sécurité et vie privée](#sécurité-et-vie-privée)
10. [Décisions techniques et produit](#décisions-techniques-et-produit)
11. [Déploiement](#déploiement)
12. [Modération](#modération)
13. [Limites connues](#limites-connues)

## Stack

| Couche    | Technologies                                                                                       |
| --------- | -------------------------------------------------------------------------------------------------- |
| API       | Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256 + refresh tokens rotatifs, bcrypt, WebSocket     |
| Mobile    | Expo 54 / React Native 0.81, TypeScript strict, React Navigation, TanStack Query, `react-native-web` |
| Infra     | Docker, Docker Compose, GitHub Actions                                                              |
| Qualité   | `go test` (intégration sur vraie base), Playwright (parcours complet en navigateur), ESLint, Prettier |

Pas d'ORM, pas de « magie » : du SQL explicite dans les repositories.

## Architecture

```text
.
├─ apps/mobile/                 # application Expo (iOS, Android, web)
│  ├─ App.tsx
│  ├─ e2e/web/                  # parcours Playwright (inscription → match → chat → suppression)
│  ├─ e2e/maestro/              # flux appareil (Maestro)
│  ├─ scripts/                  # e2e-web.js, e2e-smoke.js (API), fixtures
│  └─ src/
│     ├─ api/                   # client HTTP unique (refresh auto du token) + modules par domaine
│     ├─ components/            # carte profil, deck de swipe, chat, photos, préférences…
│     ├─ hooks/                 # auth, requêtes, temps réel (WebSocket), bus du deck
│     ├─ navigation/            # navigateurs racine / auth / principal
│     ├─ screens/               # écrans
│     └─ shared/                # design system (ui, layout, feedback)
├─ services/api/
│  ├─ cmd/api                   # serveur HTTP (un seul)
│  ├─ cmd/seed                  # données de démonstration (comptes @demo.invalid)
│  └─ internal/
│     ├─ app/                   # assemblage des features sur un moteur Gin
│     ├─ features/              # handler → service → repository, par domaine
│     │  ├─ auth/ profiles/ photos/ matching/ chat/ safety/ notifications/
│     └─ platform/              # config, db (+migrations), middleware, realtime, storage, mailer, logger…
├─ docker-compose.yml           # postgres + api
└─ docs/ARCHITECTURE.md
```

Règle de dépendance : `handler → service → repository`. Aucun SQL dans les handlers, aucune logique métier dans les
handlers. Les préoccupations transverses vivent dans `platform`, les préoccupations métier dans `features`.

## Démarrage rapide

Prérequis : Docker, Node 20+, (Go 1.24 pour lancer l'API hors Docker).

```bash
# 1. API + base de données
export JWT_SECRET="$(openssl rand -base64 48)"
docker compose up --build -d
curl http://localhost:18080/health            # => {"status":"ok"...}

# 2. Données de démonstration (facultatif)
docker compose exec -e SEED_CONFIRM=yes api ./seed
#    compte : demo+01@demo.invalid / demo-password-123

# 3. Application
cd apps/mobile
npm ci
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start          # téléphone : scanner le QR code
EXPO_PUBLIC_API_URL=http://localhost:18080 npx expo start --web   # navigateur
```

Sur un téléphone physique, **n'utilisez jamais `localhost`** : renseignez l'IP de votre machine sur le réseau local.
Vérifiez qu'aucun autre service n'occupe le port : `docker compose ps`, puis
`curl http://<IP_LAN>:18080/health`. En cas de conflit, changez le port hôte (`18080:8080`) dans
`docker-compose.yml`, `infra/docker-compose.yml` et `EXPO_PUBLIC_API_URL`.

Sans Docker pour l'API :

```bash
docker compose up -d postgres
cd services/api
cp ../../.env.example .env   # puis exportez-le, ou : set -a; . ./.env; set +a
go run ./cmd/api
```

## Variables d'environnement

Voir [`.env.example`](.env.example). Aucun secret n'est versionné ni embarqué dans l'application.

| Variable                          | Rôle                                                                                     | Défaut             |
| --------------------------------- | ---------------------------------------------------------------------------------------- | ------------------ |
| `DATABASE_URL` (obligatoire)      | Connexion PostgreSQL                                                                     | —                  |
| `JWT_SECRET` (obligatoire)        | Secret HS256, ≥ 32 caractères, refusé s'il ressemble à un exemple. Sert aussi à signer les URLs de photos, les tickets WebSocket et à chiffrer (HMAC) les codes de récupération | —   |
| `PORT`                            | Port HTTP                                                                                | `8080`             |
| `APP_ENV`                         | `production` désactive les comportements de dev (log des e-mails, seed)                  | `development`      |
| `UPLOAD_DIR`                      | Dossier des photos (volume persistant en production)                                     | `./data/uploads`   |
| `ALLOWED_ORIGINS`                 | Origines navigateur autorisées (CORS), utile pour la version web uniquement              | vide               |
| `TRUSTED_PROXIES`                 | Proxys dont `X-Forwarded-For` est cru (IP client pour le rate limiting)                  | vide (aucun)       |
| `RATE_LIMIT_ENABLED`              | Active le rate limiting                                                                  | `true`             |
| `SMTP_HOST/PORT/USER/PASSWORD/FROM` | Envoi des codes de récupération de mot de passe                                        | vide               |
| `EXPO_PUBLIC_API_URL` (mobile)    | URL de l'API. **Publique** : ne jamais y mettre de secret                                | `localhost:18080` en dev |

Sans SMTP, en développement, le code de récupération est écrit dans les logs de l'API ; en production il n'est
**jamais** loggé (le mail est simplement abandonné avec un avertissement).

## Base de données, migrations, seed

- Migrations SQL embarquées dans le binaire (`services/api/internal/platform/db/migrations`), appliquées au démarrage,
  **une fois chacune**, dans une transaction, avec verrou consultatif (plusieurs réplicas peuvent démarrer ensemble).
  Le suivi est dans la table `schema_migrations`.
- `014_dating_schema.sql` crée le modèle de rencontre et supprime les anciennes tables génériques du boilerplate.
- Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`,
  `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`.
- Contraintes : clés étrangères `ON DELETE CASCADE` (la suppression d'un compte efface tout), unicité
  (`swipes (swiper, target)`, `matches (user_a, user_b)` avec `user_a < user_b`, `photos (user, position)`), `CHECK` sur
  les énumérations, longueurs et bornes, e-mails en minuscules.
- Index : `users.email`, `profiles (latitude, longitude)`, `swipes (target, swiper) WHERE like`,
  `messages (conversation_id, created_at DESC, id DESC)`, `notifications (user_id, created_at DESC)`,
  `conversation_participants (user_id, conversation_id)`, `blocks (blocked_id, blocker_id)`, etc.
- **Seed** (`cmd/seed`) : 12 comptes fictifs `demo+NN@demo.invalid` (domaine réservé qui ne peut appartenir à personne),
  portraits abstraits générés (jamais de vrais visages). Refuse de tourner avec `APP_ENV=production` ou sans
  `SEED_CONFIRM=yes`. `-reset` supprime uniquement ces comptes. Position de base : `SEED_LAT` / `SEED_LON` /
  `SEED_CITY` (Paris par défaut).

## Tests, lint, build

```bash
# API (les tests d'intégration créent une base jetable par test ; pointez vers n'importe quel PostgreSQL)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
gofmt -l .            # doit être vide
go vet ./...
go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci
npm run typecheck && npm run lint && npm run format:check
npm run e2e:web       # parcours complet en navigateur (voir ci-dessous)
```

- **Tests d'intégration API** (`services/api/cmd/api/main_integration_test.go`) : authentification (rotation et rejeu du
  refresh token, révocation de session, récupération de mot de passe, verrouillage après 5 essais), profils (âge ≥ 18,
  validation, arrondi des coordonnées), photos (fichiers dangereux, taille, limite, propriété, URLs signées), découverte
  (règles de préférences mutuelles, blocages, absence de fuites), likes simultanés (un seul match), chat (permissions,
  lu/non lu, masquage, pagination), blocage, signalement, suppression de compte (données et fichiers), WebSocket.
- **Tests unitaires** : traitement d'image (décompression bomb, métadonnées EXIF supprimées), signatures, ranker,
  validation de profil, rate limiter, stockage (path traversal), hub temps réel.
- **E2E navigateur** (`npm run e2e:web`) : démarre l'API réelle, exporte le build web, puis rejoue avec Playwright :
  inscription → onboarding complet (upload de photo réel) → découverte → like → match → conversation temps réel entre
  deux navigateurs → lu → blocage → suppression de compte. Prérequis : `E2E_DATABASE_URL` vers une base jetable, et
  `npx playwright install chromium` la première fois. Captures dans `apps/mobile/.e2e/screens`.
- **Smoke API** (`node apps/mobile/scripts/e2e-smoke.js`, `MOBILE_E2E_API_URL=...`) contre une stack déjà lancée.
- **Maestro** (`e2e/maestro/critical-flow.yaml`) : flux appareil, non exécuté en CI.

## Application mobile

Écrans : connexion / création de compte (case « 18 ans ou plus »), mot de passe oublié (code par e-mail), onboarding en
5 étapes (identité, préférences, position, bio + centres d'intérêt, photos), **Découvrir** (deck à swipe + boutons,
carte avec galerie de photos, détail du profil), **Messages** (nouveaux matchs, conversations, non-lus), conversation
(temps réel, heure, accusé de lecture, menu : profil / masquer / supprimer le match / bloquer / signaler),
**Activité** (notifications), **Profil** (aperçu public, édition, photos réordonnables, préférences), **Paramètres**
(visibilité, distance, mot de passe, personnes bloquées, déconnexion, suppression du compte).

Identité visuelle : palette chaude (framboise `#b23a5b`, abricot, crème), typographie système, coins arrondis, ombres
douces, animations sobres. États de chargement, d'erreur, de vide et retours (toasts) partout ; libellés d'accessibilité
et alternatives sans geste (boutons ✕ / ♥ / i en plus du swipe).

Toutes les requêtes passent par `src/api/client.ts` (timeout, refresh automatique et mono-requête du token, erreurs
normalisées) : aucun écran n'appelle `fetch` directement.

## API

Toutes les routes (sauf `/health`, `/auth/*` et `/media/*`) exigent `Authorization: Bearer <accessToken>`. Erreurs :
`{"error": "...", "code"?: "..."}`. Listes paginées par curseur (`?limit=&cursor=`).

| Domaine        | Routes                                                                                                                       |
| -------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Santé          | `GET /health`                                                                                                                |
| Auth           | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/password/forgot`, `/auth/password/reset`       |
| Compte         | `GET /me`, `POST /me/password`, `DELETE /me` (mot de passe requis)                                                           |
| Profil         | `GET/PUT /me/profile`, `PATCH /me/settings`, `GET /interests`                                                                |
| Photos         | `GET/POST /me/photos`, `PUT /me/photos/order`, `PUT/DELETE /me/photos/:id`, `GET /media/photos/:id/:variant?exp&sig`         |
| Découverte     | `GET /discover?limit=`, `POST /swipes {userId, action}`, `DELETE /swipes/passes`, `GET /profiles/:userId`, `DELETE /matches/:id` |
| Chat           | `GET /conversations`, `GET/POST /conversations/:id/messages`, `POST /conversations/:id/read`, `DELETE /conversations/:id`    |
| Notifications  | `GET /notifications`, `POST /notifications/read-all`, `POST /notifications/:id/read`                                         |
| Sécurité       | `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports`                                                                |
| Temps réel     | `POST /realtime/ticket` puis `GET /realtime/ws?ticket=` (événements `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`) |

## Sécurité et vie privée

- **Authentification** : bcrypt, JWT HS256 strict (algorithme vérifié, expiration obligatoire, claims `uid`/`sid`),
  access token 15 min, refresh token opaque 30 jours, haché en base, **rotation à usage unique** (le rejeu est refusé,
  même en concurrence). Le token d'accès est lié à une session en base : déconnexion, changement de mot de passe ou
  suppression du compte le révoquent immédiatement. Temps de réponse constant pour les e-mails inconnus.
- **Récupération** : code à 8 caractères (≈ 40 bits), valable 30 min, 5 essais maximum, 3 demandes/heure, stocké en
  HMAC (une fuite de base ne permet pas de le retrouver), réponse identique que le compte existe ou non, toutes les
  sessions sont révoquées après réinitialisation.
- **Autorisations côté serveur** : chaque règle (âge, préférences mutuelles, blocages, appartenance à une conversation,
  propriété d'une photo) est appliquée en SQL/service, jamais déléguée au client. Une conversation étrangère répond
  `404` (aucune fuite d'existence).
- **Données exposées** : un profil public ne contient ni e-mail, ni date de naissance, ni coordonnées, ni drapeaux de
  visibilité : seulement prénom, âge, bio, ville, centres d'intérêt, photos et une **distance par paliers**
  (1 km sous 2 km, pas de 1 km jusqu'à 10 km, puis pas de 5 km), masquable.
- **Localisation minimale** : seules des coordonnées **arrondies à ~1 km** sont stockées ; la position exacte ne quitte
  jamais l'appareil. Un repli « choisir une ville » évite d'exiger la permission GPS.
- **Photos** : type détecté sur le contenu (pas sur l'en-tête du client), JPEG/PNG uniquement, 8 Mo max, dimensions
  bornées avant décodage (anti « decompression bomb »), **ré-encodage systématique** (supprime EXIF/GPS et tout contenu
  polyglotte), 6 photos max, noms de fichiers générés côté serveur, stockage hors racine web, accès uniquement via
  des **URLs signées HMAC expirantes** (1–2 h) délivrées aux seules personnes autorisées à voir le profil.
- **Abus** : rate limiting (auth 20/min/IP, récupération 5/15 min, likes 120/min, messages 60/min, uploads 30/h,
  signalements 10/h), corps JSON limités à 1 Mo, validation stricte de toutes les entrées (prénom, dates, bornes
  géographiques, énumérations, longueurs, curseurs), caractères de contrôle retirés, requêtes SQL paramétrées.
- **Suppression de compte** : mot de passe requis ; la base cascade sur profil, photos, swipes, matchs, messages,
  blocages, notifications, sessions ; les fichiers sont effacés du stockage.
- **Journaux** : JSON structuré avec masquage des secrets (tokens, mots de passe…).

## Décisions techniques et produit

- **Construit sur l'existant** : Go/Gin/pgx, sessions JWT, Expo, TanStack Query et le kit `shared/ui` sont conservés.
  Retirés car hors produit : Stripe/facturation, multi-tenant (organisations), posts/commentaires/votes, « files »
  (métadonnées sans upload réel), liste globale d'utilisateurs (fuite de données).
- **Pas de PostGIS** : distance par formule de haversine en SQL avec pré-filtre par boîte englobante indexée.
  Suffisant pour un MVP ; migrer vers PostGIS (`geography` + index GiST) est le chemin d'évolution.
- **Recommandation extensible** : le dépôt SQL ne renvoie que des candidats *éligibles* (jamais de blocage, jamais de
  profil déjà traité) ; l'interface `Ranker` (`features/matching/ranker.go`) ne décide que de l'*ordre*. Le ranker par
  défaut favorise proximité et centres d'intérêt communs. Remplaçable sans toucher au SQL ni à l'API.
- **Éligibilité mutuelle** : on ne voit (et on ne peut liker) qu'une personne qui correspond à nos critères **et** dont
  les critères nous incluent (genre, âge, distance), visible, avec au moins une photo, sans blocage.
- **Match atomique** : like + détection de réciprocité + match + conversation + notifications dans **une transaction**,
  verrou consultatif sur la paire : deux likes simultanés créent exactement un match.
- **Un « passer » est définitif**… mais réversible volontairement (« Revoir les profils passés »).
- **Supprimer un match** efface la conversation pour les deux ; les deux restent mutuellement exclus de la découverte.
  **Bloquer** supprime le match et coupe toute visibilité dans les deux sens. **Masquer** une conversation est local
  (réapparaît à un nouveau message, sans l'historique masqué).
- **Date de naissance non modifiable** après création (anti-contournement du contrôle d'âge) ; âge 18–99 vérifié côté
  serveur ; le motif de signalement « mineur » existe.
- **Temps réel** : WebSocket authentifié par ticket à usage unique de 30 s (jamais le token d'accès dans l'URL) ; hub en
  mémoire. Chaque événement est aussi disponible via REST : le client se resynchronise à la reconnexion.
- **Notifications** : en base (match, message — une seule non lue par conversation) et poussées en direct ; pas encore
  de notifications push système (APNs/FCM), voir [Limites](#limites-connues).
- **Thème clair uniquement** et **interface en français** : choix de simplicité ; les libellés sont centralisés dans les
  écrans/`labels.ts` pour faciliter une future i18n.
- **Dépendances ajoutées** (justifiées) : `gorilla/websocket` (temps réel), `golang.org/x/image` (redimensionnement),
  `google/uuid`, `expo-image-picker`, `expo-location`, `expo-linear-gradient`, `@react-navigation/native-stack`,
  `react-native-web`/`react-dom` (build web + E2E), `@playwright/test`, ESLint/Prettier.

## Déploiement

1. Fournir PostgreSQL (managé de préférence) et un `JWT_SECRET` fort ; `APP_ENV=production`.
2. Construire l'image : `docker build -t tendre-api services/api` (utilisateur non-root, `GIN_MODE=release`).
3. Monter un **volume persistant** sur `UPLOAD_DIR` (`/data/uploads`) — ou implémenter `storage.Storage` pour S3/GCS.
4. Placer l'API derrière un proxy TLS (obligatoire : les jetons transitent dans les en-têtes) ; renseigner
   `TRUSTED_PROXIES` avec l'IP du proxy pour que le rate limiting voie les vraies IP.
5. Configurer SMTP (récupération de compte) et `ALLOWED_ORIGINS` si vous publiez la version web.
6. Mobile : `EXPO_PUBLIC_API_URL=https://api.exemple.fr` puis builds EAS / `expo export --platform web`.

Mise à l'échelle horizontale : le hub WebSocket et le rate limiter sont en mémoire. Au-delà d'une instance, branchez
`realtime.Publisher` sur Postgres `LISTEN/NOTIFY` ou Redis et remplacez le store du limiter.

## Modération

Pas d'interface d'administration (hors périmètre) ; les signalements sont en base, à traiter avec SQL :

```sql
-- signalements ouverts, du plus récent au plus ancien, avec le nombre de signalements par personne
SELECT r.id, r.created_at, r.reason, r.details, r.reported_id, p.first_name,
       count(*) OVER (PARTITION BY r.reported_id) AS reports_for_user
FROM reports r LEFT JOIN profiles p ON p.user_id = r.reported_id
WHERE r.status = 'open' ORDER BY r.created_at DESC;

-- suspendre un profil de la découverte
UPDATE profiles SET is_visible = FALSE WHERE user_id = '<uuid>';

-- supprimer un compte (cascade) puis marquer les signalements traités
DELETE FROM users WHERE id = '<uuid>';
UPDATE reports SET status = 'reviewed' WHERE reported_id = '<uuid>';
```

## Limites connues

- Pas de notifications push système (APNs/FCM) : les notifications sont dans l'application et en temps réel tant
  qu'elle est ouverte.
- Pas de vérification de l'adresse e-mail à l'inscription ni de vérification de photo.
- Pas d'interface de modération (voir ci-dessus).
- Hub temps réel, rate limiter et stockage disque : une seule instance par défaut (voir Déploiement).
- Orientation EXIF des photos non appliquée côté serveur (le sélecteur d'images mobile ré-encode en orientation
  correcte).
- Anciens modules du boilerplate (`features/users|posts|comments|billing|files`, `docs/agents/payments-agent.md`,
  `apps/mobile/src/theme`, `tamagui.config.ts`) : **non câblés** et à supprimer — voir `CHANGELOG.md`.
