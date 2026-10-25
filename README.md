# Lueur — une application de rencontre 100 % gratuite

Lueur est une application de rencontre mobile (iOS / Android via Expo) adossée à une API Go.
**Aucun abonnement, aucun paywall, aucun boost payant, aucune limite artificielle de likes, aucune publicité.**
Toutes les fonctionnalités sont accessibles à tous les utilisateurs.

Parcours : inscription → profil (photos, bio, centres d'intérêt, préférences) → découverte (swipe ou boutons) →
like / passer → match réciproque → discussion en temps réel → notifications → blocage / signalement → suppression du compte.

> Le dépôt est issu du boilerplate `go-react-saas`. Les modules génériques (posts, commentaires, organisations, facturation Stripe)
> ont été retirés au profit du domaine « rencontre ». Voir [CHANGELOG.md](CHANGELOG.md).

## Stack

| Couche | Technologies |
|---|---|
| API | Go 1.24, Gin, PostgreSQL 16, `pgxpool`, SQL explicite (pas d'ORM), JWT HS256 + refresh tokens rotatifs, bcrypt, WebSocket (`gorilla/websocket`) |
| Mobile | Expo SDK 54, React Native 0.81, TypeScript strict, React Navigation, TanStack Query |
| Infra | Docker, Docker Compose, GitHub Actions |

## Architecture

```text
.
├─ apps/mobile/                 # Application Expo (src/api, src/features/*, src/navigation, src/realtime, src/shared, src/theme)
├─ services/api/
│  ├─ cmd/api/                  # Point d'entrée de l'API (config, migrations, arrêt gracieux)
│  ├─ cmd/seed/                 # Comptes de démonstration (refuse de tourner en production)
│  └─ internal/
│     ├─ app/                   # Câblage (routes, services) + tests d'intégration bout en bout
│     ├─ features/              # handler -> service -> repository (+ model, routes) par domaine
│     │  ├─ auth/  profiles/  photos/  discovery/  matches/  chat/  safety/  notifications/
│     └─ platform/              # config, db (+ migrations SQL embarquées), middleware, httpx, realtime, mailer, logger
├─ docs/API.md                  # Contrat REST + WebSocket
├─ docs/ARCHITECTURE.md         # Décisions techniques et modèle de données
└─ infra/docker-compose.yml     # PostgreSQL + API
```

Règles : handlers fins, aucune requête SQL hors des repositories, un seul pool de connexions partagé, listes paginées par curseur
(keyset), aucun N+1. Détails dans [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation rapide (Docker)

Prérequis : Docker, Node.js 20+ (pour le mobile).

```bash
cp .env.example .env
# renseigner JWT_SECRET :  openssl rand -hex 32
docker compose --env-file .env -f infra/docker-compose.yml up --build -d
curl http://localhost:18080/health        # {"status":"ok"}
```

L'API écoute sur `http://localhost:18080` (port hôte `18080` -> conteneur `8080`). Les migrations sont appliquées au démarrage
(table `schema_migrations`, verrou consultatif : plusieurs instances peuvent démarrer en même temps).

### Données de démonstration (optionnel, jamais en production)

```bash
docker compose --env-file .env -f infra/docker-compose.yml run --rm api ./seed      # 21 comptes @demo.invalid
# ou hors Docker : cd services/api && DATABASE_URL=... go run ./cmd/seed [-reset]
```

Les comptes de démo sont marqués `users.is_demo = true`, utilisent le domaine `demo.invalid` et ne réagissent qu'entre eux.
Connexion : `demo.alex@demo.invalid` ou `demo.sam@demo.invalid`, mot de passe `demo-password-1` (ou `DEMO_PASSWORD`).
Ces deux comptes ont déjà été « likés » par les autres profils de démo compatibles : un like en retour donne un match immédiat.
`-reset` supprime tous les comptes de démo et leurs données.

## Lancer l'application mobile

```bash
cd apps/mobile
cp .env.example .env            # EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080
npm ci
npm start                       # ou: npm run android | npm run ios
```

* Sur un **téléphone physique**, n'utilisez jamais `localhost` : mettez l'IP de votre machine sur le réseau local
  (`EXPO_PUBLIC_API_URL=http://192.168.x.y:18080`). Émulateur Android : `http://10.0.2.2:18080` (valeur par défaut en dev).
* Vérifier qu'aucun autre service n'occupe le port : `docker compose -f infra/docker-compose.yml ps` puis
  `curl http://<IP_LAN>:18080/health`. En cas de conflit, changer le port hôte dans `infra/docker-compose.yml` et `EXPO_PUBLIC_API_URL`.
* L'envoi de photos demande l'accès à la galerie, la géolocalisation (facultative) l'accès à la position approximative.

## Variables d'environnement

| Variable | Obligatoire | Description |
|---|---|---|
| `JWT_SECRET` | oui | Secret HS256 (≥ 32 caractères, valeurs « placeholder » refusées). `openssl rand -hex 32` |
| `DATABASE_URL` | oui (hors Compose) | URL PostgreSQL |
| `PORT` | non | Port de l'API (défaut `8080`) |
| `APP_ENV` | non | `development` (défaut) ou `production` (exige `SMTP_HOST`, désactive le seed) |
| `ALLOWED_ORIGINS` | non | Origines navigateur autorisées (CORS / WebSocket). Les apps natives n'en ont pas besoin |
| `TRUSTED_PROXIES` | non | Proxys autorisés à fournir `X-Forwarded-For` (rate limiting par IP). Vide si l'API est exposée directement |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | prod | Envoi du code de réinitialisation du mot de passe. Sans SMTP (dev), le code est écrit dans les logs de l'API |
| `DEMO_PASSWORD` | non | Mot de passe des comptes de démo |
| `EXPO_PUBLIC_API_URL` | mobile | URL publique de l'API (jamais de secret côté mobile) |

## Tests et qualité

```bash
# API (PostgreSQL requis pour les tests d'intégration ; chaque test crée puis supprime son propre schéma)
cd services/api
export TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
gofmt -l . && go vet ./... && go test -race ./... && go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test

# Parcours complet contre une API qui tourne (inscription -> profil -> like -> match -> chat -> blocage)
MOBILE_E2E_API_URL=http://localhost:18080 node apps/mobile/scripts/e2e-smoke.js
```

Sans `TEST_DATABASE_URL`/`DATABASE_URL`, les tests d'intégration sont ignorés (`t.Skip`). `make test` lance tout (API + mobile).
La CI (`.github/workflows/ci.yml`) exécute ces mêmes commandes, plus le smoke test sur la pile Docker.

Ce qui est couvert par les tests d'intégration Go : authentification (rotation du refresh token, révocation de session, jetons forgés,
réinitialisation par code, limites de tentatives, rate limiting), profil et validations (âge ≥ 18), localisation grossière, photos
(types dangereux, bombes de décompression, polyglottes, droits d'accès), découverte et préférences mutuelles, likes / passes / matchs
sans doublons (dont likes simultanés), chat (pagination, non-lus, lecture, droits), blocage, signalement, notifications,
suppression de compte (aucune donnée résiduelle) et temps réel WebSocket.

## Build et déploiement

```bash
docker build -t lueur-api services/api      # image minimale, utilisateur non-root, HEALTHCHECK /health
```

Production : fournir `APP_ENV=production`, `JWT_SECRET`, `DATABASE_URL`, `SMTP_*`, mettre l'API derrière un proxy TLS
(renseigner `TRUSTED_PROXIES` avec l'IP du proxy). Le mobile se construit avec EAS (`eas build`) en définissant
`EXPO_PUBLIC_API_URL` sur l'URL HTTPS publique. Le temps réel et le rate limiting sont **en mémoire par instance** : au-delà d'une
instance, prévoir un bus partagé (voir ci-dessous).

## Principales décisions techniques

* **Auth** : JWT HS256 de 15 min (expiration obligatoire, algorithme verrouillé) + refresh token opaque haché (SHA-256), rotatif, stocké en
  base. Chaque requête vérifie que la session est active : déconnexion, changement/réinitialisation de mot de passe et suppression de compte
  prennent effet immédiatement. Mots de passe bcrypt (8–72 octets), comparaison factice pour ne pas révéler les comptes existants.
* **Âge** : date de naissance obligatoire, 18 ans minimum vérifiés côté serveur ; seul l'âge est exposé.
* **Matching** : un swipe est définitif (like ou pass, jamais deux fois). Un verrou consultatif par paire sérialise deux likes simultanés :
  exactement un match est créé. Une seule ligne `matches` par paire (`user_a < user_b`). Les cibles inéligibles (bloquées, hors préférences,
  inexistantes) répondent 404.
* **Découverte** : les préférences doivent être **mutuelles** (genre, tranche d'âge, distance). Exclusion des profils déjà traités, bloqués,
  masqués ou sans photo. Le classement est une expression SQL isolée (`feedScore`) : intérêts communs, personnes qui vous ont liké,
  activité récente — facile à remplacer par un autre algorithme.
* **Localisation** : coordonnées arrondies à ~1 km avant stockage, jamais renvoyées par l'API ; distance affichée arrondie au multiple
  de 5 km supérieur, masquable. La position est facultative ; sans position, seuls les filtres de distance « partout » s'appliquent.
* **Photos** : stockées en PostgreSQL (`bytea`, 6 max, taille limitée) pour des sauvegardes et suppressions atomiques. Chaque upload est décodé
  (JPEG/PNG uniquement, dimensions plafonnées), orienté selon l'EXIF, redimensionné (1080 px / miniature 320 px) et ré-encodé en JPEG :
  métadonnées, GPS et données ajoutées sont supprimés. Accès uniquement authentifié et soumis aux règles de visibilité (blocage, profil masqué).
* **Chat** : une conversation par match ; REST fait foi, WebSocket (`/ws`, authentification par première trame pour ne jamais mettre le
  jeton dans une URL) pousse messages, accusés de lecture, matchs et notifications. « Supprimer » une conversation n'efface que
  l'historique de l'utilisateur.
* **Sécurité** : limites de débit (auth, swipes, messages, signalements, uploads), limite de taille des corps, en-têtes de sécurité,
  `X-Forwarded-For` ignoré sauf proxys déclarés, erreurs homogènes `{error, code}`, journaux sans secrets.
* **Suppression de compte** : `DELETE /me` (mot de passe requis) supprime l'utilisateur ; toutes les tables liées sont en `ON DELETE CASCADE`
  (profil, photos, swipes, matchs, messages, blocages, signalements, notifications, sessions).
* **Passage à l'échelle de la découverte** : la requête de découverte évalue les candidats éligibles puis trie par score ; c'est très bien jusqu'à des dizaines de milliers de profils. Au-delà, pré-filtrer par zone géographique (boîte englobante indexée ou PostGIS) et/ou matérialiser des files de candidats par utilisateur — le contrat d'API ne change pas.
* **Mono-instance** : hub WebSocket et rate limiting sont en mémoire. Passer à plusieurs instances = brancher le `realtime.Publisher` sur
  PostgreSQL `LISTEN/NOTIFY` (ou Redis) et un limiteur partagé.
* **Migrations** : fichiers SQL embarqués, appliqués une fois chacun dans une transaction (`schema_migrations`). Le schéma initial
  (`001_init.sql`) remplace celui du boilerplate : une base issue de l'ancien boilerplate doit être recréée.
* **Modération** : les signalements sont stockés (`reports`, statut `open`) pour traitement par l'équipe ; il n'y a pas encore d'interface d'administration.

## Limites connues

* L'application mobile a été vérifiée par typecheck, lint, 81 tests Jest, un bundle Android (`expo export`) et le parcours API
  `e2e-smoke.js` (36 contrôles). Elle n'a **pas** été exécutée sur un appareil ou un émulateur : gestes de swipe, sélecteur de photos,
  géolocalisation et flux Maestro (`apps/mobile/e2e/maestro`) restent à valider manuellement avant une mise en production.
* Le service WebSocket mobile est testé avec un faux socket ; le endpoint `/ws` réel est couvert côté API.
* Pas d'interface d'administration pour traiter les signalements.

## Licence et contribution

Voir [CONTRIBUTING.md](CONTRIBUTING.md) et [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
