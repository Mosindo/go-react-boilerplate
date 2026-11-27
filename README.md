# Lueur — application de rencontre 100 % gratuite

**Lueur** est une application de rencontre moderne, mobile-first, **entièrement gratuite** : pas d'abonnement, pas de paywall, pas de « boost » payant, pas de likes limités, pas de publicité.

Deux personnes peuvent : créer un compte → créer leur profil (photos, infos, centres d'intérêt, préférences) → découvrir des profils compatibles → liker ou passer → obtenir un **match** quand l'intérêt est réciproque → **discuter en temps réel** → recevoir des notifications → gérer leur confidentialité → **bloquer / signaler** → **supprimer leur compte**.

Le dépôt contient :

- `services/api` — API Go (Gin, PostgreSQL via pgxpool, sans ORM), WebSocket temps réel, stockage des photos.
- `apps/mobile` — application React Native (Expo SDK 54, TypeScript) pour iOS, Android et Web.

---

## Sommaire

1. [Fonctionnalités](#fonctionnalités)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Installation et lancement local](#installation-et-lancement-local)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données et migrations](#base-de-données-et-migrations)
7. [Données de démonstration (seed)](#données-de-démonstration-seed)
8. [Tests et qualité](#tests-et-qualité)
9. [Build et déploiement](#build-et-déploiement)
10. [API](#api)
11. [Sécurité et vie privée](#sécurité-et-vie-privée)
12. [Principales décisions techniques et produit](#principales-décisions-techniques-et-produit)

---

## Fonctionnalités

| Domaine | Ce qui est implémenté |
| --- | --- |
| Compte | Inscription, connexion, session avec refresh token rotatif, déconnexion, **récupération de compte par code e-mail**, suppression définitive du compte |
| Onboarding | Assistant en 5 étapes : prénom / date de naissance (18+ vérifié côté serveur) / genre / ville, préférences (genres, tranche d'âge, distance), position approximative, photos, bio et centres d'intérêt |
| Profil | Jusqu'à 6 photos (ajout, remplacement, suppression, photo principale/ordre), bio 500 caractères, 10 centres d'intérêt parmi un catalogue, ville libre |
| Découverte | Pile de cartes avec **swipe** (et boutons accessibles), carrousel de photos, distance arrondie, centres d'intérêt communs, profil complet |
| Matching | Like / pass, match si réciproque, **aucun doublon** (contraintes + verrou par paire), un profil traité ne réapparaît jamais, classement extensible |
| Chat | Conversation après match, temps réel (WebSocket), heure d'envoi, lu/non lu (« Vu »), pagination, envoi optimiste et idempotent, retry, annulation du match |
| Notifications | Notifications in-app (nouveau match) en temps réel, compteur non lu, tout marquer comme lu, badges d'onglets |
| Confidentialité | Mettre son profil en pause, masquer sa distance, partager / mettre à jour / supprimer sa position |
| Sécurité | Blocage (bidirectionnel, supprime le match), signalement avec motif et instantané de conversation, masquage automatique après 3 signalements distincts, liste des personnes bloquées |
| UX | Identité visuelle originale, thème clair/sombre (système ou forcé), états de chargement / vides / erreurs, toasts, accessibilité (labels, rôles, tailles de police) |

---

## Stack technique

**API** : Go 1.24, Gin, PostgreSQL 16, pgx/pgxpool, JWT HS256 (`golang-jwt`), bcrypt, `gorilla/websocket` (temps réel), `golang.org/x/image` (décodage WebP, redimensionnement de qualité).

**Mobile** : Expo SDK 54 / React Native 0.81 / React 19, TypeScript strict, React Navigation 7, TanStack Query 5, expo-image (cache disque), expo-image-picker + expo-image-manipulator (compression locale), expo-location (position approximative), expo-secure-store (jetons), expo-linear-gradient, @expo/vector-icons.

**Outillage** : Docker / Docker Compose, GitHub Actions, ESLint (`eslint-config-expo`), Playwright (E2E web).

---

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (iOS / Android / Web)
│  ├─ App.tsx                   Providers : SafeArea, React Query, thème, auth
│  ├─ e2e/critical-flow.mjs     Test E2E Playwright du parcours critique
│  └─ src/
│     ├─ api/                   Client HTTP (refresh auto), endpoints typés, WebSocket
│     ├─ hooks/                 Session, requêtes React Query, synchro temps réel
│     ├─ features/              Écrans par domaine (auth, onboarding, discover,
│     │                         matches, notifications, profile, safety)
│     ├─ navigation/            Navigateurs (auth / onboarding / onglets + pile)
│     ├─ theme/                 Palette clair/sombre, typographie, espacements
│     ├─ ui/                    Composants réutilisables (Button, TextField, Sheet…)
│     └─ utils/                 Formatage FR, stockage sécurisé, sélection de photos
├─ services/api/
│  ├─ cmd/api                   Point d'entrée (composition, routes, limites)
│  ├─ cmd/seed                  Générateur de profils de démonstration
│  └─ internal/
│     ├─ platform/              Transverse : config, db + migrations, httpx,
│     │                         middleware (auth, CORS, rate limit, logs),
│     │                         realtime (hub WebSocket), storage, mailer
│     └─ features/              Domaines : auth, profiles, photos, matching,
│                                chat, safety, notifications
├─ scripts/smoke-api.mjs        Smoke test API multiplateforme (Node 20+)
├─ docker-compose.yml           Postgres + API (+ volume photos)
└─ docs/ARCHITECTURE.md         Détails d'architecture
```

Chaque feature Go suit `handler.go → service.go → repository.go` (+ `model.go`, `routes.go`) : handlers fins, règles métier dans les services, **SQL uniquement dans les repositories**. Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Installation et lancement local

### Prérequis

- Docker (avec Compose) **ou** PostgreSQL 14+ installé localement
- Go 1.24+ (seulement pour lancer l'API hors Docker)
- Node.js 20+ et npm

### 1. Configurer l'environnement

```bash
cp .env.example .env
# Générer un secret fort et le coller dans JWT_SECRET :
openssl rand -base64 48
```

### 2. Lancer PostgreSQL + l'API avec Docker

```bash
docker compose up --build -d
curl http://localhost:18080/health      # {"status":"ok"}
```

Les migrations s'appliquent automatiquement au démarrage. Les photos sont stockées dans le volume Docker `uploads`.

**Sans Docker** (Postgres local) :

```bash
cd services/api
export DATABASE_URL='postgresql://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
export PORT=18080
go run ./cmd/api
```

### 3. (Optionnel) Ajouter des profils de démonstration

```bash
docker compose exec api ./seed -like ton-email@exemple.com
# ou, hors Docker : cd services/api && go run ./cmd/seed -like ton-email@exemple.com
```

### 4. Lancer l'application mobile

```bash
cd apps/mobile
npm ci
cp .env.example .env
npm run start        # QR code Expo Go, simulateurs (i / a) ou web (w)
```

- **Téléphone physique** : `npm run start:phone` (détecte l'IP LAN et la passe dans `EXPO_PUBLIC_API_URL`). N'utilise jamais `localhost` sur un téléphone.
- **Émulateur Android** : `EXPO_PUBLIC_API_URL=http://10.0.2.2:18080`.
- **Web** : `npm run web` (ajoute `http://localhost:8081` à `ALLOWED_ORIGINS` côté API).

Vérifications avant de déclarer l'environnement prêt (voir aussi `AGENTS.md`) : `docker compose ps`, port 18080 libre, `curl http://<IP_LAN>:18080/health`, puis le smoke test : `node scripts/smoke-api.mjs --api http://<IP_LAN>:18080`.

---

## Variables d'environnement

### API (`.env` à la racine, lu par Docker Compose)

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | **oui** | — | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées). Sert aussi à dériver la clé de signature des URLs de photos et le HMAC des codes de réinitialisation. |
| `DATABASE_URL` | oui (hors Compose) | — | Connexion PostgreSQL. |
| `APP_ENV` | non | `development` | `development`, `test` ou `production`. En production : profils démo masqués, seed refusé, codes de réinitialisation jamais journalisés. |
| `PORT` | non | `8080` | Port HTTP de l'API (Compose expose `18080`). |
| `UPLOAD_DIR` | non | `./data/uploads` | Répertoire de stockage des photos. |
| `ALLOWED_ORIGINS` | non | vide | Origines CORS autorisées (build web), séparées par des virgules. |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR des reverse proxies autorisés à fixer `X-Forwarded-For`. |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | — | Envoi des codes de réinitialisation. Sans SMTP : code affiché dans les logs en dev, abandonné en prod. |
| `POSTGRES_PASSWORD` | non | `postgres` | Mot de passe du Postgres fourni par Compose. |
| `SEED_PASSWORD` | non | `DemoLueur2026` | Mot de passe des comptes démo. |

### Mobile (`apps/mobile/.env`)

| Variable | Rôle |
| --- | --- |
| `EXPO_PUBLIC_API_URL` | URL de l'API embarquée dans l'app. **Aucun secret** côté mobile. |

> Les fichiers `.env` sont ignorés par git ; seuls les `.env.example` sont versionnés.

---

## Base de données et migrations

Les migrations SQL sont dans `services/api/internal/platform/db/migrations/`, embarquées dans le binaire et appliquées **une seule fois chacune** au démarrage (table `schema_migrations`, verrou consultatif PostgreSQL pour les démarrages concurrents, une transaction par migration).

Pour ajouter une migration : créer `0NN_description.sql` (numéro suivant), redémarrer l'API.

Tables principales (`014_create_dating_schema.sql`) :

| Table | Contenu | Contraintes notables |
| --- | --- | --- |
| `users` | compte, e-mail unique, hash bcrypt, `is_demo` | — |
| `profiles` | prénom, date de naissance, genre, bio, ville, position **arrondie**, visibilité | CHECK longueurs, genres, couple lat/lng |
| `preferences` | genres recherchés, âge min/max, distance max | 18–99 ans, 1–500 km |
| `photos` | ordre (0 = principale), clé de stockage, dimensions | ≤ 6 positions, `UNIQUE (user_id, position) DEFERRABLE` |
| `interests`, `user_interests` | catalogue (36 entrées) et choix | PK composite |
| `swipes` | like/pass, une ligne par (auteur, cible) | PK (auteur, cible), pas d'auto-like |
| `matches` | paire ordonnée `user_low_id < user_high_id` | `UNIQUE` paire → aucun doublon possible |
| `conversations`, `conversation_participants`, `messages` | conversation liée au match, marqueur de lecture, `client_id` idempotent | cascade à la suppression du match |
| `blocks` | blocages | PK (bloqueur, bloqué) |
| `reports` | motif, détails, **instantané de conversation**, statut | un signalement ouvert par paire |
| `notifications` | notifications in-app | index (user_id, created_at) |
| `password_reset_tokens` | codes hachés (HMAC), tentatives, expiration | — |

Toutes les données d'un utilisateur sont en `ON DELETE CASCADE` : supprimer le compte supprime profil, photos (y compris les fichiers), swipes, matchs, conversations, messages, notifications et sessions. Les signalements restent pour la modération (auteur/cible passés à `NULL`).

Les anciennes tables du boilerplate (`posts`, `comments`, `files`, `votes`, `subscriptions`) ne sont plus utilisées par l'API ; elles sont laissées en place pour ne détruire aucune donnée existante.

---

## Données de démonstration (seed)

`cmd/seed` crée jusqu'à 24 profils **clairement identifiés** :

- e-mails `demoNN@demo.lueur.invalid` (TLD réservé `.invalid`), drapeau `users.is_demo` ;
- badge « Profil de démonstration » dans l'app et bio préfixée ;
- photos = **illustrations abstraites générées** (jamais de photos de vraies personnes) ;
- **refus d'exécution si `APP_ENV=production`**, et l'API ne montre jamais de profils démo en production.

```bash
go run ./cmd/seed [-count 24] [-lat 48.8566 -lng 2.3522] [-like toi@exemple.com] [-reset]
```

`-like` fait liker ton compte par tous les profils démo : un like en retour crée un match immédiat (pratique pour tester le chat). `-reset` supprime d'abord les comptes démo et leurs fichiers.

---

## Tests et qualité

### API

```bash
cd services/api
gofmt -l .                       # doit être vide
go vet ./...
# Tests unitaires + intégration (Postgres requis pour l'intégration) :
DATABASE_URL_TEST='postgresql://postgres:postgres@localhost:5432/app_test?sslmode=disable' go test -race ./...
go build ./cmd/api ./cmd/seed
```

- **Unitaires** : âge/majorité, arrondi de position, distance affichée, préférences, prénom, nettoyage de texte, classement de découverte, messages, e-mail, mot de passe, codes de réinitialisation, pipeline d'images (orientation EXIF, suppression des métadonnées/GPS, bombe de décompression, SVG/scripts refusés), signature d'URL, stockage (path traversal), rate limiting, JWT `alg=none`, pagination.
- **Intégration** (`cmd/api/main_integration_test.go`, via le routeur réel et Postgres) : auth complète + révocation, réinitialisation de mot de passe (usage unique, verrouillage après 5 essais), validation de profil, cycle de vie des photos et URLs signées, découverte (préférences mutuelles de genre et d'âge, distance, pass/like jamais revus), like → match, **likes simultanés = un seul match**, chat (membres uniquement, idempotence, non lus, accusés de lecture, pagination), annulation de match, blocage, signalement + preuve + masquage automatique, confidentialité, notifications, **temps réel WebSocket**, suppression de compte (données + fichiers), rate limiting.
- Sans base de données disponible, les tests d'intégration sont ignorés (skip).

### Mobile

```bash
cd apps/mobile
npm ci
npm run typecheck
npm run lint
```

### E2E (navigateur)

Parcours complet **inscription → onboarding → découverte → like → match → conversation (temps réel + « Vu ») → notifications → signalement/blocage → suppression du compte** :

```bash
# API lancée avec ALLOWED_ORIGINS=http://localhost:8081
cd apps/mobile
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run export:web
npx serve -s dist -l 8081 &
npx playwright install chromium     # une seule fois
E2E_API_URL=http://localhost:18080 E2E_WEB_URL=http://localhost:8081 npm run e2e:web
```

Captures d'écran dans `apps/mobile/.e2e/screenshots/`.

### Smoke test API

```bash
node scripts/smoke-api.mjs --api http://localhost:18080
```

### CI

`.github/workflows/ci.yml` exécute à chaque push : gofmt, vet, tests Go avec `-race` contre Postgres, build ; typecheck, lint et export web/Android/iOS ; smoke test + E2E Playwright ; build Docker Compose + smoke + seed.

---

## Build et déploiement

### API

- Image Docker multi-étapes (`services/api/Dockerfile`), utilisateur non-root, binaires `api` et `seed`, volume `/app/data` pour les photos.
- En production : `APP_ENV=production`, `JWT_SECRET` fort, Postgres managé (`sslmode=require`), `UPLOAD_DIR` sur un volume persistant et sauvegardé, SMTP configuré, `TRUSTED_PROXIES` réglé si derrière un load balancer, HTTPS terminé au proxy (HSTS envoyé si `X-Forwarded-Proto: https`).
- **Une seule instance** recommandée en l'état : le hub WebSocket et le rate limiting sont en mémoire. Pour scaler horizontalement, remplacer `realtime.Publisher` par un relais Postgres LISTEN/NOTIFY ou Redis et partager le rate limiter (interfaces prévues).

### Mobile

- Web : `npm run export:web` → dossier statique `dist/` (penser à `--clear` après un changement d'`EXPO_PUBLIC_API_URL`, déjà inclus dans le script).
- iOS / Android : builds natifs via EAS (`npx eas build`) avec `EXPO_PUBLIC_API_URL` pointant vers l'API HTTPS. Les permissions (photos, position approximative uniquement) sont déclarées dans `app.json`.

---

## API

Toutes les réponses d'erreur ont la forme `{"error": "message lisible", "code": "code_machine"}`. Les listes sont paginées par curseur (`?limit=&cursor=`).

| Méthode | Route | Description |
| --- | --- | --- |
| GET | `/health` | Santé (DB incluse) |
| POST | `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` | Session |
| POST | `/auth/password/forgot`, `/auth/password/reset` | Récupération de compte par code |
| GET / DELETE | `/me` | Compte courant / suppression (mot de passe requis) |
| GET / PUT | `/me/profile` | Profil privé (avec état de complétude) / création-modification |
| PUT | `/me/preferences`, `/me/interests`, `/me/privacy` | Préférences, centres d'intérêt, confidentialité |
| PUT / DELETE | `/me/location` | Position (arrondie côté serveur) |
| GET | `/interests` | Catalogue |
| GET / POST | `/me/photos` | Liste / ajout (multipart `photo`) |
| PUT / DELETE | `/me/photos/:id` | Remplacement / suppression |
| PUT | `/me/photos/order` | Réordonner (la première est la principale) |
| GET | `/media/photos/:id?exp&sig` | Photo via URL signée expirante |
| GET | `/discovery` | Profils à découvrir |
| POST | `/discovery/swipes` | `{targetUserId, action: like\|pass}` |
| GET | `/profiles/:userId` | Profil public (si visible et non bloqué, ou matché) |
| GET | `/matches`, `/matches/:id` | Boîte de réception / un match |
| DELETE | `/matches/:id` | Annuler le match |
| GET / POST | `/matches/:id/messages` | Messages / envoi (`clientId` idempotent) |
| POST | `/matches/:id/read` | Marquer comme lu |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId` | Blocages |
| POST | `/reports` | Signalement (bloque par défaut) |
| GET | `/notifications`, `/notifications/unread-count` | Notifications |
| POST | `/notifications/:id/read`, `/notifications/read-all` | Lecture |
| GET | `/ws` | WebSocket : 1er message `{"type":"auth","token":"…"}` ; événements `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new` |

---

## Sécurité et vie privée

- **Authentification** : JWT HS256 strict (méthode HMAC vérifiée, `alg=HS256` imposé, expiration obligatoire, 15 min) + refresh token opaque rotatif (30 jours, haché en base, rotation atomique ; **le rejeu d'un ancien refresh token révoque la session**). **Chaque requête vérifie que la session est encore active** : déconnexion, réinitialisation de mot de passe et suppression de compte prennent effet immédiatement. Comparaison bcrypt factice pour les e-mails inconnus (pas d'énumération par le temps de réponse).
- **Autorisations côté serveur** : un non-participant reçoit `404` sur une conversation (pas de fuite d'existence) ; profils bloqués, en pause ou masqués invisibles ; aucune route n'expose l'e-mail, la date de naissance exacte ni les coordonnées d'un autre membre.
- **Position** : demandée uniquement sur action explicite, précision « basse », **arrondie à une grille d'~1 km avant stockage**, distance affichée arrondie (≥ 1 km, par pas de 5 km au-delà de 20 km), masquable.
- **Photos** : type détecté sur les octets (JPEG/PNG/WebP uniquement), 10 Mo max, dimensions vérifiées **avant décodage** (anti-bombe), orientation EXIF appliquée puis **ré-encodage JPEG qui supprime toutes les métadonnées (GPS inclus)**, clés aléatoires, servies via **URLs signées HMAC expirantes**, fichiers supprimés avec le compte.
- **Abus** : rate limiting (par IP pour inscription/connexion, **par compte** pour les tentatives de mot de passe, récupération de compte, refresh, uploads, swipes, messages, signalements, global par IP hors médias) — les plafonds sont des garde-fous anti-robot, pas des quotas produit ; `X-Forwarded-For` ignoré sauf proxies de confiance.
- **Entrées** : validation stricte (UUID, pagination, longueurs, caractères de contrôle, énumérations), e-mails normalisés, corps JSON typés.
- **En-têtes** : `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, CORS sur liste blanche, HSTS derrière HTTPS ; timeouts serveur (anti-Slowloris).
- **Jetons côté mobile** : Keychain/Keystore via expo-secure-store (sessionStorage sur web).

---

## Principales décisions techniques et produit

- **Construire sur le boilerplate** : Go/Gin/pgx et Expo étaient sains ; l'architecture feature-first est conservée. Ont été **retirés** car contraires au produit ou dangereux : facturation Stripe (app gratuite), annuaire `/users` (exposait tous les e-mails), posts/commentaires (fil public non modéré contournant les blocages), « files » à métadonnées seules (remplacé par un vrai upload), Tamagui (installé mais inutilisé), scripts PowerShell (remplacés par des scripts Node multiplateformes).
- **Temps réel en WebSocket** avec jeton envoyé dans le premier message (jamais dans l'URL), reconnexion exponentielle, resynchronisation REST au retour, et **polling de secours** si la socket est indisponible.
- **Pass définitif** : un profil passé ne revient pas (simple et prévisible). L'annulation d'un match supprime la conversation pour les deux ; les swipes sont conservés pour que les profils ne réapparaissent pas.
- **Signaler = bloquer par défaut** (désactivable), et la conversation est **copiée dans le signalement** avant suppression pour que la modération garde le contexte. Trois signaleurs distincts **crédibles** en 30 jours (profil complet avec photo et interaction réelle avec la personne signalée) masquent automatiquement le profil en attendant une revue humaine ; un compte masqué ne peut plus découvrir ni écrire. Des comptes jetables ne peuvent donc pas faire disparaître quelqu'un. *Une console de modération n'est pas incluse* : les signalements sont dans la table `reports` (`status = 'open'`).
- **Classement de découverte extensible** : SQL applique toutes les règles dures (préférences mutuelles, blocages, distance, déjà vu) sur un lot de 100 candidats ; un `Ranker` Go (intérêts communs, personnes qui t'ont déjà liké, activité récente, proximité, petite variation stable sur la journée) peut être remplacé sans toucher à la sécurité.
- **Récupération de compte par code** (8 caractères, 30 min, 5 essais, usage unique, toutes les sessions révoquées) : fonctionne sans deep link et sur toutes les plateformes.
- **Genres** : femme, homme, non-binaire ; préférences (genre, âge, distance) appliquées **des deux côtés**, y compris pour un like envoyé directement par identifiant.
- **Position facultative mais réciproque** : sans position, on ne voit (et ne peut liker) que des personnes sans position ou ayant choisi « 500 km », pour qu'on ne puisse jamais contourner le rayon de quelqu'un en masquant sa propre position.
- **Notifications push** (APNs/FCM) non incluses : les notifications sont in-app et temps réel. L'ajout d'Expo Push se branche sur `notifications.Service.Notify`.
- **Organisations** (multi-tenant hérité du boilerplate) conservées : tous les membres sont dans l'organisation par défaut ; l'isolation reste appliquée.

### Limites connues

- Hub temps réel et rate limiting en mémoire : une seule instance d'API (voir « Build et déploiement »).
- Pas de notifications push natives ni de console de modération (points d'extension documentés).
- La pagination de la boîte de réception est triée par activité : une conversation qui reçoit un message pendant qu'on fait défiler peut remonter en tête (elle est alors rafraîchie en temps réel).
- Pas de vérification d'e-mail à l'inscription (la récupération de compte, elle, passe par l'e-mail).

Voir aussi : [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [CHANGELOG.md](CHANGELOG.md), [CONTRIBUTING.md](CONTRIBUTING.md).
