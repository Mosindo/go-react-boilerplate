# Lueur — application de rencontre 100 % gratuite

Lueur est une application de rencontre mobile (iOS, Android, et web pour le développement) avec une API Go. Elle couvre tout le parcours : inscription, profil, photos, découverte, likes, matchs, chat en temps réel, notifications, préférences, confidentialité, blocage, signalement et suppression de compte.

**Gratuit, sans contrepartie** : pas d'abonnement, pas de paywall, pas de « boost », pas de limite de likes, pas de publicité. Les seules limites sont des garde-fous anti-abus (ex. 120 swipes/minute) bien au-delà d'un usage humain.

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Démarrage rapide (Docker)](#démarrage-rapide-docker)
5. [Développement local sans Docker](#développement-local-sans-docker)
6. [Variables d'environnement](#variables-denvironnement)
7. [Base de données et migrations](#base-de-données-et-migrations)
8. [Données de démonstration](#données-de-démonstration)
9. [Lancer l'application mobile](#lancer-lapplication-mobile)
10. [Tests et qualité](#tests-et-qualité)
11. [Build et déploiement](#build-et-déploiement)
12. [API](#api)
13. [Décisions techniques et produit](#décisions-techniques-et-produit)
14. [Limites connues et pistes](#limites-connues-et-pistes)

---

## Fonctionnalités

| Domaine | Ce qui est livré |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation des refresh tokens, déconnexion immédiate, récupération par code à 8 chiffres envoyé par e-mail, suppression définitive du compte (mot de passe requis) |
| Onboarding | Parcours guidé : prénom, date de naissance (18 ans minimum), genre, ce que je cherche, bio, métier, taille, ville, préférences, centres d'intérêt, photos, position approximative |
| Profil | Édition complète, 34 centres d'intérêt (10 max), préférences (genres recherchés, âge, distance) |
| Photos | 1 à 6 photos, compression côté client, réencodage côté serveur (métadonnées EXIF/GPS supprimées), photo principale, réordonnancement, remplacement, suppression, URLs signées et expirantes |
| Découverte | Pile de cartes avec swipe et boutons, filtrage mutuel (genres, âges, distances), classement (proximité, intérêts communs, activité, réciprocité), profils déjà vus jamais reproposés, profil complet consultable |
| Matching | Like + like = match, unicité garantie même en cas de likes simultanés, conversation créée automatiquement, annulation du match |
| Chat | Conversations par match, messages texte, temps réel (WebSocket), heure d'envoi, statut « Envoyé / Lu », compteur de non-lus, pagination, suppression locale d'une conversation |
| Notifications | Notifications in-app (nouveau match) en temps réel, badge, « tout marquer comme lu » |
| Confidentialité | Pause du profil, masquage de la distance, distance arrondie, position arrondie à ~1 km avant stockage, e-mail jamais exposé |
| Sécurité & modération | Blocage (supprime le match et la conversation, invisibilité mutuelle), signalement (bloque aussi), mise à l'écart automatique d'un profil signalé par 3 comptes distincts de plus de 24 h, rate limiting, contrôle d'accès strict côté serveur |

## Stack technique

| Couche | Choix |
| --- | --- |
| Mobile | Expo SDK 54, React Native 0.81, React 19, TypeScript strict, React Navigation 7, TanStack Query 5, expo-image, expo-image-picker / -manipulator, expo-location, expo-secure-store |
| API | Go 1.24, Gin, pgx/pgxpool, JWT HS256, bcrypt, gorilla/websocket, golang.org/x/image (WebP + redimensionnement). Pas d'ORM. |
| Base | PostgreSQL 16 |
| Infra | Docker, Docker Compose, GitHub Actions |
| Qualité | `go vet`, `gofmt`, tests unitaires et d'intégration Go, ESLint (config Expo), Jest (jest-expo), smoke API Node, E2E UI Playwright sur la version web |

## Architecture

```text
.
├─ apps/mobile/                 Application Expo
│  ├─ App.tsx                   Providers (Query, Auth, SafeArea) + navigation + toasts
│  ├─ src/api/                  Client HTTP (refresh token, erreurs typées) + endpoints + DTO
│  ├─ src/auth/                 Session (SecureStore), AuthProvider
│  ├─ src/realtime/             WebSocket → cache React Query
│  ├─ src/hooks/                Requêtes/mutations (profil, conversations, swipe…)
│  ├─ src/navigation/           Pile racine, onglets, garde d'onboarding
│  ├─ src/screens/              auth, onboarding, discover, matches, notifications, profile
│  ├─ src/components/           Kit UI + composants métier (PhotoGrid, SwipeCard, ReportSheet…)
│  ├─ src/theme/tokens.ts       Design system (couleurs, typo, espacements)
│  ├─ src/utils/                Validation, formats FR, erreurs, géoloc, médias
│  ├─ scripts/e2e-smoke.js      Smoke test API du parcours critique
│  └─ e2e/web/critical-flow.mjs E2E UI (Playwright) du parcours critique
├─ services/api/
│  ├─ cmd/api/                  Point d'entrée + tests d'intégration
│  ├─ cmd/seed/                 Données de démonstration
│  └─ internal/
│     ├─ platform/              config, db (+ migrations), middleware, httpx, errors,
│     │                         tokens (JWT), ratelimit, storage, media (URLs signées),
│     │                         mailer, realtime (hub), health, logger
│     └─ features/              auth, profiles, photos, matching, chat, safety,
│                               notifications, realtime
├─ docs/ARCHITECTURE.md         Détails d'architecture et de sécurité
├─ scripts/qa.sh, qa-lite.ps1   Porte qualité locale
└─ docker-compose.yml           Postgres + API
```

Chaque feature backend suit `handler.go → service.go → repository.go` (+ `model.go`, `routes.go`) : handlers minces, règles métier dans les services, SQL uniquement dans les repositories. Dépendances à sens unique : `matching → profiles → photos`, `chat → photos`, tout le monde → `notifications`/`realtime`. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Démarrage rapide (Docker)

Prérequis : Docker, Node.js 20+.

```bash
cp .env.example .env
# Renseigner JWT_SECRET, par exemple :
echo "JWT_SECRET=$(openssl rand -base64 48)" >> .env

docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# Profils de démo (facultatif, jamais en production)
docker compose exec api ./seed
```

Puis lancer l'app (voir [Lancer l'application mobile](#lancer-lapplication-mobile)).

PowerShell : `$env:JWT_SECRET = [Convert]::ToBase64String((1..48 | % { Get-Random -Max 256 }))` avant `docker compose up --build -d`, ou renseigner `.env`.

## Développement local sans Docker

```bash
# Postgres 16 local, base "lueur"
createdb lueur
cd services/api
export DATABASE_URL=postgres://localhost:5432/lueur?sslmode=disable
export JWT_SECRET=$(openssl rand -base64 48)
export ALLOWED_ORIGINS=http://localhost:8081
go run ./cmd/api            # écoute sur :8080 (PORT pour changer)
```

Les migrations s'appliquent automatiquement au démarrage. Pensez à `PORT=18080` si l'app pointe sur le port par défaut.

## Variables d'environnement

Toutes sont documentées dans [.env.example](.env.example). Aucun secret n'est commité.

| Variable | Obligatoire | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256 (≥ 32 caractères aléatoires). Sert aussi à dériver la clé de signature des URLs de photos. Les valeurs « placeholder » sont refusées. |
| `DATABASE_URL` | oui | Connexion PostgreSQL (pool unique partagé). |
| `PORT` | non (8080) | Port HTTP de l'API. |
| `APP_ENV` | non (`development`) | `development`, `test` ou `production`. En production : profils de démo masqués, e-mails de récupération jamais journalisés, Gin en mode release. |
| `UPLOAD_DIR` | non (`./data/uploads`) | Stockage des photos (volume Docker `uploads`). |
| `ALLOWED_ORIGINS` | non | Origines web autorisées (CORS). L'app native n'en a pas besoin. |
| `TRUSTED_PROXIES` | non | IP/CIDR de reverse proxies dont `X-Forwarded-For` est fiable (pour le rate limiting par IP). |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | en production | Envoi des codes de récupération. Sans SMTP en développement, le code apparaît dans les logs de l'API. |
| `EXPO_PUBLIC_API_URL` | oui pour un téléphone | URL publique de l'API vue par l'appareil (`apps/mobile/.env`). Ce n'est pas un secret. |
| `SEED_LAT`, `SEED_LNG`, `SEED_PASSWORD` | non | Centre et mot de passe des profils de démo. |

## Base de données et migrations

- Migrations SQL embarquées dans le binaire : `services/api/internal/platform/db/migrations/*.sql`.
- Appliquées au démarrage, **une seule fois chacune**, dans une transaction, avec suivi dans `schema_migrations` et un verrou consultatif (plusieurs réplicas peuvent démarrer en même temps).
- Pour ajouter une migration : créer `000N_description.sql` (ne jamais modifier une migration déjà appliquée).
- Une base issue de l'ancien boilerplate est détectée et refusée avec un message explicite : la recréer (`docker compose down -v`).

Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes` (likes et passes), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Contraintes `CHECK` (âge, genres, longueurs, coordonnées), clés étrangères en cascade depuis `users`, unicités (`swipes (swiper, cible)`, `matches (low, high)` avec `low < high`, un seul signalement ouvert par paire), index sur tous les chemins chauds (découverte géographique, messages par conversation, notifications par utilisateur…).

## Données de démonstration

```bash
cd services/api
DATABASE_URL=... UPLOAD_DIR=./data/uploads go run ./cmd/seed               # 12 profils autour de SEED_LAT/SEED_LNG (Paris par défaut)
DATABASE_URL=... go run ./cmd/seed -like ton.email@exemple.fr               # tous les profils de démo te likent déjà → like en retour = match
DATABASE_URL=... go run ./cmd/seed -purge                                    # tout supprimer
```

Garde-fous : comptes marqués `users.is_demo`, e-mails en `@demo.lueur.invalid`, bio préfixée « Profil de démonstration », photos abstraites générées (aucun visage), commande refusée si `APP_ENV=production`, et profils de démo **jamais proposés** par la découverte en production. Mot de passe par défaut : `DemoLueur2026`.

## Lancer l'application mobile

```bash
cd apps/mobile
npm ci
cp .env.example .env        # EXPO_PUBLIC_API_URL
npx expo start              # Expo Go / émulateurs
npm run start:phone         # téléphone physique : détecte l'IP LAN et la passe à l'app
npm run web                 # version web (développement, E2E)
```

Rappels réseau :
- Sur un téléphone physique, n'utilisez jamais `localhost` : `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`.
- Émulateur Android : `http://10.0.2.2:18080` (valeur par défaut en développement).
- Vérifier qu'aucun service local n'occupe le port : `docker compose ps`, `netstat -ano | findstr :18080` (Windows) ou `lsof -i :18080`, puis `curl http://<IP_LAN>:18080/health`.
- Pour la version web, ajouter son origine à `ALLOWED_ORIGINS` (par défaut `http://localhost:8081`).

## Tests et qualité

```bash
scripts/qa.sh               # Linux/macOS : gofmt, vet, tests Go, build, npm ci, tsc, ESLint, Jest, smoke API
scripts/qa-lite.ps1         # Windows : équivalent
```

Détail :

| Commande | Contenu |
| --- | --- |
| `cd services/api && go test ./...` | Unitaires (validation profil, âge, arrondis, classement, JWT, rate limit, traitement d'images, stockage, URLs signées, hub temps réel…) + **intégration** si `DATABASE_URL_TEST` est défini : auth complet, récupération, onboarding, photos, découverte, likes, match, likes simultanés, chat, permissions, blocage, signalement, suppression de compte, WebSocket. |
| `cd apps/mobile && npx tsc --noEmit` | Typecheck strict |
| `npm run lint` | ESLint (config Expo, `any` interdit) |
| `npm test` | Jest : validation, formats, client HTTP (refresh concurrent, déconnexion, erreurs) |
| `npm run e2e:api` | Smoke API du parcours critique contre une API qui tourne (`E2E_API_URL`) |
| `npm run e2e:web` | E2E UI Playwright : inscription → onboarding complet → découverte → like → match → chat temps réel → lu → activité → signalement → suppression du compte. Nécessite l'API et `npm run web`. Captures dans `e2e/web/screenshots/`. |

Les tests d'intégration Go ont besoin d'une base dédiée :

```bash
createdb lueur_test
DATABASE_URL_TEST=postgres://localhost:5432/lueur_test?sslmode=disable go test ./...
```

La CI (`.github/workflows/ci.yml`) exécute tout cela, y compris les tests d'intégration contre Postgres, le build Docker, le smoke API et l'E2E web.

## Build et déploiement

- **API** : `docker build -t lueur-api services/api` (binaire statique, utilisateur non-root, contient aussi `./seed`). À déployer derrière un reverse proxy TLS (Caddy, Nginx, load balancer) ; renseigner `TRUSTED_PROXIES`, `APP_ENV=production`, `ALLOWED_ORIGINS` et le SMTP. Monter un volume persistant sur `UPLOAD_DIR` et le sauvegarder avec la base.
- **Une seule instance API** pour l'instant (voir limites) ; Postgres managé recommandé.
- **Mobile** : builds natifs avec EAS (`npx eas build -p ios|android`) en définissant `EXPO_PUBLIC_API_URL` sur l'URL HTTPS publique. Les permissions sont déclarées dans `app.json` (position « pendant l'utilisation » uniquement, position approximative sur Android, accès à la photothèque).
- **Web** : `npm run build:web` produit `dist/` (statique).

## API

Toutes les réponses d'erreur ont la forme `{"error": "message", "code": "code_stable"}` ; l'app traduit les codes en français.

| Méthode & chemin | Rôle |
| --- | --- |
| `GET /health` | Santé (API + base) |
| `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Session (access 15 min + refresh rotatif 30 jours) |
| `POST /auth/password/forgot`, `/auth/password/reset` | Récupération par code (réponse identique que le compte existe ou non) |
| `GET /me`, `DELETE /me` | Compte ; suppression avec mot de passe |
| `GET/PUT /me/profile`, `PUT /me/location`, `PUT /me/privacy`, `PUT /me/preferences`, `PUT /me/interests`, `GET /interests` | Profil et réglages |
| `GET /me/photos`, `POST /me/photos`, `PUT /me/photos/order`, `PUT /me/photos/:id`, `DELETE /me/photos/:id` | Photos (multipart, champ `photo`) |
| `GET /media/photos/:id?exp=&sig=` | Fichier photo via URL signée expirante |
| `GET /profiles/:userId` | Profil public (si visible et non bloqué) |
| `GET /discovery?limit=` | Profils suivants à découvrir |
| `POST /swipes` `{targetUserId, decision: like|pass}` | Like / passe → `matched` + match éventuel |
| `DELETE /matches/:id` | Annuler un match |
| `GET /conversations?before=`, `GET /conversations/:id/messages?before=`, `POST /conversations/:id/messages`, `POST /conversations/:id/read`, `DELETE /conversations/:id` | Chat (pagination par curseur) |
| `GET /notifications`, `GET /notifications/unread-count`, `POST /notifications/:id/read`, `POST /notifications/read-all` | Notifications |
| `GET/POST /blocks`, `DELETE /blocks/:userId`, `POST /reports` | Sécurité |
| `POST /realtime/ticket`, `GET /realtime?ticket=` | WebSocket (ticket d'une minute) |

Événements temps réel : `message.created`, `conversation.read`, `match.created`, `match.removed`, `notification.created`.

## Décisions techniques et produit

- **Conserver le socle existant** (Go/Gin/pgx/Expo/React Query) qui était sain ; supprimer ce qui contredit le produit (facturation Stripe, organisations multi-tenant) ou ne sert pas une app de rencontre (posts, commentaires, votes, annuaire d'utilisateurs exposant les e-mails, fichiers à clé fournie par le client). Tamagui, inutilisé, a été retiré.
- **Schéma repensé et migrations suivies** : l'ancien système rejouait tous les fichiers à chaque démarrage.
- **Matching** : une décision (like/passe) par paire, définitive ; la paire est verrouillée (`pg_advisory_xact_lock`) pendant l'enregistrement, ce qui supprime la course où deux likes simultanés se manqueraient. Les règles d'éligibilité (blocage, visibilité, compatibilité mutuelle genre/âge, modération) sont revérifiées sous ce verrou, que le blocage prend aussi : un like ne peut jamais « passer » un blocage concurrent, ni viser par identifiant un profil que la découverte n'aurait pas proposé. Le classement est isolé derrière une interface `Ranker` (pondérations transparentes) pour pouvoir le remplacer plus tard.
- **Découverte mutuelle** : on ne propose que des personnes dont les préférences (genre, âge, distance) sont compatibles **dans les deux sens** ; pré-filtre géographique indexé (boîte englobante) puis distance exacte.
- **Localisation minimale** : lue uniquement à la demande, précision basse, arrondie à 0,01° (~1 km) sur l'appareil **et** sur le serveur ; seule une distance arrondie (jamais < 2 km affiché) est montrée, et peut être masquée. Le filtre de distance applique un plancher de 5 km (rayon minimal 5 km) et les changements de position sont limités (6/min), pour empêcher la triangulation.
- **Photos** : type détecté par signature (jamais l'extension ou le type déclaré), dimensions bornées (anti « decompression bomb »), réencodage JPEG qui supprime toutes les métadonnées (dont le GPS), orientation EXIF appliquée, fichiers servis uniquement via URLs signées HMAC qui expirent (stables par fenêtre d'1 h pour le cache, valides 1 à 2 h ; un remplacement change l'identifiant de la photo).
- **Sessions** : chaque requête vérifie que la session existe encore → déconnexion, changement de mot de passe et suppression de compte prennent effet immédiatement. Le WebSocket utilise un ticket JWT d'une minute (audience dédiée), jamais le token d'accès dans l'URL ; chaque socket est liée à sa session, fermée à la déconnexion et revérifiée toutes les ~4 minutes.
- **Modération** : signaler bloque automatiquement ; un profil signalé par 3 comptes distincts (créés depuis plus de 24 h, pour éviter les comptes jetables) sort de la découverte en attendant une revue (table `reports`, statut `open/reviewed/dismissed`).
- **Âge** : 18 ans minimum vérifié côté client et serveur (contrainte métier + bornes de préférences ≥ 18).
- **Suppression de compte** : suppression physique en cascade (profil, photos + fichiers, swipes, matchs, conversations, messages, notifications, sessions) ; les signalements sont conservés anonymisés pour la modération.
- **Notifications** : in-app + temps réel. Le push natif (APNs/FCM) n'est pas activé car il exige des identifiants de store/EAS propres au projet.
- **Interface** : identité originale « Lueur » — thème sombre chaud (encre, braise, crème), typographie système, swipe fait maison (Animated + PanResponder, aucune dépendance), boutons équivalents pour l'accessibilité, libellés d'accessibilité, états vides/chargement/erreur partout. Langue : français.

## Limites connues et pistes

- **Mise à l'échelle horizontale** : le hub WebSocket et le rate limiting sont en mémoire (une instance). Pour plusieurs réplicas : bus partagé (Postgres `LISTEN/NOTIFY` ou Redis) derrière l'interface `realtime.Publisher`, et limiteur partagé.
- **Stockage** : disque local derrière l'interface `storage.Store` ; un backend S3 compatible peut être ajouté sans toucher aux features.
- **Push natif** : à brancher (Expo Push / FCM / APNs) en enregistrant les tokens d'appareil.
- **Outil de modération** : les signalements sont stockés et exploitables en SQL ; une console d'administration reste à construire.
- **Vérification d'e-mail** et **vérification de photo** : non implémentées.
- **Distance & triangulation** : arrondi ~1 km, plancher de 5 km et limitation des changements de position réduisent fortement le risque ; un attaquant patient avec plusieurs comptes peut encore estimer une zone. Masquer la distance (`Confidentialité`) protège davantage.
- **Antiméridien** : le pré-filtre géographique ne gère pas le passage à ±180° de longitude.
