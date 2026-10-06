# Lien — architecture

Lien est un monorepo : une API Go (`services/api`), une application mobile Expo (`apps/mobile`), l'infrastructure locale (`infra/`, `docker-compose.yml`) et des scripts de qualité (`scripts/`). Le [README](../README.md) couvre l'installation, la configuration et les décisions produit ; ce document explique **comment le code est organisé** et **quelles règles protègent ses frontières**.

## 1. Backend (Go, Gin, PostgreSQL)

```text
services/api/
├─ cmd/api/            composition root : config → pool → migrations → routeur
├─ cmd/seed/           données de démo (refuse APP_ENV=production)
├─ cmd/moderation/     outil opérateur (signalements, suspension)
└─ internal/
   ├─ platform/        préoccupations partagées, sans logique métier
   │  ├─ config/       variables d'environnement, validation stricte
   │  ├─ db/           pool pgx, migrations embarquées et suivies
   │  ├─ middleware/   auth JWT + validation de session, rate limit, limite de corps, CORS, logs
   │  ├─ realtime/     hub WebSocket en mémoire + handler authentifié
   │  ├─ ratelimit/    fenêtre fixe en mémoire
   │  ├─ mailer/       SMTP, ou log en développement
   │  ├─ storage/      blobs sur disque (clés validées)
   │  ├─ httpx/        enveloppe d'erreur, pagination
   │  └─ logger/       logs JSON, valeurs sensibles masquées
   └─ features/        un dossier par domaine
      ├─ auth/  profiles/  photos/  matching/  chat/  notifications/  safety/  account/
```

### Règles

- Chaque feature : `handler.go` (HTTP, mince) → `service.go` (règles métier, validation) → `repository.go` (SQL, un `pgxpool` partagé). `model.go` porte types et payloads, `routes.go` l'enregistrement.
- **Pas de SQL dans les handlers**, pas de logique métier dans les handlers, pas de `net/http` en parallèle de Gin (un seul serveur).
- Les features ne s'importent pas entre elles, sauf contrats explicites et étroits : `matching` et `chat` reçoivent un `Notifier` (interface) ; `photos` reçoit un `Access` (`profiles.Service.CanView`) ; `matching` lit `profiles.Repository` pour hydrater les cartes (requêtes groupées, pas de N+1).
- **Une seule règle de visibilité** (`profiles.viewableSQL`) décide qui peut voir un profil et ses photos : actif, non bloqué (dans les deux sens), et (découvrable avec photo, ou match actif, ou a liké). Elle sert au profil public et au contenu des photos.
- Toute entrée est validée côté serveur (bornes de pagination, longueurs, énumérations, UUID, types d'image par contenu). Un identifiant inconnu **ou interdit** répond `404`.

### Flux principaux

**Découverte** — `GET /discover` : `Viewer` (profil complet requis) → `Candidates` (filtres durs en SQL : genre/âge/distance mutuels, déjà swipé, blocages, photo, signalements) borné à 200 → `Ranker.Score` en Go → top *N* hydraté (intérêts et photos en deux requêtes pour tous les profils).

**Match** — `POST /swipes` : une transaction, verrou consultatif sur le couple, insertion idempotente du swipe, si like réciproque création de `conversations` + `conversation_participants` + `matches` (`user_a < user_b`, unique) ; après commit : notifications et événements temps réel (au mieux, erreurs journalisées).

**Chat** — `Membership` (match actif dont l'appelant est membre) est la porte de toute opération. L'état « lu » est `conversation_participants.last_read_at` ; « masquer » est `hidden_at` (la conversation réapparaît à tout message plus récent) ; défaire un match ou bloquer pose `matches.ended_at`.

**Temps réel** — le client ouvre `/ws`, s'authentifie par sa première trame, reçoit des événements poussés par `realtime.Hub.Publish` depuis les services. Les consommateurs lents perdent des événements plutôt que de bloquer l'émetteur ; le mobile se resynchronise par REST à la reconnexion.

### Base de données

Migrations numérotées, exécutées une fois, en transaction, sous verrou. Invariants portés par la base : `CHECK` (âge, énumérations, longueurs, coordonnées appariées), unicités (un swipe par couple orienté, un match par couple, une position de photo), clés étrangères `ON DELETE CASCADE` (la suppression d'un compte efface tout), fonctions SQL `haversine_km` et `age_years`. Positions de photos contiguës garanties par transaction + contrainte `DEFERRABLE`.

Index requis par `AGENTS.md` : `users(email)`, `messages(conversation_id, created_at)`, `notifications(user_id, created_at)`, `conversation_participants(conversation_id, user_id)` ; ajoutés pour le domaine : `profiles(latitude)`, `swipes(to_user_id, created_at) WHERE action='like'`, `matches(user_a|user_b, created_at)`, `blocks(blocked_id, blocker_id)`, `reports(reported_id, status)`.

## 2. Application mobile (Expo, React Native, TypeScript)

```text
apps/mobile/
├─ App.tsx                   AuthProvider → RealtimeProvider → garde : connexion | onboarding | application
└─ src/
   ├─ api/                   client unique (aucun fetch dans l'UI), types, endpoints, appels par domaine
   ├─ hooks/                 session (useAuth), requêtes partagées
   ├─ realtime/              WebSocket authentifié, backoff, resynchronisation
   ├─ navigation/            stack racine (détails, chat, réglages…) + onglets
   ├─ screens/               un écran = une responsabilité
   ├─ components/            SwipeDeck, ProfileCard, MatchModal, forms/*
   ├─ shared/                design system (ui), layout, feedback (toasts, états vides/erreur)
   ├─ state/                 petits stores externes (onboarding, bus de swipes)
   └─ lib/                   logique pure et testée (validation, formats, JWT, erreurs)
```

- **Client API** : un seul point d'entrée (`api/client.ts`) ; refresh automatique du token avec *single-flight* (les 401 concurrents partagent un seul refresh) ; la session expirée remonte à `AuthProvider`. Les photos privées sont chargées avec l'en-tête `Authorization`.
- **Design system** : `shared/ui/tokens.ts` (palettes clair/sombre) + `useTheme()` suit l'apparence du système ; aucune couleur en dur dans les écrans (hors surcouches photo).
- **Accessibilité** : rôles et libellés sur les contrôles, toute action de swipe a un bouton équivalent, annonces `alert`/`live region` sur les erreurs, cibles tactiles confortables.
- **Performance** : listes virtualisées (`FlatList`), pagination par curseur pour les messages, pile de découverte rechargée par lots de 10 quand il reste 3 cartes, photos mises en cache (`immutable`) et redimensionnées avant upload.
- Le répertoire `src/theme` et `tamagui.config.ts` sont des vestiges de l'ancien socle : non utilisés à l'exécution.

## 3. Qualité

`scripts/check.sh` exécute tout (gofmt, vet, tests, build, tsc, eslint, prettier, tests mobile). La CI GitHub (`ci.yml`) fait de même avec PostgreSQL en service et `REQUIRE_DB=1`, puis construit l'image Docker et vérifie `/health`.
