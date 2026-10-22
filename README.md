# Aurore — application de rencontre 100 % gratuite

Aurore est une application de rencontre complète : inscription, profil et photos, découverte par swipe, matchs réciproques, messagerie en temps réel, notifications, blocage, signalement et suppression de compte.

**Gratuite pour de vrai** : aucun abonnement, aucun paywall, aucun « boost », aucune limite artificielle de likes, aucune publicité. Le code ne contient d'ailleurs aucun module de paiement.

Le dépôt est parti du boilerplate `go-react-saas` ; ce README décrit l'état actuel du code.

## Sommaire

- [Pile technique](#pile-technique)
- [Fonctionnalités](#fonctionnalités)
- [Architecture](#architecture)
- [Démarrage rapide](#démarrage-rapide)
- [Variables d'environnement](#variables-denvironnement)
- [Base de données, migrations, seed](#base-de-données-migrations-seed)
- [Lancer en local](#lancer-en-local)
- [Tests et qualité](#tests-et-qualité)
- [Build et déploiement](#build-et-déploiement)
- [API REST](#api-rest)
- [Sécurité et vie privée](#sécurité-et-vie-privée)
- [Décisions techniques](#décisions-techniques)
- [Modération](#modération)
- [Limites connues](#limites-connues)

## Pile technique

| Couche | Technologies |
| --- | --- |
| API | Go 1.24, Gin, PostgreSQL 16, `pgx/v5` (pool partagé), JWT HS256, bcrypt, WebSocket (`gorilla/websocket`), `golang.org/x/image` — pas d'ORM |
| Mobile / web | Expo 54, React Native 0.81, React 19, TypeScript strict, React Navigation, TanStack Query |
| Tests | `go test` (unitaires + intégration PostgreSQL), Jest, Playwright (parcours complet dans un navigateur) |
| Infra | Docker, Docker Compose, GitHub Actions |

## Fonctionnalités

- **Compte** : inscription, connexion, sessions avec rotation du refresh token, déconnexion, récupération de mot de passe par email (code à usage unique, 1 h), suppression définitive du compte.
- **Onboarding** : prénom, date de naissance (18 ans minimum, vérifié côté serveur), genre, préférences de rencontre (genres, tranche d'âge, distance), localisation, 1 à 6 photos, bio, centres d'intérêt.
- **Découverte** : cartes à swipe (gestes **et** boutons accessibles), galerie photo, profil complet, like / passer. Un profil traité ne réapparaît jamais.
- **Matchs** : like réciproque ⇒ match, créé de façon atomique (sans doublon, même si les deux likes arrivent au même instant). « C'est un match » en temps réel des deux côtés.
- **Messagerie** : texte, heure, lu / non lu, liste des conversations paginée, masquage local d'une conversation, annulation de match. Temps réel via WebSocket.
- **Notifications** : matchs et nouveaux messages (une notification non lue par conversation), badges sur les onglets.
- **Vie privée** : mode invisible (ne plus apparaître dans la découverte), masquer la distance, distance toujours approximative (par paliers de 5 km).
- **Sécurité** : blocage (retire le match et la conversation, invisibilité mutuelle), signalement avec motif, limitation de débit.

## Architecture

```text
.
├─ services/api/                 API Go (un seul serveur HTTP, Gin)
│  ├─ cmd/api/                   point d'entrée + tests d'intégration
│  ├─ cmd/seed/                  données de démonstration (dev uniquement)
│  └─ internal/
│     ├─ platform/               config, db (+migrations), middleware, logger, errors, geo, realtime, mailer
│     └─ features/               auth, profiles, photos, matching, chat, notifications, safety
│        └─ <feature>/           handler.go → service.go → repository.go (+ model.go, routes.go)
├─ apps/mobile/                  application Expo (iOS, Android, web)
│  ├─ src/api/                   client REST unique (aucun appel réseau dans les écrans)
│  ├─ src/hooks/                 auth, temps réel, files de données
│  ├─ src/screens, components/   écrans et composants
│  ├─ src/shared/ui, layout/     design system (jetons, boutons, champs…)
│  └─ e2e/                       Playwright (web) et Maestro (natif)
├─ docker-compose.yml            PostgreSQL + API
└─ docs/ARCHITECTURE.md          détails et décisions
```

Règles : `handler → service → repository`, pas de SQL dans les handlers, validation et règles métier côté serveur, une seule piscine de connexions.

## Démarrage rapide

Prérequis : Docker, Node 20+, (Go 1.24+ pour travailler sur l'API hors Docker).

```bash
cp .env.example .env
# éditez .env : JWT_SECRET (openssl rand -base64 48)
docker compose up --build -d          # PostgreSQL + API sur http://localhost:18080
curl http://localhost:18080/health     # {"status":"ok"}

# Données de démonstration (facultatif, jamais en production)
docker compose exec api ./seed

cd apps/mobile
npm ci
npm run web                            # ou : npm start (Expo Go, simulateurs)
```

Compte de démonstration : `camille.01@demo.invalid` / `demo-password-123` (14 comptes `*.NN@demo.invalid`).

### Téléphone physique

Ne jamais utiliser `localhost` sur un téléphone : utilisez l'IP locale de votre machine.

```bash
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npm start     # ou : npm run start:phone
curl http://<IP_LAN>:18080/health                        # à vérifier avant de lancer l'app
```

Si le port 18080 est déjà pris, changez le mapping de port dans `docker-compose.yml` / `infra/docker-compose.yml`, `EXPO_PUBLIC_API_URL`, `apps/mobile/scripts/start-phone.js` et ce README.

Le web (`npm run web`) appelle l'API depuis un autre port : ajoutez l'origine à `ALLOWED_ORIGINS` (ex. `http://localhost:8081`).

## Variables d'environnement

Tout est documenté dans [`.env.example`](.env.example). Aucun secret n'est versionné ni embarqué dans l'application mobile.

| Variable | Rôle | Défaut |
| --- | --- | --- |
| `JWT_SECRET` | Secret HS256, 32 caractères minimum, valeurs d'exemple refusées | **obligatoire** |
| `DATABASE_URL` | Chaîne de connexion PostgreSQL | **obligatoire** |
| `PORT` | Port de l'API | `8080` |
| `APP_ENV` | `development`, `test` ou `production` (production exige `SMTP_HOST`) | `development` |
| `UPLOAD_DIR` | Dossier des photos traitées | `./data/uploads` |
| `ALLOWED_ORIGINS` | Origines navigateur autorisées (CSV) | vide |
| `TRUSTED_PROXIES` | Proxys dont `X-Forwarded-For` est cru (CSV) | vide |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `MAIL_FROM` | Envoi des emails de récupération | log serveur en dev |
| `PASSWORD_RESET_URL` | Lien ajouté à l'email de récupération | vide |
| `EXPO_PUBLIC_API_URL` | URL de l'API pour l'app (publique, à la compilation) | `localhost:18080` / `10.0.2.2:18080` en dev |

## Base de données, migrations, seed

- Les migrations SQL sont embarquées dans le binaire (`internal/platform/db/migrations`) et appliquées **au démarrage**, dans l'ordre, chacune dans une transaction. La table `schema_migrations` mémorise les versions appliquées, un verrou consultatif évite les courses entre instances.
- `014_dating_schema.sql` remplace les tables génériques du boilerplate (posts, commentaires, organisations, abonnements…). Le boilerplate ne contenait pas de données de production : elles sont supprimées, pas migrées.
- Tables : `users`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`, `password_resets`, `sessions`.
- Contraintes : clés étrangères en `ON DELETE CASCADE`, unicité (un match par paire grâce à `user_a_id < user_b_id`, un swipe par paire orientée, une photo par position), `CHECK` sur les énumérations et les bornes, index sur chaque chemin chaud (découverte, messages, notifications…).
- `go run ./cmd/seed` (ou `./seed` dans le conteneur) crée 14 comptes fictifs `@demo.invalid` avec des photos abstraites générées (aucune vraie personne). Il est idempotent et refuse de s'exécuter si `APP_ENV=production`.

## Lancer en local

API sans Docker :

```bash
cd services/api
export JWT_SECRET=$(openssl rand -base64 48)
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
go run ./cmd/api
```

Mobile : `cd apps/mobile && npm ci && npm start` (ou `npm run web`, `npm run android`, `npm run ios`).

## Tests et qualité

```bash
# API (les tests d'intégration utilisent une base PostgreSQL JETABLE : le schéma public y est recréé)
cd services/api
export DATABASE_URL_TEST=postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable
gofmt -l .            # doit être vide
go vet ./...
go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npx tsc --noEmit
npm run lint
npm run format:check
npm test              # Jest : règles de validation, formats

# Parcours complet dans un navigateur : inscription → profil → découverte → like → match → chat → blocage → suppression
EXPO_PUBLIC_API_URL=http://localhost:18081 npx expo export --clear --platform web --output-dir dist
#   (API démarrée sur 18081 avec ALLOWED_ORIGINS=http://localhost:19006 et une base VIDE)
npx playwright install chromium && npx playwright test

# Smoke test API (register, login, profil, découverte, match, chat, notifications)
MOBILE_E2E_API_URL=http://localhost:18080 npm run e2e:smoke
```

Les tests d'intégration couvrent : authentification et récupération de compte, validation des profils (18+), uploads dangereux, contrôle d'accès aux photos, filtres de découverte, règles de swipe, match + chat + permissions, blocage / signalement, suppression de compte, **likes simultanés** (un seul match), WebSocket, limitation de débit.

Test natif optionnel : `npm run e2e:ui:setup` puis `npm run e2e:ui:run` (Maestro, PowerShell).

## Build et déploiement

- Image API : `docker build -t aurore-api services/api` (multi-étapes, utilisateur non-root, volume `/data/uploads`, `HEALTHCHECK`).
- Production : `APP_ENV=production`, `JWT_SECRET` fort, `DATABASE_URL` vers un PostgreSQL géré avec `sslmode=require`, SMTP configuré, TLS terminé par un reverse proxy (renseigner `TRUSTED_PROXIES` pour que la limitation de débit voie la vraie IP), volume persistant pour `UPLOAD_DIR` (ou à terme un stockage objet, voir *Limites*), sauvegardes de la base **et** des photos.
- Application : `npx expo export --platform web` pour le web ; builds natifs avec EAS (`eas build`), `EXPO_PUBLIC_API_URL` pointant vers l'API publique en HTTPS.

## API REST

Toutes les routes (sauf `/health` et `/auth/*`) exigent `Authorization: Bearer <accessToken>`. Erreurs : `{"error": "message", "field": "champ"?}`. Les listes sont paginées par curseur (`?limit=&cursor=`).

| Domaine | Routes |
| --- | --- |
| Santé | `GET /health` |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `/auth/forgot-password`, `/auth/reset-password` · `GET /me` · `DELETE /me` (mot de passe requis) |
| Profil | `GET/PUT /profile` · `GET/PUT /preferences` · `GET /interests` · `GET /profiles/:userId` |
| Photos | `GET/POST /photos` · `PUT /photos/order` · `PUT /photos/:id` (remplacer) · `PUT /photos/:id/primary` · `DELETE /photos/:id` · `GET /photos/:id/file?size=thumb` |
| Découverte | `GET /discover?limit=` · `POST /discover/swipes {userId, action: like\|pass}` · `GET /matches` · `DELETE /matches/:id` |
| Chat | `GET /conversations` · `GET/POST /conversations/:id/messages` · `POST /conversations/:id/read` · `DELETE /conversations/:id` (masquer) |
| Notifications | `GET /notifications` · `POST /notifications/:id/read` · `POST /notifications/read-all` |
| Sécurité | `GET/POST /blocks` · `DELETE /blocks/:userId` · `POST /reports {userId, reason, details, alsoBlock}` |
| Temps réel | `GET /ws` (WebSocket) : premier message `{"type":"auth","token":"…"}`, puis événements `match.new`, `match.removed`, `message.new`, `message.read`, `notification.new` |

## Sécurité et vie privée

- **Authentification** : bcrypt, JWT HS256 strict (algorithme vérifié, expiration obligatoire, claims `uid`/`sid`), access token 15 min, refresh token aléatoire haché en base et tourné à chaque usage. Chaque requête vérifie que la session existe encore et que le compte est actif : déconnexion ou suppression coupent l'accès immédiatement (y compris les WebSocket).
- **Autorisation côté serveur partout** : une conversation n'est lisible que par ses deux membres (sinon `404`, sans fuite d'existence), un swipe n'est accepté que sur un profil que la découverte aurait proposé (mêmes règles SQL), les photos ne sont servies qu'à un utilisateur autorisé (propriétaire, profil visible non bloqué, ou match).
- **Entrées** : validation stricte (âge ≥ 18, énumérations, bornes, longueurs, nettoyage des caractères de contrôle), corps JSON limité à 64 Ko, UUID vérifiés, SQL 100 % paramétré.
- **Uploads** : détection par contenu (JPEG/PNG/WebP), 8 Mo max, dimensions bornées contre les bombes de décompression, **ré-encodage systématique en JPEG** (supprime EXIF/GPS et tout contenu polyglotte), noms de fichiers générés côté serveur, chemins vérifiés contre la traversée, en-têtes `nosniff` et CSP restrictive.
- **Localisation** : coordonnées arrondies à ~1 km avant stockage (côté client puis serveur), jamais renvoyées aux autres, distance affichée par paliers de 5 km (« À moins de 5 km », « À environ 15 km »), désactivable.
- **Abus** : limitation de débit par IP (auth), par email (connexion), par utilisateur (swipes, messages, uploads, signalements). Réponse identique que l'email existe ou non pour la récupération de mot de passe, comparaison bcrypt factice pour égaliser les temps de connexion.
- **Suppression** : `DELETE /me` supprime en une transaction profil, photos (fichiers compris), swipes, matchs, messages, blocages, notifications, sessions et signalements liés au compte.

## Décisions techniques

- **On garde Go / Gin / pgx / Expo** : l'architecture du boilerplate (feature-first, `handler → service → repository`, SQL explicite) convenait ; on a retiré ce qui était incompatible avec « 100 % gratuit » (Stripe, abonnements, organisations multi-locataires) et le contenu générique (posts, commentaires).
- **Temps réel = WebSocket serveur → client** ; l'envoi passe par l'API REST (validation, limitation de débit et persistance uniques). Le hub est en mémoire : pour plusieurs instances, brancher `LISTEN/NOTIFY` ou Redis derrière l'interface `realtime.Publisher`. À la reconnexion, le client resynchronise ses données par REST.
- **Pas de PostGIS** : bounding box sur la latitude + formule de haversine en SQL, suffisant et sans dépendance (≈ 55 ms pour une page avec 30 000 profils dans la même zone).
- **Recommandation simple et évolutive** : les personnes qui vous ont déjà liké d'abord, puis le nombre d'intérêts communs, puis la distance. Tout le classement tient dans un `ORDER BY` unique (`matching/repository.go`), point d'extension prévu.
- **Passes définitives** : un profil passé ne revient pas (pas de « rewind » payant à mimer). Les rencontres se défont avec « Annuler le match ».
- **Date de naissance non modifiable** après création (anti-fraude sur l'âge) ; seul l'âge est affiché.
- **Conversation masquée = historique effacé de son côté** : un nouveau message la fait réapparaître sans les anciens messages.
- **Pas de mode sombre** pour cette version : un thème unique (palette « Aurore » chaude) a été retenu pour rester cohérent et accessible (contrastes ≥ 4,5:1, cibles tactiles ≥ 44 px, libellés d'accessibilité, aucun geste sans équivalent bouton).
- **Pas de notifications push** (Expo Push/APNs/FCM) : les notifications sont intégrées (liste + badges + temps réel). Ajouter le push demande des identifiants développeur Apple/Google.
- **Navigateur : photos** récupérées avec le jeton puis affichées depuis un `Blob` (un `<img>` ne peut pas envoyer d'en-tête `Authorization`).

## Modération

Les signalements sont stockés dans `reports` (`status`: `open`, `reviewed`, `dismissed`). Il n'y a pas encore d'interface d'administration : à consulter en SQL.

```sql
SELECT r.created_at, r.reason, r.details, r.reported_id, count(*) OVER (PARTITION BY r.reported_id) AS reports_on_user
FROM reports r WHERE r.status = 'open' ORDER BY r.created_at;

-- Suspendre un compte (connexion et sessions refusées immédiatement, profil retiré de la découverte)
UPDATE users SET status = 'suspended' WHERE id = '<uuid>';
```

## Limites connues

- Stockage des photos sur disque local (volume Docker) : pour plusieurs instances API, déplacer `photos.Storage` vers un stockage objet (S3 compatible) derrière la même interface.
- Hub temps réel en mémoire (voir ci-dessus) ; limitation de débit en mémoire par instance.
- Pas d'interface d'administration ni de vérification de photo ; pas de push.
- Création de compte : le message « email déjà utilisé » révèle l'existence d'un compte (compromis classique d'ergonomie).
- Docker n'est pas exécuté dans l'environnement de développement automatisé : les fichiers Compose sont validés (`docker compose config`) mais l'image est construite par la CI.
- Fichiers hérités du boilerplate non supprimés (suppression à valider, voir `CHANGELOG.md`).
