# Alma — application de rencontre 100 % gratuite

Alma est une application mobile de rencontre (iOS / Android) et son API. Deux personnes peuvent créer un compte,
composer leur profil, découvrir des profils compatibles, liker ou passer, matcher en cas d'intérêt réciproque, puis
discuter en temps réel.

**Gratuite, vraiment :** aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de likes,
aucune publicité. Aucune fonctionnalité n'est verrouillée.

## Fonctionnalités

| Domaine | Détails |
| --- | --- |
| Compte | inscription, connexion, sessions avec rotation du refresh token, récupération de mot de passe par e-mail, suppression de compte (données et fichiers effacés) |
| Profil | prénom, âge (18+ vérifié côté serveur), genre, bio, localisation approximative, centres d'intérêt |
| Photos | jusqu'à 6, upload multipart, recompression JPEG 1080 px (métadonnées EXIF/GPS supprimées), photo principale, ordre, remplacement, suppression ; fichiers servis uniquement via l'API authentifiée |
| Découverte | pile de profils (swipe **ou** boutons), carte détaillée, jamais de profil déjà traité, filtres genre / âge / distance appliqués de façon réciproque, classement extensible |
| Matching | like / pass définitifs, match sur like réciproque (transaction + verrou : aucun doublon, même en cas de likes simultanés), annulation de match |
| Chat | conversation créée au match, temps réel (WebSocket), heure d'envoi, non-lus, accusé de lecture, suppression locale d'une conversation, accès strictement limité aux participants |
| Notifications | matchs et messages (liste, compteur de non-lus, marquage comme lu), événements temps réel |
| Sécurité & modération | blocage (bilatéral, supprime le match), signalement (6 motifs), rate limiting, profil masquable, distance arrondie |

## Stack

- **Mobile** : Expo SDK 54 / React Native 0.81, TypeScript strict, React Navigation, TanStack Query, `expo-image`,
  `expo-location`, `expo-image-picker`. Thème clair/sombre automatique.
- **API** : Go 1.24, Gin, PostgreSQL 16 (`pgxpool`, SQL explicite, pas d'ORM), JWT HS256, bcrypt, `gorilla/websocket`,
  `x/image` (redimensionnement).
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Architecture

```text
apps/mobile/            application Expo (src/{api,components,navigation,realtime,screens,state,theme,lib})
services/api/
  cmd/api/              point d'entrée HTTP (+ tests d'intégration)
  cmd/seed/             comptes de démonstration (@demo.invalid)
  internal/platform/    config, db (migrations), middleware, logger, realtime (hub WS), geo, mailer, httpx
  internal/features/    auth · profiles · photos · matching · chat · moderation · notifications
infra/docker-compose.yml
docs/ARCHITECTURE.md    schéma de données, règles métier, décisions
```

Chaque feature suit `handler -> service -> repository` (`handler.go`, `service.go`, `repository.go`, `model.go`,
`routes.go`). Voir [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Démarrage rapide (Docker)

```bash
cp .env.example .env
# renseigner JWT_SECRET (>= 32 caractères aléatoires) :  openssl rand -base64 48
cd infra
docker compose --env-file ../.env up -d --build
curl http://localhost:18080/health                 # {"status":"ok"}

# (optionnel) comptes de démonstration :
docker compose exec -e SEED_CONFIRM=yes api ./seed
```

L'API écoute sur **`http://<hôte>:18080`** (le port 8080 du conteneur est publié en 18080 pour éviter les conflits
locaux). Les migrations s'appliquent automatiquement au démarrage et sont tracées dans `schema_migrations`.

### Application mobile

```bash
cd apps/mobile
npm ci
# Téléphone physique : JAMAIS localhost. Utiliser l'IP LAN de la machine qui héberge l'API.
EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npx expo start
# ou : npm run start:phone   (détecte l'IP LAN automatiquement)
```

Émulateurs : sans variable, Android utilise `http://10.0.2.2:18080` et iOS `http://localhost:18080`. Ouvrir ensuite
l'app dans Expo Go. Compte de démonstration après seed : `camille@demo.invalid` / `Demo-Passw0rd`.

> Les clés de production (Apple/Google) et le build natif (EAS) ne sont pas configurés ici ; `app.json` fournit les
> identifiants `app.alma.mobile` à adapter.

## Développement local sans Docker

Prérequis : Go 1.24+, Node 22+, PostgreSQL 16+.

```bash
createdb app && createdb app_test
export DATABASE_URL="postgres://USER@localhost:5432/app?sslmode=disable"
export JWT_SECRET="$(openssl rand -base64 48)"
cd services/api
go run ./cmd/api                      # migrations automatiques, API sur :8080
SEED_CONFIRM=yes go run ./cmd/seed    # comptes de démonstration (refuse APP_ENV=production)
```

## Variables d'environnement

Voir [`.env.example`](.env.example). Résumé :

| Variable | Obligatoire | Rôle |
| --- | --- | --- |
| `JWT_SECRET` | oui | secret HS256 (≥ 32 caractères, valeurs d'exemple refusées) |
| `DATABASE_URL` | oui | connexion PostgreSQL |
| `PORT` | non (8080) | port HTTP |
| `APP_ENV` | non | `production` exige SMTP et désactive le seed |
| `UPLOAD_DIR` | non | dossier des photos (volume Docker `uploads_data`) |
| `ALLOWED_ORIGINS` | non | origines CORS / WebSocket navigateur |
| `TRUSTED_PROXIES` | non | proxys de confiance (IP réelle pour le rate limiting) |
| `SMTP_HOST/PORT/USER/PASSWORD/FROM`, `APP_BASE_URL` | prod | envoi du code de récupération de mot de passe |
| `EXPO_PUBLIC_API_URL` | mobile | URL de l'API (IP LAN sur téléphone) |

Sans `SMTP_HOST` (développement), le code de réinitialisation est écrit dans les logs de l'API.

## Tests et qualité

```bash
# API (les tests d'intégration exigent une base dont le nom finit par _test : elle est VIDÉE à chaque test)
cd services/api
export TEST_DATABASE_URL="postgres://USER@localhost:5432/app_test?sslmode=disable"
gofmt -l .            # doit être vide
go vet ./... && go test -race ./... && go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm run typecheck && npm run lint && npm run format:check && npm test && npm run bundle

# Parcours complet contre une API en marche : inscription -> profil -> photo -> découverte -> like -> match -> chat
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run e2e:smoke
```

Sans `TEST_DATABASE_URL`, les tests d'intégration sont explicitement ignorés (skip). Les tests UI sur appareil
(Maestro/Detox) ne sont pas fournis : le parcours critique est couvert par `e2e:smoke` côté API + temps réel.

## Déploiement

1. Construire l'image : `docker build -t alma-api services/api` (binaires `api` et `seed`, utilisateur non-root).
2. Fournir `DATABASE_URL`, `JWT_SECRET`, `APP_ENV=production`, SMTP, `ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, et monter un
   volume persistant sur `UPLOAD_DIR` (`/data/uploads`).
3. Placer l'API derrière un proxy TLS (HTTPS/WSS obligatoires en production ; `usesCleartextTraffic` dans `app.json`
   ne sert qu'au développement local et doit être retiré pour un build de production).
4. Limites connues d'une instance unique : le hub WebSocket et le rate limiting sont en mémoire. Pour monter à
   plusieurs instances, relayer les événements via PostgreSQL `LISTEN/NOTIFY` ou Redis et externaliser les compteurs.
5. Modération : les signalements sont stockés dans la table `reports` (statut `open`) ; aucune interface d'administration
   n'est fournie (`SELECT * FROM reports WHERE status='open'`).

## Principales décisions

- **Un seul schéma SQL propre** : les anciens modules du boilerplate (posts, commentaires, Stripe, organisations
  multi-tenant) ont été retirés, incompatibles avec un produit gratuit centré sur le matching.
- **Swipe + boutons** : le glissement est un confort, jamais l'unique moyen d'agir (accessibilité).
- **Décisions de swipe définitives** : un like/pass ne se rejoue pas (409) ; un match annulé ne se recrée pas.
- **Match atomique** : verrou advisory par paire + contrainte `user_a < user_b` + `UNIQUE` → un seul match possible.
- **Vie privée de la localisation** : coordonnées arrondies à ~1 km avant stockage, jamais renvoyées aux autres ; la
  distance est affichée par paliers de 5 km et masquable.
- **Photos privées** : stockées hors racine web, re-encodées, servies par l'API après contrôle (propriétaire, profil
  visible ou match, absence de blocage).
- **WebSocket sans token dans l'URL** : authentification par premier message ; l'état fait toujours foi côté REST
  (le temps réel n'est qu'un accélérateur, resynchronisation à la reconnexion).
- **Recommandation** : classement SQL isolé (`feedQuery` dans `matching/repository.go`) : likes reçus, intérêts
  communs, proximité, activité. Point d'extension pour un futur algorithme.
- **Âge** : 18 ans minimum imposé par le serveur (contrainte SQL + validation) ; la date de naissance n'est pas
  modifiable après création côté application.
- **Mot de passe oublié** : code à usage unique (hash SHA-256 en base, 1 h), réponse identique que le compte existe ou
  non, toutes les sessions révoquées après réinitialisation.

## Licence

Voir le dépôt.
