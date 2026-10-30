# Alba — application de rencontre 100 % gratuite

Alba est une application de rencontre complète (mobile + web + API) construite sur le boilerplate `go-react-saas`.
Elle est **entièrement gratuite** : aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de likes, aucune publicité.

Fonctionnalités :

- Création de compte, connexion, récupération de mot de passe, changement de mot de passe, suppression de compte.
- Onboarding guidé : prénom, âge (18+ vérifié côté serveur), genre, préférences de rencontre, localisation approximative, 1 à 6 photos, bio, centres d'intérêt.
- Découverte : cartes avec photos, profil complet, **like / passer** (glisser la carte ou boutons), jamais deux fois le même profil.
- **Match** quand l'intérêt est réciproque, avec notification.
- **Messagerie temps réel** : heure d'envoi, lu / non lu, liste des conversations, masquage local, désappariement.
- Notifications (matchs, nouveaux messages).
- Confidentialité : profil en pause, distance masquable, position jamais exacte.
- **Blocage** et **signalement** ; liste des personnes bloquées.
- Thème clair / sombre automatique, états de chargement, états vides, erreurs et retours d'action.

## Stack technique

| Couche | Technologies |
| --- | --- |
| Mobile / web | React Native (Expo SDK 54), TypeScript strict, React Navigation, TanStack Query |
| API | Go 1.24, Gin, pgx/pgxpool, JWT HS256, bcrypt — pas d'ORM |
| Base de données | PostgreSQL 16 |
| Infra | Docker, Docker Compose |

## Architecture

```text
apps/mobile/            Application Expo (iOS, Android, web)
  src/api/              Client HTTP unique + endpoints typés (aucun fetch dans l'UI)
  src/state/            Session, store de découverte, QueryClient
  src/hooks/            Requêtes, temps réel (long polling), localisation
  src/screens/          Écrans ; src/components/ : composants métier ; src/ui/ : design system
  e2e/                  Smoke API + parcours navigateur (Playwright)
services/api/
  cmd/api/              Serveur HTTP (un seul serveur Gin)
  cmd/seed/             Données de démonstration (dev uniquement)
  internal/platform/    config, db (+ migrations), middleware, realtime, storage, imaging, mail, signedurl…
  internal/features/    auth, account, profiles, photos, matching, chat, safety, notifications
docs/                   ARCHITECTURE.md, API.md
```

Chaque feature suit `handler -> service -> repository` (+ `model.go`, `routes.go`). Le SQL n'existe que dans les repositories.
Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) et [docs/API.md](docs/API.md).

## Démarrage rapide (Docker)

Prérequis : Docker, Docker Compose, Node 20.19+.

```bash
cp .env.example .env
# Renseignez JWT_SECRET (>= 32 caractères aléatoires) :
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env   # ou éditez .env à la main
docker compose up --build -d
curl http://localhost:18080/health                    # {"status":"ok"}
```

Les migrations s'appliquent automatiquement au démarrage de l'API. Les photos sont stockées dans le volume Docker `uploads`.

Données de démonstration (12 profils fictifs, e-mails `demo.<prénom>@demo.invalid`, mot de passe `DemoPass1234!`) :

```bash
docker compose exec api ./seed          # créer
docker compose exec api ./seed -purge   # supprimer
```

> Le seed refuse de s'exécuter avec `APP_ENV=production`. Les comptes de démo sont marqués `users.is_demo` et utilisent le domaine réservé `.invalid`.

## Lancer l'application

```bash
cd apps/mobile
npm ci
```

- **Navigateur** : `EXPO_PUBLIC_API_URL=http://localhost:18080 npm run web` (ajoutez l'origine affichée à `ALLOWED_ORIGINS` côté API, ex. `http://localhost:8081`).
- **Émulateur** : `npm run android` / `npm run ios` (l'URL de l'API par défaut est `10.0.2.2:18080` sur Android, `localhost:18080` sur iOS).
- **Téléphone physique** : ne jamais utiliser `localhost`. Lancez `npm run start:phone` : le script détecte l'IP du réseau local et définit `EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080`.
  Vérifiez avant : `docker compose ps` et `curl http://<IP_LAN>:18080/health`. Si le port 18080 est occupé, changez le port hôte dans `docker-compose.yml`, `apps/mobile/src/config.ts`, `apps/mobile/scripts/start-phone.js` et ce README.

## Variables d'environnement

Voir [.env.example](.env.example). Résumé :

| Variable | Obligatoire | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256, 32 caractères minimum, pas de valeur factice |
| `DATABASE_URL` | oui | URL PostgreSQL |
| `APP_ENV` | non | `development` (défaut), `production`, `test`. En production, SMTP est obligatoire |
| `PORT` | non | Port HTTP (défaut `8080`) |
| `UPLOAD_DIR` | non | Dossier des photos (défaut `./data/uploads`) — montez un volume persistant |
| `APP_BASE_URL` | non | URL publique, utilisée dans les e-mails de réinitialisation |
| `ALLOWED_ORIGINS` | non | Origines CORS autorisées (build web) |
| `TRUSTED_PROXIES` | non | IP/CIDR des proxys de confiance (pour l'IP cliente du rate limiting) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | prod | Envoi des e-mails de réinitialisation. En développement sans SMTP, l'e-mail est écrit dans le log de l'API |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API vue par l'appli. **Aucun secret** ne doit être mis côté client |

## Base de données et migrations

Les migrations SQL sont embarquées dans le binaire (`services/api/internal/platform/db/migrations`) et appliquées dans l'ordre, une seule fois chacune (table `schema_migrations`, verrou consultatif pour les déploiements simultanés).
Schéma : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`.

> Migration depuis l'ancien boilerplate SaaS : le schéma a été reconstruit. Recréez la base (`docker compose down -v`) ; il n'existe pas de chemin de migration des anciennes tables (billing, organisations, posts…).

## Tests et qualité

```bash
# API (PostgreSQL requis pour les tests d'intégration)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'
gofmt -l . && go vet ./... && go test ./... && go build ./cmd/api

# Mobile
cd apps/mobile
npm run typecheck && npm run lint && npm run format:check && npm test && npm run build:web

# Smoke et parcours complet (API lancée sur :18080, build web servi sur :8081)
npm run e2e:api       # inscription -> profil -> photo -> découverte -> like -> match -> chat -> notifications
npm run e2e:web       # même parcours dans un vrai navigateur (nécessite `npm i --no-save playwright` + `npx playwright install chromium`)
```

Sans `DATABASE_URL_TEST`/`DATABASE_URL`, les tests d'intégration sont ignorés (ils ne le sont jamais en CI).
Les tests d'intégration couvrent : authentification, réinitialisation de mot de passe, profils, photos (type, taille, signature), préférences mutuelles, likes idempotents, match unique sous concurrence, conversations et permissions, blocage, signalement, suppression de compte, rate limiting.

## Build et déploiement

- API : `docker build -t alba-api services/api` (image Alpine, utilisateur non-root, binaires `api` et `seed`).
- Web : `EXPO_PUBLIC_API_URL=https://api.example.com npm run build:web` produit `apps/mobile/dist` (hébergement statique).
- Mobile natif : `eas build` (Expo Application Services) avec `EXPO_PUBLIC_API_URL` défini pour le build.
- Production : `APP_ENV=production`, SMTP configuré, `JWT_SECRET` aléatoire, HTTPS devant l'API (reverse proxy ; renseignez `TRUSTED_PROXIES`), base PostgreSQL gérée avec sauvegardes, volume persistant (ou stockage objet) pour `UPLOAD_DIR`.

## Décisions techniques

- **Stack conservée** : Go/Gin/pgx/JWT et Expo ont été gardés. Supprimés car hors produit : billing Stripe, organisations multi-tenant, posts/commentaires/votes, Tamagui (inutilisé). Les migrations ont été repartis de zéro (une seule baseline propre).
- **Temps réel par long polling** (`GET /events`) : fonctionne partout (React Native inclus) sans dépendance ni WebSocket. Les événements ne portent aucun contenu privé : le client rafraîchit via les endpoints autorisés. Le hub est en mémoire (une instance) ; pour plusieurs instances, remplacez-le derrière la même interface par Postgres `LISTEN/NOTIFY` ou Redis.
- **Photos privées** : fichiers ré-encodés en JPEG (EXIF/GPS supprimés, taille limitée à 1600 px, 8 Mo max, JPEG/PNG vérifiés par leur contenu), stockés hors de la racine web sous des noms aléatoires, servis via des **URL signées HMAC** valables 1 à 2 h, émises uniquement aux personnes autorisées à voir le profil.
- **Localisation minimale** : seules des coordonnées arrondies (~1 km) sont stockées ; les autres membres ne voient qu'une distance approximative (1 km, puis paliers de 5 km). La distance est calculée en SQL (haversine), sans PostGIS.
- **Règles côté serveur** : âge ≥ 18, préférences mutuelles (genre, âge, distance), exclusion des profils déjà traités et des blocages, autorisation des swipes, accès aux conversations. Le client n'est jamais cru.
- **Match** : paire ordonnée unique `(user_a < user_b)`, verrou consultatif par paire → un seul match même si deux likes arrivent simultanément.
- **Classement de la découverte** : personnes qui vous ont déjà liké, puis centres d'intérêt communs, proximité, activité récente (requête unique, extensible : voir `candidateQuery`).
- **Blocage** : supprime le match, la conversation et ses messages, et masque mutuellement les profils. **Signalement** : conservé pour la modération, même après suppression des comptes (références mises à `NULL`).
- **Suppression de compte** : confirmation par mot de passe ; cascade SQL sur toutes les données, puis suppression des fichiers photos.
- **Connexion** : jeton d'accès 15 min + jeton de rafraîchissement rotatif (30 j, haché en base). Changer le mot de passe révoque toutes les sessions. Un jeton d'accès reste valide jusqu'à son expiration (≤ 15 min) après une déconnexion.
- **Limites de débit** en mémoire par instance (connexion/inscription 10/min/IP, messages 60/min, likes 240/min, signalements 10/h…).
- **Langue** : interface en français ; le code et les identifiants restent en anglais.

## Limites connues

- Les notifications sont dans l'application (liste + badges + temps réel). Les notifications push (APNs/FCM) ne sont pas incluses : elles nécessitent des comptes développeur Apple/Google.
- Pas d'outil d'administration pour traiter la table `reports` (statuts `open` / `reviewing` / `closed`) : à consulter en SQL pour l'instant.
- Photos sur disque local : pour plusieurs instances API, implémentez `storage.Store` sur un stockage objet.
- Pas de vérification d'e-mail à l'inscription.
