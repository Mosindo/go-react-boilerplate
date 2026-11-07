# Aube — application de rencontre 100 % gratuite

Aube est une application de rencontre complète (mobile iOS/Android + web) : profils, photos, découverte par swipe, matchs, chat temps réel, notifications, blocage/signalement et suppression de compte.

**Gratuit pour de vrai** : aucun abonnement, paywall, boost payant, limite artificielle de likes ni publicité. Toutes les fonctionnalités sont ouvertes à tout le monde (y compris « qui m'a liké »).

## Fonctionnalités

* Compte, connexion, récupération de mot de passe (code par e-mail), changement de mot de passe, suppression complète du compte.
* Onboarding : prénom, âge (18+ vérifié côté serveur), genre, ville / position approximative, bio, centres d'intérêt, préférences (genres recherchés, tranche d'âge, distance), 1 à 6 photos.
* Découverte : carte avec galerie de photos, like / passer (swipe ou boutons), profil complet, jamais le même profil deux fois.
* Matching : like réciproque ⇒ match (jamais de doublon, même en cas de likes simultanés), liste des likes reçus.
* Chat : conversation après match, temps réel (WebSocket), heure d'envoi, lu / non lu, suppression locale d'une conversation, annulation de match.
* Notifications in-app (nouveaux likes et matchs) + badges.
* Sécurité : blocage, signalement (motif + détails), rate limiting, uploads contrôlés, URLs de photos signées et temporaires.

## Stack

| Couche | Choix |
|---|---|
| API | Go 1.24, Gin, pgx/pgxpool, PostgreSQL 16, JWT HS256, bcrypt, gorilla/websocket — sans ORM |
| Mobile / web | Expo 54, React Native 0.81, TypeScript strict, React Navigation, TanStack Query, react-native-web |
| Qualité | `go test` (unitaires + intégration sur vraie base), Vitest, ESLint, Prettier, Playwright (E2E navigateur) |
| Infra | Docker, Docker Compose, GitHub Actions |

Détails d'architecture : [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Démarrage rapide (Docker)

```bash
cp .env.example .env
# renseigner JWT_SECRET :  openssl rand -hex 32
docker compose up --build -d          # Postgres + API sur http://localhost:18080
curl http://localhost:18080/health     # {"status":"ok"}
docker compose exec api ./seed         # (optionnel) 12 comptes de démo @demo.invalid
```

Les migrations s'appliquent automatiquement au démarrage de l'API (table `schema_migrations`, une fois par fichier, dans une transaction).

### Lancer l'application

```bash
cd apps/mobile
npm ci
# Web (le plus simple pour essayer) :
npm run web
# Téléphone physique : ne JAMAIS utiliser localhost, mettre l'IP LAN de la machine
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npx expo start
```

Si vous testez le build web depuis un autre origin que l'API, ajoutez-le à `ALLOWED_ORIGINS` (ex. `http://localhost:8081`).
Comptes de démo : `demo01@demo.invalid` … `demo12@demo.invalid`, mot de passe `demo-password-123` (ou `SEED_PASSWORD`). `go run ./cmd/seed -reset` les supprime. Le seed refuse de s'exécuter avec `APP_ENV=production`.

## Développement local sans Docker

```bash
# Postgres 16 local, deux bases : app et app_test
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/app?sslmode=disable
export JWT_SECRET=$(openssl rand -hex 32)
cd services/api && go run ./cmd/api          # écoute sur :8080 (PORT)
```

## Variables d'environnement

Toutes sont documentées dans [`.env.example`](.env.example).

| Variable | Rôle |
|---|---|
| `JWT_SECRET` | **Obligatoire**, ≥ 32 caractères, non-placeholder (l'API refuse de démarrer sinon) |
| `DATABASE_URL` | **Obligatoire**, URL PostgreSQL |
| `PORT` | Port HTTP (défaut `8080`) |
| `UPLOAD_DIR` | Dossier des photos (volume persistant en production) |
| `ALLOWED_ORIGINS` | Origines navigateur autorisées (CORS + WebSocket) pour le build web |
| `TRUSTED_PROXIES` | IP/CIDR des reverse proxies dont on accepte `X-Forwarded-For` (vide = aucun) |
| `SMTP_*`, `APP_BASE_URL` | E-mail de récupération de mot de passe ; sans `SMTP_HOST` le message est écrit dans les logs (dev uniquement) |
| `EXPO_PUBLIC_API_URL` | URL publique de l'API embarquée dans l'app (aucun secret !) |

## Tests et qualité

```bash
# Backend (les tests d'intégration utilisent DATABASE_URL_TEST ou DATABASE_URL, et sont ignorés sans base)
cd services/api
gofmt -l . ; go vet ./... ; go test ./... ; go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm run typecheck && npm run lint && npm run format:check && npm test && npm run build:web

# E2E navigateur (API lancée, ALLOWED_ORIGINS=http://localhost:19006, build web fait)
npx playwright install chromium            # ou E2E_CHROMIUM_PATH=/chemin/vers/chrome
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
npm run e2e
```

Le parcours E2E couvre : inscription → onboarding (profil, intérêts, préférences, photo) → découverte → like → match (temps réel côté autre utilisateur) → conversation → lu → suppression du compte.
Les tests d'intégration Go couvrent auth (rotation/révocation de session, reset de mot de passe), validation des profils et photos, découverte et filtres, likes/matchs (dont la course de likes simultanés), chat et permissions, WebSocket, blocage, signalement, suppression de compte, rate limiting.

## Build et déploiement

* **API** : `docker build -t aube-api services/api` (image Alpine non-root, `UPLOAD_DIR=/data/uploads` à monter en volume). Fournir `DATABASE_URL`, `JWT_SECRET`, et derrière un proxy TLS : `TRUSTED_PROXIES`, `ALLOWED_ORIGINS`. Une seule instance de l'API est supposée pour le temps réel (hub en mémoire) ; voir `docs/ARCHITECTURE.md` pour passer à plusieurs instances.
* **Web** : `EXPO_PUBLIC_API_URL=https://api.exemple.com npm run build:web` produit `apps/mobile/dist`, à servir en statique (HTTPS).
* **iOS / Android** : builds via EAS (`eas build`) avec `EXPO_PUBLIC_API_URL` pointant vers l'API en HTTPS.
* Sauvegardez la base **et** le volume des photos.

## Décisions produit et techniques

* **Pivot du boilerplate** : suppression de la facturation Stripe, du multi-tenant (organisations), des posts/commentaires et de Tamagui, incompatibles avec un produit gratuit et plus simple. Conservés : Go/Gin/pgx, JWT + sessions rotatives, middleware, Expo/React Query.
* **Âge** : 18 ans minimum, calculé côté serveur à partir de la date de naissance ; seul l'âge est exposé.
* **Genres** : femme / homme / non-binaire ; on choisit les genres recherchés (un ou plusieurs). La compatibilité doit être **mutuelle** (orientation, tranche d'âge, distance) pour apparaître en découverte.
* **Décision finale** : un like ou un « passer » est définitif (clé primaire sur la paire) ; réessayer renvoie `409`. Un profil traité ne revient jamais, même après annulation du match.
* **Localisation** : facultative, arrondie à ~1 km avant stockage, jamais renvoyée ; distance affichée approximative et désactivable. Sans position, le filtre de distance est ignoré pour l'utilisateur.
* **Photos** : disque local derrière une interface minimale (`platform/storage`), fichiers non devinables, URLs HMAC d'une heure. Pour du S3, implémenter la même interface (`Put/Read/Delete`).
* **Temps réel** : WebSocket authentifié par première trame ; notifications push natives (APNs/FCM) hors périmètre — les notifications sont in-app.
* **Notifications** : « like » et « match » ; les messages non lus sont signalés par les badges de conversation (pas de doublon dans le fil).
* **Suppression de conversation** : « locale » (masque l'historique pour soi ; réapparaît à un nouveau message). « Annuler le match » supprime la conversation pour les deux.
* **Blocage** : retire le match et la conversation, masque mutuellement les profils, réversible depuis Réglages (le match n'est pas restauré).
* **Modération** : les signalements sont stockés (`reports`, statut `open`) ; l'outil d'administration n'est pas inclus (requête SQL ou futur back-office).
* **Rate limiting** : en mémoire, par processus (auth par IP ; likes, messages, uploads, signalements par utilisateur). Placez un limiteur partagé en frontal si vous scalez.
* **Mot de passe oublié** : code à usage unique (1 h) envoyé par e-mail, réponse identique que le compte existe ou non, toutes les sessions sont révoquées après réinitialisation.

## Licence et contribution

Voir [`CONTRIBUTING.md`](CONTRIBUTING.md) et [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).
