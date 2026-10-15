# Alma — application de rencontre 100 % gratuite

Alma est une application de rencontre complète : création de compte, profil et photos, découverte, likes, matchs,
messagerie en temps réel, notifications, blocage, signalement et suppression de compte.

**Gratuite pour de vrai** : aucun abonnement, aucun paywall, aucun « boost » payant, aucune limite artificielle de likes,
aucune publicité. Les seuls plafonds du code sont des garde-fous anti-robots, très au-dessus d'un usage humain.

Le dépôt est un monorepo : une API Go + PostgreSQL et une application mobile React Native (Expo).

---

## Fonctionnalités

| Domaine | Ce que fait l'application |
| --- | --- |
| Compte | Inscription, connexion, sessions avec rotation du refresh token, mot de passe oublié (code à 6 chiffres par e-mail), changement de mot de passe, suppression définitive du compte |
| Profil | Prénom, âge (18+ vérifié côté serveur), genre, présentation, centres d'intérêt, jusqu'à 6 photos (ordre, photo principale, remplacement, suppression) |
| Préférences | Genres recherchés, tranche d'âge, distance maximale, mode discret (profil masqué), affichage de la distance activable |
| Découverte | Pile de cartes avec swipe (ou boutons), galerie de photos, profil complet, centres d'intérêt en commun |
| Matching | Like / passer. Like réciproque = match. Jamais de doublon, un profil traité ne revient plus. Les critères sont **mutuels** (genre, âge, distance des deux côtés) |
| Messagerie | Conversation ouverte uniquement après un match, texte, heure, non lus, « Lu », temps réel (WebSocket), masquage local, annulation du match |
| Notifications | Match et nouveaux messages, centre de notifications, compteurs, bannière en temps réel |
| Sécurité | Blocage (invisible dans les deux sens), signalement avec motif, limitation de débit, autorisations vérifiées côté serveur |
| Confidentialité | Position arrondie à ~1 km, jamais renvoyée ; seule une distance approximative (tranches de 5 km) est visible, désactivable |
| Interface | Thème clair/sombre automatique, états de chargement/vide/erreur, accessibilité (rôles, libellés, zones tactiles ≥ 44 px) |

## Pile technique

- **API** : Go 1.24, Gin, PostgreSQL 16, `pgxpool`, JWT HS256, bcrypt, SQL explicite (pas d'ORM), WebSocket (`gorilla/websocket`), traitement d'image (`golang.org/x/image`).
- **Mobile** : Expo SDK 54 / React Native 0.81, TypeScript strict, React Navigation, TanStack Query, `expo-image`, `expo-image-picker`, `expo-location`.
- **Infra** : Docker, Docker Compose, GitHub Actions.

## Architecture

```text
.
├─ apps/mobile/                 Application Expo (TypeScript)
│  ├─ App.tsx
│  ├─ src/api/                  Client HTTP typé (refresh de session automatique), endpoints, types
│  ├─ src/hooks/                Auth, requêtes (TanStack Query), temps réel (WebSocket)
│  ├─ src/navigation/           Navigation (auth → onboarding → onglets)
│  ├─ src/screens/              Écrans
│  ├─ src/components/           Composants métier (pile de swipe, galerie, gestion des photos…)
│  ├─ src/shared/               Design system (ui), mise en page (layout), retours (feedback)
│  ├─ src/theme/                Palette claire/sombre et jetons de design
│  ├─ src/lib/                  Logique pure testée (dates, validation, pile, photos)
│  ├─ src/i18n/                 Textes français et messages d'erreur
│  ├─ scripts/                  Smoke API, fixtures et lanceur Maestro, démarrage sur téléphone
│  └─ e2e/maestro/              Parcours UI critique
├─ services/api/                API Go
│  ├─ cmd/api/                  Point d'entrée + tests d'intégration
│  ├─ cmd/seed/                 Données de démonstration (jamais en production)
│  └─ internal/
│     ├─ platform/              config, db (+ migrations), middleware, httpx, errors, logger, realtime, blob, mailer
│     └─ features/              auth, users, files, matching, chat, notifications, safety
├─ docs/ARCHITECTURE.md         Détails d'architecture
├─ docker-compose.yml           PostgreSQL + API (port hôte 18080)
└─ .github/workflows/           CI (tests, lint, build, parcours complet sur Docker)
```

Chaque module de l'API suit `handler → service → repository` (`handler.go`, `service.go`, `repository.go`, `model.go`,
`routes.go`) : handlers fins, règles métier dans les services, SQL uniquement dans les repositories.

| Module | Rôle |
| --- | --- |
| `auth` | Inscription, connexion, refresh, déconnexion, récupération, suppression de compte |
| `users` | Profil, préférences, centres d'intérêt, localisation, confidentialité, cartes publiques |
| `files` | Photos : validation, ré-encodage, miniatures, URLs signées |
| `matching` | Découverte, likes, matchs, annulation de match |
| `chat` | Conversations, messages, état de lecture |
| `notifications` | Création côté serveur, lecture, compteurs |
| `safety` | Blocages et signalements |

## Démarrage rapide (Docker)

```bash
cp .env.example .env
# Générez un secret fort et collez-le dans JWT_SECRET :
openssl rand -hex 32

docker compose up --build -d
curl http://localhost:18080/health           # {"status":"ok"}
```

Données de démonstration (comptes `*@demo.invalid`, mot de passe `DemoPass123!`) :

```bash
docker compose exec api ./seed               # crée 14 profils fictifs autour de Paris
docker compose exec api ./seed --reset       # les supprime (comme une suppression de compte)
```

Application mobile :

```bash
cd apps/mobile
npm ci
# Téléphone physique : JAMAIS localhost. Utilisez l'IP locale de votre ordinateur.
EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start
# ou, détecte l'IP automatiquement :
npm run start:phone
```

Émulateur Android : l'application utilise `http://10.0.2.2:18080` par défaut ; simulateur iOS : `http://localhost:18080`.

> Si le port 18080 est déjà pris, changez le port hôte dans `docker-compose.yml`, puis `EXPO_PUBLIC_API_URL`,
> `apps/mobile/scripts/start-phone.js` et ce README.

## Développement local sans Docker

```bash
# 1. PostgreSQL (exemple avec Docker pour la base seule)
docker run -d --name alma-db -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=app -p 5432:5432 postgres:16-alpine

# 2. API
cd services/api
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -hex 32)"
export PORT=18080
go run ./cmd/api          # applique les migrations au démarrage
go run ./cmd/seed         # facultatif : données de démonstration

# 3. Mobile
cd ../../apps/mobile && npm ci && EXPO_PUBLIC_API_URL=http://<IP_LAN>:18080 npx expo start
```

En développement, sans `SMTP_HOST`, le code de récupération de mot de passe est écrit dans les logs de l'API
(`mail_dev`). En production (`APP_ENV=production`) il n'est jamais journalisé : configurez le SMTP.

## Variables d'environnement

Voir `.env.example`. Les secrets ne sont jamais dans le code ni dans l'application mobile.

| Variable | Obligatoire | Défaut | Rôle |
| --- | --- | --- | --- |
| `JWT_SECRET` | oui | — | Signe les jetons d'accès et les URLs de photos. 32 caractères minimum ; les valeurs « placeholder » sont refusées |
| `DATABASE_URL` | oui | — | Chaîne de connexion PostgreSQL |
| `PORT` | non | `8080` | Port d'écoute de l'API |
| `APP_ENV` | non | `development` | `production` désactive le mailer de développement et la commande `seed` |
| `UPLOAD_DIR` | non | `./data/uploads` | Stockage des photos (volume Docker `uploads_data`) |
| `ALLOWED_ORIGINS` | non | vide | Origines navigateur autorisées (CORS, WebSocket). Inutile pour les apps natives |
| `TRUSTED_PROXIES` | non | vide | IP/CIDR du reverse proxy dont `X-Forwarded-For` est fiable (limitation de débit par IP) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | non | — | E-mails de récupération de compte |
| `EXPO_PUBLIC_API_URL` | mobile | `localhost:18080` / `10.0.2.2:18080` en dev | URL de l'API vue **depuis le téléphone** |

## Base de données et migrations

PostgreSQL, SQL explicite, migrations embarquées dans le binaire et appliquées au démarrage, **une seule fois chacune**
(table `schema_migrations`, verrou consultatif pour les déploiements concurrents). Les migrations `001`–`013` sont
l'historique du boilerplate d'origine ; `014`–`019` créent le modèle de l'application de rencontre.

| Table | Contenu |
| --- | --- |
| `users`, `sessions`, `password_resets` | Comptes, sessions (refresh token haché), codes de récupération hachés (5 essais, 15 min) |
| `profiles`, `preferences` | Profil public et critères de recherche (contraintes `CHECK` : âge ≥ 18, genres, distance 1–500 km) |
| `interests`, `user_interests` | Catalogue de centres d'intérêt (données de référence) et choix des utilisateurs |
| `photos` | Position dense 0–5 (la position 0 est la photo principale), clé de stockage opaque |
| `likes` | Une décision par couple ordonné (`like` ou `pass`), clé primaire `(from, to)` |
| `matches` | Un seul enregistrement par couple (`user_a < user_b`, `UNIQUE`) |
| `conversations`, `conversation_participants`, `messages` | Une conversation par match, état de lecture et masquage par participant |
| `blocks`, `reports` | Blocages et signalements (le rapporteur est anonymisé si son compte disparaît) |
| `notifications` | Notifications serveur, dédoublonnées par clé tant qu'elles sont non lues |

Toutes les clés étrangères sont en `ON DELETE CASCADE` : supprimer un utilisateur efface en une requête profil,
photos (puis fichiers), likes, matchs, conversations, messages et notifications. Index : `users.email`,
`messages (conversation_id, created_at)`, `notifications (user_id, created_at)`, `photos (user_id, position)`,
`profiles` partiel sur les profils visibles, `likes (to_user_id, kind, created_at)`, etc.

## Aperçu de l'API

JSON, erreurs au format `{"error": "...", "code": "..."}` (codes stables, traduits côté application). Authentification :
`Authorization: Bearer <accessToken>` (15 min) ; `POST /auth/refresh` fait tourner le refresh token (30 jours). Un jeton
est refusé dès que sa session est révoquée (déconnexion, changement de mot de passe, suppression).

| Méthode | Route | Description |
| --- | --- | --- |
| `GET` | `/health` | Santé de l'API et de la base |
| `POST` | `/auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` | Session |
| `POST` | `/auth/password/forgot` · `/auth/password/reset` | Récupération par code e-mail |
| `GET` · `DELETE` | `/me` | Compte courant · suppression (mot de passe requis) |
| `PUT` | `/me/password` | Changer le mot de passe (déconnecte les autres appareils) |
| `GET` · `PUT` | `/me/profile` | Profil et progression de l'onboarding |
| `PUT` | `/me/preferences` · `/me/location` · `DELETE /me/location` · `PATCH /me/privacy` | Préférences, position, confidentialité |
| `GET` | `/interests` | Catalogue des centres d'intérêt |
| `GET` `POST` | `/photos` | Lister · ajouter (multipart, champ `file`) |
| `PUT` `DELETE` `POST` | `/photos/order` · `/photos/:id` · `/photos/:id/main` | Ordre, remplacement, suppression, photo principale |
| `GET` | `/photos/:id/file?size=full\|thumb&exp&sig` | Fichier image, autorisé par une URL signée |
| `GET` | `/discover?limit&exclude=id,id` | Profils à découvrir |
| `GET` | `/profiles/:userId` | Profil complet d'une personne visible |
| `POST` | `/swipes` `{userId, action: like\|pass}` | Décision ; renvoie `matched` |
| `DELETE` | `/matches/:userId` | Annuler un match |
| `GET` | `/chats` | Conversations (dernier message, non lus) |
| `GET` `POST` | `/chats/:id/messages` | Historique paginé (`before`) · envoi |
| `POST` `DELETE` | `/chats/:id/read` · `/chats/:id` | Marquer lu · masquer pour soi |
| `GET` `POST` | `/notifications` · `/notifications/read-all` · `/notifications/:id/read` | Notifications |
| `GET` `POST` `DELETE` | `/blocks` · `/blocks/:userId` | Utilisateurs bloqués |
| `POST` | `/reports` `{userId, reason, details}` | Signalement |
| `POST` · `GET` | `/ws/ticket` · `/ws?ticket=` | Ticket à usage unique puis WebSocket |

Événements temps réel (WebSocket) : `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`.

## Tests, qualité et build

```bash
# API (les tests d'intégration ont besoin de PostgreSQL ; sans variable ils sont ignorés)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'
gofmt -l .            # doit ne rien afficher
go vet ./...
go test -count=1 ./...
go build ./cmd/api ./cmd/seed

# Mobile
cd apps/mobile
npm ci
npm run typecheck && npm run lint && npm run format:check && npm test
npx expo export --platform android      # vérifie que l'application se bundle

# Parcours complet sur une API en marche : inscription → profil → découverte → like → match → chat → suppression
npm run e2e:smoke
```

Ce que couvrent les tests d'intégration (`services/api/cmd/api`, PostgreSQL réel) : authentification et jetons invalides
(expirés, mauvais algorithme, `alg: none`), déconnexion immédiate, récupération de mot de passe, validation de profil
(âge, genre, date de naissance verrouillée), position arrondie et jamais exposée, cycle de vie des photos (types dangereux,
polyglotte, taille, signature), règles de découverte **mutuelles**, likes et passes sans doublon, **match unique même avec
deux likes simultanés**, annulation de match, messagerie (non lus, « Lu », pagination, masquage), accès refusé aux
non-participants, notifications dédoublonnées, WebSocket (tickets à usage unique), blocage, signalement, suppression
complète du compte, limitation de débit.

Test UI de bout en bout (Maestro, nécessite un téléphone ou un émulateur) :

```bash
cd apps/mobile
npm run e2e:ui:setup        # crée deux comptes complets, dont un qui a déjà liké le premier
npm run e2e:ui:run          # connexion → découverte → like → match → message → onglet Matchs
```

Intégration continue (`.github/workflows/ci.yml`) : format, `go vet`, tests unitaires et d'intégration (service
PostgreSQL), build ; typecheck, lint, format, tests et bundle du mobile ; construction de l'image Docker et parcours
complet sur la pile Docker.

## Déploiement

1. Construire l'image : `docker build -t alma-api services/api` (utilisateur non-root, `HEALTHCHECK` sur `/health`).
2. Fournir `DATABASE_URL`, `JWT_SECRET` (aléatoire, 32+ caractères), `APP_ENV=production`, un volume persistant monté sur
   `UPLOAD_DIR`, et le SMTP pour la récupération de compte.
3. Placer l'API derrière un reverse proxy HTTPS et renseigner `TRUSTED_PROXIES` avec son adresse.
4. L'application s'exécute en **une seule instance** (limitation de débit et temps réel en mémoire, photos sur disque).
   Pour passer à l'échelle : stockage objet derrière l'interface `blob.Store`, Redis ou `LISTEN/NOTIFY` derrière
   `realtime.Hub.Publish`, limiteur partagé.
5. Mobile : `EXPO_PUBLIC_API_URL` doit pointer vers l'URL HTTPS publique de l'API au moment du build (EAS Build).

## Décisions techniques

- **On construit sur l'existant** : Go/Gin/pgx, sessions JWT + refresh, Expo, TanStack Query conservés ; `tamagui`
  (utilisé uniquement pour des jetons) supprimé ; le design system a été refait en thème clair/sombre.
- **Critères mutuels** : une personne n'apparaît que si *chacun* correspond aux préférences de l'autre. Les règles
  d'éligibilité sont écrites une seule fois en SQL (`matching/repository.go`) et réutilisées par la découverte, la
  consultation d'un profil et le like : le client ne peut pas contourner les règles.
- **Un match, une conversation** : créés dans la même transaction que le second like, sous un verrou consultatif du
  couple. Pas d'endpoint pour écrire à un inconnu.
- **Les décisions sont définitives** : pour revenir en arrière, on annule le match. Un profil passé ne réapparaît pas
  (exigence du produit) ; l'ordre de proposition est un unique `ORDER BY` (personnes qui vous ont liké, intérêts communs,
  proximité, activité récente), facile à remplacer par un score.
- **Position** : arrondie à 2 décimales (~1 km) avant stockage, jamais renvoyée ; distances affichées par tranches de
  5 km ; facultative (sans position, pas de filtre de distance). Demandée uniquement quand l'utilisateur appuie sur le bouton.
- **Âge** : 18–99 ans contrôlé côté serveur ; la date de naissance n'est plus modifiable une fois saisie. L'âge est déclaratif.
- **Photos** : validation par contenu (JPEG/PNG/WebP), garde anti « bombe de décompression » avant décodage, orientation
  EXIF appliquée, **ré-encodage systématique** (supprime EXIF/GPS et contenu parasite), redimensionnement 1080 px +
  miniature 360 px. Servies par URL signée HMAC (valide 24–48 h, stable pour le cache) ; un utilisateur bloqué ne reçoit
  plus de nouvelles URLs.
- **Jetons** : accès 15 min, vérification de session à chaque requête (déconnexion immédiate), refresh à rotation.
- **Temps réel** : WebSocket authentifié par un ticket à usage unique de 30 s (pas de JWT dans l'URL), reconnexion
  avec backoff et resynchronisation côté application. Les envois passent par REST (validation et limitation uniques).
- **Suppression de compte** : demande le mot de passe, supprime tout en cascade puis les fichiers, ferme les WebSocket.
- **Rate limiting** : mémoire, par IP (authentification) ou par utilisateur (swipes, messages, uploads, signalements),
  seuils très larges.
- **Notifications** : dans l'application (centre + bannière temps réel). Les notifications push système (APNs/FCM)
  demandent un compte développeur et ne sont pas incluses.

## Limites connues et pistes

- Adresse e-mail non vérifiée à l'inscription (la récupération repose sur l'accès à la boîte).
- Pas de vérification d'identité ni de modération automatique ; les signalements sont stockés (`reports.status`) mais
  l'outil de traitement reste à construire.
- Notifications push système, instance unique, algorithme de recommandation à enrichir.
- Les modules hérités du boilerplate `posts`, `comments` et `billing` (ainsi que les tables correspondantes des
  migrations `002`, `003`, `010`, `013`) ne sont plus montés ni utilisés. Leur suppression n'a pas pu être effectuée
  dans l'environnement automatisé de cette livraison : voir `CHANGELOG.md`.

## Contribuer

Voir `CONTRIBUTING.md` et `AGENTS.md`.
