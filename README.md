# Lien — application de rencontre 100 % gratuite

Lien est une application de rencontre mobile (React Native / Expo) adossée à une API Go. Deux personnes créent un profil, se découvrent, se likent, **matchent** lorsque l'intérêt est réciproque, puis discutent **en temps réel**.

**Gratuite, vraiment** : aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de likes, aucune publicité. Il n'existe dans le code aucune notion de paiement.

## Sommaire

- [Fonctionnalités](#fonctionnalités)
- [Stack technique](#stack-technique)
- [Architecture](#architecture)
- [Installation et lancement](#installation-et-lancement)
- [Variables d'environnement](#variables-denvironnement)
- [Base de données, migrations, seed](#base-de-données-migrations-seed)
- [Tests, lint et build](#tests-lint-et-build)
- [Aperçu de l'API](#aperçu-de-lapi)
- [Sécurité et vie privée](#sécurité-et-vie-privée)
- [Décisions techniques et produit](#décisions-techniques-et-produit)
- [Déploiement](#déploiement)
- [Limites connues](#limites-connues)
- [Dépannage](#dépannage)

## Fonctionnalités

| Domaine | Ce que fait l'application |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation du refresh token, déconnexion, récupération du mot de passe par code e-mail, suppression définitive du compte |
| Onboarding | Prénom, date de naissance (18 ans minimum, vérifié côté serveur), genre, localisation approximative, préférences de rencontre, bio, centres d'intérêt, 1 à 6 photos |
| Découverte | Pile de profils (swipe **ou** boutons), galerie photos, profil complet, intérêts en commun mis en avant, jamais deux fois le même profil |
| Matching | Like / passer ; like réciproque = match ; aucun doublon, même avec des likes simultanés ; page « Ils vous aiment » (gratuite) |
| Chat | Conversation ouverte uniquement après un match, temps réel (WebSocket), heure, lu / non lu, accusé de lecture, masquer une conversation, défaire un match |
| Notifications | Nouveau match / nouveau message, centre de notifications, pastilles non lues, mises à jour en direct |
| Confidentialité | Masquer l'âge, la distance/la ville, ou sortir de la découverte ; distance toujours arrondie (tranches de 5 km) |
| Sécurité | Bloquer, signaler (avec blocage optionnel), masquage automatique après 3 signalements distincts, outil de modération pour l'opérateur |
| Photos | Upload validé par le contenu, recompression JPEG, ordre, photo principale, remplacement, suppression, accès privé |
| Interface | Identité visuelle originale (crème / terracotta), mode sombre automatique, états de chargement, vides et d'erreur, accessibilité (libellés, rôles, alternatives aux gestes) |

## Stack technique

- **Mobile** : Expo SDK 54, React Native 0.81, TypeScript strict, React Navigation (stack + onglets), TanStack Query. Aucune librairie d'animation ni de gestes supplémentaire : le swipe utilise `PanResponder` + `Animated`.
- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool` (pas d'ORM), JWT HS256, bcrypt, WebSocket (`gorilla/websocket`).
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Architecture

```text
services/api/
  cmd/api/            point d'entrée HTTP + tests d'intégration (Postgres réel)
  cmd/seed/           données de démonstration (jamais en production)
  cmd/moderation/     outil opérateur : revue des signalements, suspension
  internal/
    platform/         config, db (migrations suivies), middleware (auth, rate limit,
                      body limit, CORS), realtime (hub WebSocket), mailer, storage, logger
    features/
      auth/           inscription, connexion, refresh, récupération de mot de passe
      profiles/       profil, préférences, intérêts, localisation, vue publique
      photos/         pipeline d'images et accès privé
      matching/       découverte (filtres SQL + ranker), swipes, matchs, « likes reçus »
      chat/           conversations de match, messages, lecture
      notifications/  création côté serveur, lecture, non lus
      safety/         blocages et signalements
      account/        suppression du compte
apps/mobile/
  App.tsx             garde de navigation : connexion → onboarding → application
  src/api/            client HTTP (refresh automatique), types, appels par domaine
  src/screens/        écrans
  src/components/     deck de swipe, carte profil, formulaires (profil, préférences, photos…)
  src/shared/         design system (tokens clair/sombre, Button, Input…), feedback, layout
  src/realtime/       connexion WebSocket authentifiée avec reconnexion
  src/lib/            logique pure testée (validation, formats, JWT, erreurs)
```

Chaque feature backend suit `handler → service → repository` : handlers fins, aucune SQL dans les handlers, aucune logique métier dans les handlers.

**Modèle de données** (migrations `014` à `016` pour le domaine, `001`–`013` pour l'existant) :
`users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (like/pass), `matches`, `conversations` + `conversation_participants` + `messages`, `blocks`, `reports`, `notifications`. Contraintes d'unicité, clés étrangères en cascade, `CHECK` (âge ≥ 18, genres, longueurs, coordonnées, `user_a < user_b` pour qu'un couple n'ait qu'une ligne), et index pour chaque chemin chaud (`swipes` des likes reçus, `messages (conversation_id, created_at)`, `notifications (user_id, created_at)`, `profiles (latitude)`…).

## Installation et lancement

### Prérequis

Docker (recommandé) **ou** Go 1.24+ et PostgreSQL 16 ; Node.js 22+ pour l'application mobile.

### 1. API + base de données avec Docker

```bash
cp .env.example .env
# Renseignez JWT_SECRET :  openssl rand -hex 32
docker compose up --build
curl http://localhost:18080/health        # {"status":"ok"}
```

L'API écoute sur le port hôte **18080** (voir [Dépannage](#dépannage)). Les migrations s'appliquent au démarrage.

### 2. Données de démonstration (facultatif, développement uniquement)

```bash
docker compose exec api ./seed            # ou : cd services/api && go run ./cmd/seed
```

Crée 12 comptes `*@demo.invalid` (domaine réservé, jamais livrable) avec photos générées, dont `demo@demo.invalid` / `Demo-password-123`, déjà « liké » par 3 profils. Retrait : `./seed -purge`. Le seed **refuse de tourner** si `APP_ENV=production`.

### 3. Application mobile

```bash
cd apps/mobile
npm ci
# Téléphone physique : JAMAIS localhost. Utilisez l'IP LAN de la machine qui fait tourner l'API.
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start
# ou : npm run start:phone   (détecte l'IP LAN et la configure)
```

Émulateur Android : l'URL par défaut est `http://10.0.2.2:18080`. Simulateur iOS : `http://localhost:18080`. Scannez le QR code avec Expo Go ; les permissions photos/localisation sont demandées au moment utile.

### Sans Docker

```bash
createdb app && cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
go run ./cmd/api
```

## Variables d'environnement

Voir [`.env.example`](.env.example) (documenté ligne à ligne). Aucun secret n'est versionné ni embarqué dans l'application mobile.

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | oui | — | Secret HS256, ≥ 32 caractères, valeurs « placeholder » refusées |
| `DATABASE_URL` | oui | — | Connexion PostgreSQL |
| `APP_ENV` | non | `development` | `development`, `test` ou `production` |
| `PORT` | non | `8080` | Port d'écoute dans le conteneur |
| `UPLOAD_DIR` | non | `./data/uploads` | Stockage des photos (volume Docker en compose) |
| `ALLOWED_ORIGINS` | non | vide | Origines CORS (clients web) |
| `TRUSTED_PROXIES` | non | vide | Proxies dont `X-Forwarded-For` est cru (rate limit par IP) |
| `SMTP_HOST`/`PORT`/`USERNAME`/`PASSWORD`/`FROM` | production | vide | E-mail de récupération de mot de passe. En développement, le code est affiché dans les logs de l'API |
| `EXPO_PUBLIC_API_URL` | mobile | émulateur | URL de l'API vue **depuis le téléphone** |

## Base de données, migrations, seed

- Les migrations SQL sont embarquées dans le binaire (`internal/platform/db/migrations`) et appliquées **une seule fois** au démarrage, dans l'ordre, chacune dans une transaction, sous verrou consultatif (plusieurs instances peuvent démarrer ensemble). L'historique est dans `schema_migrations`.
- Ajouter une migration : créer `NNN_description.sql` (numéro suivant). Elle ne doit jamais être modifiée après publication.
- Données de démo : `cmd/seed` (voir ci-dessus), strictement séparé du code applicatif.

## Tests, lint et build

```bash
./scripts/check.sh            # tout : gofmt, vet, tests Go, build, tsc, eslint, prettier, tests mobile
```

Détail :

```bash
# Backend (PostgreSQL requis pour les tests d'intégration ; TEST_DATABASE_URL, défaut postgres://postgres:postgres@localhost:5432/app_test)
cd services/api
gofmt -l . ; go vet ./... ; go test ./... ; go build ./cmd/api

# Mobile
cd apps/mobile
npm ci && npx tsc --noEmit && npx eslint . --max-warnings 0 && npm test
```

- **Tests unitaires** : validation des profils (âge 18+, noms, préférences, arrondi de localisation), pipeline d'images (types dangereux, polyglottes, bombes de pixels, orientation EXIF), ranker, rate limiter, stockage, configuration, utilitaires mobile (validation, formats, JWT, erreurs).
- **Tests d'intégration** (`cmd/api`, vrai PostgreSQL, routeur complet) : authentification stricte (alg=none, mauvais secret, expiré, session révoquée), récupération de mot de passe, onboarding, filtres de découverte mutuels, likes/matchs (idempotence, 12 paires de likes **simultanés** → 12 matchs exacts), permissions du chat (un tiers n'accède jamais à une conversation), blocage, signalement + masquage automatique, suspension, suppression de compte (aucune donnée résiduelle, fichiers supprimés), WebSocket (livraison en direct, rejet des clients non authentifiés), limites de taille.
- **Parcours critique** : `TestCriticalJourney` (inscription → profil → photos → découverte → like → match → conversation → notifications).
- **E2E API** : `MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke` rejoue le parcours contre une API qui tourne, WebSocket compris.
- **E2E UI** : flow [Maestro](https://maestro.mobile.dev) dans `apps/mobile/e2e/maestro/critical-flow.yaml` (`npm run e2e:ui:setup` puis `npm run e2e:ui:run`, nécessite un appareil ou un émulateur).
- **CI** : `.github/workflows/ci.yml` (PostgreSQL en service, `REQUIRE_DB=1` : les tests d'intégration ne peuvent pas être ignorés silencieusement).

Build mobile de production : `npx expo export --platform android|ios` (bundle Metro) ou EAS Build. Pensez à définir `EXPO_PUBLIC_API_URL` (HTTPS) au moment du build.

## Aperçu de l'API

Toutes les routes, hors `/health`, `/auth/*` et `/ws`, exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "message"}` (+ `"code"` quand le client doit réagir, ex. `profile_incomplete`). Listes paginées (`limit` borné, `offset` ou curseur).

| Méthode | Route | Rôle |
| --- | --- | --- |
| POST | `/auth/register` `/auth/login` `/auth/refresh` `/auth/logout` | Session |
| POST | `/auth/password/forgot` `/auth/password/reset` | Récupération par code e-mail |
| GET | `/me` | Compte courant |
| GET/PUT | `/profile` `/preferences` · PUT `/profile/privacy` `/profile/location` `/profile/interests` | Mon profil, état d'onboarding |
| GET | `/interests` | Catalogue d'intérêts |
| GET | `/profiles/:userId` | Profil d'un autre (si visible) |
| GET/POST/DELETE, PUT | `/photos`, `/photos/:id`, `/photos/order`, GET `/photos/:id/content` | Photos privées |
| GET | `/discover` | Prochains profils |
| POST | `/swipes` `{userId, action: like\|pass}` | Décision (idempotente) |
| GET | `/likes/received` | Qui m'a liké |
| GET | `/matches` · GET/POST `/matches/:id/messages` · POST `/matches/:id/read` `/hide` · DELETE `/matches/:id` | Conversations |
| GET | `/notifications` · POST `/notifications/:id/read` `/notifications/read-all` | Notifications |
| GET/POST/DELETE | `/blocks` · POST `/reports` | Sécurité |
| DELETE | `/account` `{password}` | Suppression définitive |
| GET | `/ws` | WebSocket : première trame `{"type":"auth","token":"…"}`, puis événements `message.new`, `message.read`, `match.new`, `match.ended`, `notification.new` |

## Sécurité et vie privée

- **Auth** : JWT HS256 uniquement (méthode et claims strictement vérifiés, expiration obligatoire), access token 15 min, refresh token aléatoire haché en base et **tourné** à chaque usage. Chaque requête vérifie aussi que la session existe encore : déconnexion, suppression ou suspension coupent l'accès immédiatement. Mots de passe en bcrypt (72 octets max, contrôlé) ; temps de réponse constant pour les e-mails inconnus.
- **Autorisations côté serveur** : toute règle (âge, visibilité, appartenance à un match, blocages) est vérifiée en SQL/service. Un identifiant de conversation, de photo ou de profil fourni par un tiers répond `404`, sans révéler l'existence.
- **Abus** : rate limiting (connexion/inscription par IP, récupération de mot de passe, envoi de messages, upload, signalements, suppression), taille de corps JSON limitée, `X-Forwarded-For` ignoré sauf proxies déclarés.
- **Photos** : type déterminé par le **contenu** (jamais le nom ou le `Content-Type` client), dimensions contrôlées avant décodage, orientation EXIF appliquée puis **toutes les métadonnées supprimées** par ré-encodage JPEG, nom de fichier choisi par le serveur, fichiers jamais servis statiquement : chaque lecture est autorisée (propriétaire, profil visible, match).
- **Localisation** : coordonnées arrondies à ~1 km **avant** stockage, jamais renvoyées par l'API ; les autres voient une distance en tranches de 5 km, désactivable.
- **Suppression de compte** : mot de passe requis, suppression en cascade (profil, photos + fichiers, swipes, matchs, conversations des deux côtés, messages, notifications, signalements liés, sessions).
- **Logs** : valeurs sensibles (tokens, mots de passe, secrets) masquées.

## Décisions techniques et produit

- **On construit sur l'existant** : Gin/pgx/JWT/structure `handler → service → repository` conservés. Les modules génériques non utilisés par une appli de rencontre (`billing`/Stripe, `posts`, `comments`, `files`, liste `users`) ne sont plus branchés au routeur (voir [CHANGELOG](CHANGELOG.md)) — `GET /users` aurait en particulier exposé tous les comptes.
- **Un match = une conversation** (`matches.conversation_id`) : le chat réutilise `conversations`/`messages` mais n'est accessible que via un match actif.
- **Un swipe est définitif** : le premier choix gagne, un « passer » ne revient jamais (c'est ce qui garantit « jamais deux fois le même profil »). Les deux décisions concurrentes d'un couple sont sérialisées par un verrou consultatif PostgreSQL.
- **Filtres mutuels** : genre, tranche d'âge et distance doivent convenir **aux deux** personnes. Tri : un `Ranker` (interface) classe un pool de 200 candidats déjà filtrés en SQL — par défaut : a déjà liké → intérêts communs → proximité → activité récente. Remplaçable sans toucher au reste.
- **Temps réel par WebSocket** authentifié par une première trame (pas de token dans l'URL), un hub en mémoire, reconnexion avec backoff côté mobile et resynchronisation REST après coupure.
- **Photos en JPEG** : une seule représentation (≤ 1600 px, qualité 85) : pas de variantes à gérer, métadonnées garanties absentes.
- **Récupération de mot de passe** par code à usage unique (8 caractères, 1 h, 5 essais) plutôt que par lien : pas de lien profond à héberger pour une application mobile pure.
- **Pas de vérification d'e-mail** : simplicité ; l'e-mail est privé et ne sert qu'à la connexion et à la récupération.
- **Langue** : interface en français ; messages d'erreur de l'API en anglais (stables, utilisés comme clés de traduction côté client).
- **Modération** : 3 signalements ouverts de personnes distinctes retirent un profil de la découverte jusqu'à décision de l'opérateur :

  ```bash
  go run ./cmd/moderation reports            # file d'attente
  go run ./cmd/moderation dismiss   <userId> # non fondé : le profil réapparaît
  go run ./cmd/moderation suspend   <userId> # suspend, coupe les sessions
  go run ./cmd/moderation unsuspend <userId>
  ```

## Déploiement

1. Construire l'image : `docker build -t lien-api services/api` (binaires `api`, `seed`, `moderation`, utilisateur non-root, volume `/data/uploads`, `HEALTHCHECK`).
2. Fournir `APP_ENV=production`, `DATABASE_URL` (TLS), `JWT_SECRET` fort, `SMTP_*`, `TRUSTED_PROXIES` (votre reverse proxy), et **persister** le volume des photos.
3. Placer l'API derrière un reverse proxy HTTPS (HSTS émis automatiquement derrière `X-Forwarded-Proto: https`). Le WebSocket passe par `/ws` : activer l'upgrade `Upgrade/Connection` côté proxy.
4. Construire l'application mobile avec `EXPO_PUBLIC_API_URL=https://api.votre-domaine`.
5. Sauvegardes : base PostgreSQL **et** volume des photos ensemble.

## Limites connues

- **Une seule instance d'API** pour le temps réel et le rate limiting (hub et compteurs en mémoire). Pour passer à l'échelle : remplacer `realtime.Hub.Publish` par un bus partagé (PostgreSQL `LISTEN/NOTIFY` ou Redis) et `ratelimit.Limiter` par un magasin partagé.
- **Pas de notifications push système** (APNs/FCM exigent des identifiants de compte développeur) : les notifications sont en base, en direct dans l'application ouverte, et dans le centre de notifications.
- **Stockage des photos sur disque** local (volume). Derrière l'interface `photos.Blobs`, un stockage objet (S3) se branche sans toucher au reste.
- Pas d'interface d'administration web : la modération passe par `cmd/moderation`.
- Les tests UI (Maestro) ne couvrent pas l'onboarding (la sélection de photo dépend de la galerie de l'appareil) ; l'onboarding est couvert par les tests API.

## Dépannage

- **Le téléphone n'atteint pas l'API** : `localhost` désigne le téléphone lui-même. Utilisez `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080` et vérifiez : `docker compose ps`, `curl http://<IP_LAN>:18080/health`, pare-feu autorisant le port.
- **Conflit de port** : si un service local occupe 18080 (`netstat -ano | findstr :18080` sous Windows), changez le port hôte dans `docker-compose.yml` (`"28080:8080"`), `apps/mobile/scripts/start-phone.js`, le fallback de `src/api/client.ts` et ce README.
- **`JWT_SECRET must be…`** : génération : `openssl rand -hex 32`.
- **Tests Go ignorés** (« database unavailable ») : démarrez PostgreSQL et créez `app_test`, ou définissez `TEST_DATABASE_URL`. `REQUIRE_DB=1` transforme l'absence de base en échec.
- **Réinitialiser l'environnement local** : `docker compose down -v` (supprime base **et** photos).
