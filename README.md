# Lueur — une application de rencontre 100 % gratuite

Lueur est une application de rencontre complète : création de compte, profil avec photos, découverte par swipe, matchs réciproques, messagerie en temps réel, notifications, blocage, signalement et suppression de compte.

**Pas d'abonnement, pas de paywall, pas de boost payant, pas de likes limités, pas de publicité.** Aucune fonctionnalité essentielle n'est verrouillée (voir [Décisions produit](#décisions-produit)).

Le dépôt est un monorepo : une API Go et un client Expo / React Native (iOS, Android **et navigateur**).

```text
apps/mobile      Client Expo (React Native + TypeScript), aussi servi en web
services/api     API Go (Gin, PostgreSQL, pgx, JWT HS256) + commande de seed
infra/           Variante de docker compose
scripts/         Scripts de QA PowerShell (ils délèguent à des scripts Node)
docs/            Architecture détaillée
```

## Sommaire

1. [Démarrage rapide](#démarrage-rapide)
2. [Stack technique](#stack-technique)
3. [Architecture](#architecture)
4. [Installation détaillée](#installation-détaillée)
5. [Variables d'environnement](#variables-denvironnement)
6. [Base de données, migrations, seed](#base-de-données-migrations-seed)
7. [Lancer le client](#lancer-le-client)
8. [Tests et qualité](#tests-et-qualité)
9. [Build et déploiement](#build-et-déploiement)
10. [Modération](#modération)
11. [Sécurité et vie privée](#sécurité-et-vie-privée)
12. [Décisions produit](#décisions-produit)
13. [Décisions techniques](#décisions-techniques)
14. [Limites connues et pistes](#limites-connues-et-pistes)

## Démarrage rapide

Prérequis : Docker, Node 20+.

```bash
# 1. Configuration : générez un secret JWT
cp .env.example .env
sed -i "s|^JWT_SECRET=.*|JWT_SECRET=$(openssl rand -base64 48)|" .env

# 2. API + PostgreSQL (migrations appliquées au démarrage)
docker compose up --build -d
curl http://localhost:18080/health        # {"status":"ok"}

# 3. (facultatif) comptes de démonstration
docker compose run --rm api ./seed        # demo01@seed.invalid … demo12@seed.invalid / DemoPass123

# 4. Client : ouvre l'application dans le navigateur
cd apps/mobile
cp .env.example .env
npm ci
npm run web
```

Créez un compte dans l'interface, ou connectez-vous avec `demo01@seed.invalid` / `DemoPass123`. `demo02`, `demo04` et `demo06` ont déjà liké `demo01` : likez-les en retour pour voir un match.

Sur téléphone, voir [Lancer le client](#lancer-le-client) (jamais `localhost` depuis un téléphone physique).

## Stack technique

| Couche | Choix |
| --- | --- |
| Client | Expo 54, React Native 0.81, TypeScript strict, React Navigation 6, TanStack Query 5 |
| API | Go 1.24, Gin, pgx/pgxpool (pas d'ORM), JWT HS256, bcrypt, gorilla/websocket |
| Données | PostgreSQL 16 |
| Temps réel | WebSocket (`/ws`), événements poussés par le serveur |
| Infra | Docker, Docker Compose, GitHub Actions |
| Qualité | gofmt, go vet, `go test -race`, ESLint, Prettier, Jest + Testing Library, Playwright |

## Architecture

```text
services/api
├─ cmd/api         Point d'entrée (un seul serveur HTTP, Gin)
├─ cmd/seed        Données de démonstration (refuse APP_ENV=production)
└─ internal
   ├─ platform     config · db (migrations) · middleware (auth JWT, CORS, rate limit…)
   │               logger · mailer · realtime (hub WebSocket) · httpx
   └─ features     auth · profiles · photos · discover · matches · chat · notifications · safety
                   (chacune : handler → service → repository, + model.go et routes.go)
```

Règles : handlers fins, logique métier dans les services, SQL uniquement dans les repositories, préoccupations transverses dans `platform`. Détails et schéma de données dans [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

```text
apps/mobile/src
├─ api            Client HTTP (refresh automatique du token), endpoints, types
├─ hooks          Auth, temps réel (WebSocket), requêtes TanStack Query
├─ features       Éléments métier réutilisables (deck de swipe, photos, sécurité…)
├─ screens        Écrans (aucun accès réseau direct : tout passe par src/api)
├─ navigation     Pile + onglets, thème clair/sombre
└─ shared         Kit d'UI, layouts, retours (toasts, états vides/erreur)
```

## Installation détaillée

Prérequis : Go 1.24+, Node 20+, PostgreSQL 16 (ou Docker pour la base seule).

```bash
# Base de données seule
docker compose up -d postgres

# API
cd services/api
export DATABASE_URL='postgres://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET="$(openssl rand -base64 48)"
go run ./cmd/api          # écoute sur :8080 (PORT pour changer)

# Client
cd apps/mobile && npm ci
```

## Variables d'environnement

Le fichier [`.env.example`](.env.example) liste tout. Aucun secret n'est versionné ni placé dans le client.

| Variable | Obligatoire | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | Secret HS256, **32 caractères minimum**, refusé s'il ressemble à un placeholder. `openssl rand -base64 48` |
| `DATABASE_URL` | oui (hors compose) | URL PostgreSQL. Compose la construit à partir de `POSTGRES_*`. |
| `PORT` | non (`8080`) | Port d'écoute de l'API. |
| `APP_ENV` | non (`development`) | `production` : mode release Gin, SMTP obligatoire. |
| `ALLOWED_ORIGINS` | si client web | Origines navigateur autorisées (CORS et WebSocket), séparées par des virgules. |
| `TRUSTED_PROXIES` | derrière un proxy | IP/CIDR des proxys dont `X-Forwarded-For` est cru. Vide = aucun. Le rate limiting repose sur l'IP cliente. |
| `SMTP_HOST` `SMTP_PORT` `SMTP_USERNAME` `SMTP_PASSWORD` `SMTP_FROM` | production | Envoi des codes de récupération de compte. Sans `SMTP_HOST` (développement), le code est écrit dans les logs de l'API. |
| `POSTGRES_USER` `POSTGRES_PASSWORD` `POSTGRES_DB` | compose | Identifiants de la base du compose (défaut `app`). Changez-les hors développement. |
| `EXPO_PUBLIC_API_URL` | client | URL publique de l'API, intégrée au build du client. |

## Base de données, migrations, seed

- Les migrations SQL (`services/api/internal/platform/db/migrations`) sont **embarquées dans le binaire** et appliquées au démarrage, **une seule fois chacune**, dans l'ordre, sous verrou consultatif (plusieurs instances peuvent démarrer ensemble). La table `schema_migrations` suit l'état.
- Une base créée avant l'introduction du suivi (versions « boilerplate SaaS ») est reconnue : les migrations 001-013 sont considérées comme appliquées, puis `014` (schéma de rencontre) et `015` (suppression des tables SaaS : billing, posts, commentaires, fichiers, organisations) migrent les données. **Les anciennes tables de messages/notifications sont remplacées, leur contenu n'est pas conservé.**
- Tables : `users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `likes`, `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`, `notifications`. Clés étrangères en `ON DELETE CASCADE` depuis `users` : supprimer un compte efface tout. Contraintes d'unicité (un like par couple, un match par couple via `user_a < user_b`), `CHECK` sur les valeurs, index sur les chemins chauds (voir l'architecture).
- **Seed** : `go run ./cmd/seed` (ou `docker compose run --rm api ./seed`) crée 12 comptes `demoNN@seed.invalid` (domaine réservé, jamais livrable) avec photos générées. Idempotent. `go run ./cmd/seed -reset` les supprime. La commande refuse de s'exécuter si `APP_ENV=production`.

## Lancer le client

```bash
cd apps/mobile
cp .env.example .env     # EXPO_PUBLIC_API_URL
npm ci
npm run web              # navigateur
npm start                # Expo : QR code pour Expo Go, ou a / i pour un émulateur
```

`EXPO_PUBLIC_API_URL` selon la cible :

| Cible | Valeur |
| --- | --- |
| Navigateur, simulateur iOS | `http://localhost:18080` |
| Émulateur Android | `http://10.0.2.2:18080` |
| **Téléphone physique** | `http://<IP_LAN>:18080` — **jamais `localhost`** |

`npm run start:phone` détecte l'IP du réseau local et lance Expo avec la bonne URL. Avant de déclarer l'application prête sur téléphone : `docker compose ps`, `curl http://<IP_LAN>:18080/health`, puis le parcours inscription → profil → découverte → match → chat → notifications. En cas de conflit de port, changez le port hôte du compose (`18080:8080`) et alignez `.env`, le fallback de `src/api/client.ts` et les scripts.

Pour le web servi par un autre domaine, ajoutez son origine à `ALLOWED_ORIGINS`.

## Tests et qualité

```bash
# API (les tests d'intégration utilisent une vraie base PostgreSQL)
cd services/api
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'   # base vide dédiée
gofmt -l .            # doit n'afficher aucun fichier
go vet ./...
go test -race ./...
go build ./cmd/api ./cmd/seed

# Client
cd apps/mobile
npx tsc --noEmit
npm run lint -- --max-warnings 0
npm run format:check
npm test
npm run build:web
```

Sans `DATABASE_URL_TEST`, les tests d'intégration sont ignorés (et non faussement verts : ils sont listés comme `SKIP`).

**Parcours critiques** :

- Intégration API (`cmd/api/main_integration_test.go`) : inscription → profil → découverte → like → match → conversation → lecture ; permissions (un tiers ne lit jamais une conversation), filtres de préférences (âge, genre, distance, réciprocité), likes simultanés (un seul match), blocage, signalement, suppression de compte et de toutes ses données, validation de profil et d'âge, photos (types, taille, bombes de décompression), JWT (algorithme, expiration, émetteur), récupération de compte, rate limiting.
- UI (Jest) : authentification, onboarding (refus des mineurs), deck de découverte et match, conversation (envoi optimiste, erreur, accusés de lecture).
- Smoke API : `cd apps/mobile && npm run e2e:smoke` (API démarrée).
- **E2E navigateur** (Playwright) : inscription → onboarding avec upload d'une vraie photo → découverte → match → message → réponse reçue en temps réel → notifications → suppression du compte.

  ```bash
  cd apps/mobile
  EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
  python3 -m http.server 8099 --directory dist &            # n'importe quel serveur statique
  npx playwright-core install chromium
  WEB_URL=http://localhost:8099 API_URL=http://localhost:18080 npm run e2e:web
  ```
  (l'API doit autoriser l'origine : `ALLOWED_ORIGINS=http://localhost:8099`).
- Mobile natif : `npm run e2e:ui:run` joue le flux [Maestro](https://maestro.mobile.dev) (`e2e/maestro/critical-flow.yaml`) sur appareil/émulateur.
- La CI GitHub Actions (`.github/workflows/ci.yml`) exécute tout ce qui précède.

## Build et déploiement

- **API** : `docker build -t lueur-api services/api` (image Alpine, utilisateur non-root, binaires `api` et `seed`). Elle a seulement besoin de PostgreSQL et des variables ci-dessus.
- **Client web** : `EXPO_PUBLIC_API_URL=https://api.exemple.fr npm run build:web` produit `apps/mobile/dist`, à servir par n'importe quel hébergeur statique (redirigez toutes les routes vers `index.html`).
- **Applications natives** : `eas build` (Expo Application Services) à partir de `apps/mobile` ; l'identifiant d'application se règle dans `app.json`.

Checklist production :

1. `APP_ENV=production`, `JWT_SECRET` aléatoire de 48+ octets, mot de passe PostgreSQL fort, base non exposée.
2. TLS devant l'API (reverse proxy) ; renseignez `TRUSTED_PROXIES` avec l'IP du proxy, sinon toutes les requêtes semblent venir de lui et partagent les mêmes quotas.
3. SMTP configuré (les codes de récupération ne sont jamais écrits dans les logs en production).
4. `ALLOWED_ORIGINS` limité aux origines réelles du client web.
5. Sauvegardes PostgreSQL : les photos y sont stockées (voir [Décisions techniques](#décisions-techniques)).
6. Une seule instance API tant que le hub WebSocket est en mémoire, ou ajoutez un bus (Redis / LISTEN-NOTIFY) derrière `realtime.Publisher`. Les quotas de rate limiting sont aussi par instance.
7. Définissez qui traite les signalements (voir [Modération](#modération)).

## Modération

Les signalements sont enregistrés dans `reports` (statut `open`). Il n'y a pas encore d'interface d'administration ; en attendant, via `psql` :

```sql
-- Signalements à traiter, les plus signalés d'abord
SELECT reported_id, count(DISTINCT reporter_id) AS signaleurs, array_agg(DISTINCT reason) AS motifs, max(created_at) AS dernier
FROM reports WHERE status = 'open' GROUP BY reported_id ORDER BY signaleurs DESC, dernier DESC;

-- Suspendre un compte (connexion refusée, plus visible, sessions révoquées)
UPDATE users SET status = 'suspended' WHERE id = '<uuid>';
UPDATE sessions SET revoked_at = NOW() WHERE user_id = '<uuid>' AND revoked_at IS NULL;

-- Clore les signalements
UPDATE reports SET status = 'reviewed' WHERE reported_id = '<uuid>' AND status = 'open';
```

Aucune suspension automatique n'est faite : elle serait détournable par des signalements concertés.

## Sécurité et vie privée

- **Authentification** : bcrypt ; JWT d'accès HS256 de 15 min (expiration obligatoire, émetteur et claims validés, autre algorithme refusé) ; jeton de rafraîchissement opaque de 30 jours, stocké haché (SHA-256), **rotation à chaque usage** (un jeton rejoué est refusé). Mot de passe : 8-72 caractères avec lettre et chiffre. Comparaison à temps constant pour un utilisateur inconnu.
- **Autorisation côté serveur** pour toute règle métier : éligibilité d'un like (préférences mutuelles, blocages, déjà traité), accès aux conversations (404 uniforme, aucune fuite d'existence), accès aux photos et profils.
- **Récupération de compte** : code de 8 caractères à usage unique, valable 30 min, haché, 5 essais, délai de 60 s entre deux envois, réponse identique que le compte existe ou non ; tous les appareils sont déconnectés après réinitialisation.
- **Rate limiting** (par processus) : IP sur l'authentification, la récupération et le WebSocket ; par compte sur la connexion ; par utilisateur sur swipes, messages, signalements, uploads, changements de position.
- **Entrées** : validation stricte (prénom, date de naissance, bio, ville, centres d'intérêt, préférences, pagination, UUID), caractères de contrôle et invisibles supprimés, corps JSON limité à 1 Mo.
- **Photos** : seuls JPEG et PNG (détectés sur le contenu, pas sur l'en-tête déclaré), 6 Mo max, dimensions vérifiées avant décodage (anti-bombe de décompression), **réencodage systématique** en JPEG (supprime EXIF/GPS et tout contenu caché), miniature, 6 photos max. Servies uniquement à un utilisateur authentifié autorisé à voir le profil, en `private, immutable`, avec `nosniff`. Remplacer une photo change son identifiant.
- **Position** : arrondie à ~1 km avant stockage ; jamais exposée aux autres (distance affichée par tranches de 5 km, désactivable) ; le changement de position est limité à 10/heure pour empêcher la triangulation.
- **Blocage** : bidirectionnel et immédiat (découverte, profil, photos, matchs, conversations, messages). Il met fin au match.
- **Suppression de compte** : demande le mot de passe, supprime profil, photos, likes, matchs, messages, notifications, sessions et signalements liés, coupe les WebSockets. Irréversible.
- **Réseau** : CORS en liste blanche, en-têtes de sécurité (nosniff, X-Frame-Options, HSTS en HTTPS…), WebSocket authentifié par première trame (le jeton n'apparaît jamais dans une URL ou un log), IP client fiable uniquement via `TRUSTED_PROXIES`.
- Les logs masquent jetons, mots de passe et secrets.

## Décisions produit

- **Gratuité** : aucune table, route ou écran de paiement. Les quotas techniques (200 swipes/min, 60 messages/min) ne servent qu'à bloquer les abus, pas à limiter un usage normal.
- **Âge** : 18 ans minimum, vérifié côté serveur à la création du profil et pour les préférences. La date de naissance est saisie une fois et **ne peut plus être modifiée** (pas de contournement par essais successifs). L'âge n'est qu'auto-déclaré : aucune vérification d'identité.
- **Compte et profil** : e-mail + mot de passe ; pas de vérification d'e-mail à l'inscription (choix de simplicité ; à ajouter avant un lancement public si les faux comptes sont un sujet). Un profil n'est visible qu'avec au moins une photo et des préférences.
- **Genres** : `femme`, `homme`, `non-binaire` ; on choisit les genres recherchés (un ou plusieurs). L'éligibilité est **réciproque** : A voit B seulement si B recherche le genre de A, dans sa tranche d'âge et sa distance.
- **Découverte** : file serveur sans pagination par curseur — un profil liké ou passé disparaît définitivement, donc relire la même requête avance. Ordre : ceux qui vous ont déjà liké d'abord, puis les plus récemment actifs (point d'extension `discover.Strategy`). Un « pass » est définitif : on ne revoit jamais un profil traité.
- **Likes** : un seul geste par personne (like ou pass), pas d'annulation. Un like réciproque crée atomiquement le match, la conversation et les deux notifications ; deux likes simultanés donnent exactement un match.
- **Localisation** : facultative. Sans position, aucun filtre de distance n'est appliqué entre vous et les autres.
- **Chat** : texte seul (2 000 caractères), uniquement entre matchs actifs, temps réel par WebSocket avec repli par polling si la socket tombe, accusé « Lu ». « Supprimer la conversation » la masque pour soi seulement jusqu'au prochain message ; « Annuler le match » la ferme pour les deux.
- **Notifications** : intégrées à l'application (match, nouveau message — une seule notification non lue par conversation) et poussées en temps réel. Pas de notifications push système (FCM/APNs) : elles demandent des comptes développeur ; le point d'extension est `realtime.Publisher`.
- **Débloquer** ne rétablit pas le match : les deux profils ont déjà été traités.
- **Langue** : interface en français.
- **Marque** : « Lueur », nom et identité originaux (palette terracotta / prune, mode sombre automatique).

## Décisions techniques

- **Construire sur le boilerplate** : Gin, pgx, JWT, sessions, middleware et la structure `handler → service → repository` sont conservés. Le produit SaaS (Stripe, organisations, posts, commentaires, votes, fichiers, liste d'utilisateurs publique — incompatible avec la vie privée) a été retiré. Tamagui (configuré mais jamais utilisé) a été retiré au profit du kit d'UI maison.
- **Photos en base** (`bytea`, variante pleine taille + miniature) : une seule source de vérité, sauvegardée avec le reste, supprimée en cascade avec le compte, autorisation vérifiée en SQL à chaque lecture. Pour de gros volumes, migrez vers un stockage objet avec URLs signées derrière `photos.Repository`.
- **Géographie sans PostGIS** : fonction SQL `haversine_km` + pré-filtre sur la latitude (indexée). Largement suffisant jusqu'à des centaines de milliers de profils ; PostGIS/geohash au-delà.
- **Temps réel sans broker** : hub en mémoire, une instance. Le WebSocket ne porte que des événements descendants ; l'envoi de messages passe par REST (une seule voie de validation et d'autorisation).
- **Rate limiting en mémoire** : zéro dépendance, par instance.
- **Messagerie de récupération sans service tiers** : SMTP standard (`net/smtp`).
- **Dépendances ajoutées** — Go : `gorilla/websocket`. Mobile : `expo-image-picker`, `expo-image-manipulator`, `expo-location`, `@expo/vector-icons`, `@react-navigation/native-stack`, `react-dom`, `react-native-web`, `@expo/metro-runtime` (cible web) ; dev : ESLint, Prettier, Jest/jest-expo/Testing Library, `playwright-core`.

## Limites connues et pistes

- Pas de vérification d'e-mail ni de téléphone, pas de vérification de photo.
- Pas d'interface d'administration (requêtes SQL ci-dessus).
- Pas de notifications push système, ni de messages vocaux/images.
- Jetons d'accès non révocables avant expiration (15 min maximum) ; la suppression de compte et la suspension prennent effet immédiatement sur les données mais un jeton déjà émis reste syntaxiquement valide jusque-là.
- Recommandation volontairement simple (voir `discover.Strategy`) : prête à accueillir un scoring (centres d'intérêt communs, activité, taux de réponse).
