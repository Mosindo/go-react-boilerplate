# Architecture — Aurore

Aurore est un monorepo : une API Go, une application Expo (iOS, Android, web) et leur infrastructure locale.

## 1. Vue d'ensemble

```text
 Expo app (React Native / web)
   ├─ REST  ──────────────►  Gin API ──► service ──► repository ──► PostgreSQL (pgxpool)
   └─ WebSocket (push seul) ◄─ realtime.Hub ◄─ services (match, message, lecture…)
                                   └──────────► photos sur disque (UPLOAD_DIR)
```

- Un seul serveur HTTP (Gin enveloppé dans `http.Server` pour les timeouts), un seul pool de connexions.
- Les clients n'envoient **jamais** de données par WebSocket : l'écriture passe par REST (validation, limitation de débit, persistance). Le socket ne sert qu'à pousser des événements.

## 2. Backend (`services/api`)

```text
cmd/api            point d'entrée, câblage des dépendances, tests d'intégration
cmd/seed           données de démonstration (refuse APP_ENV=production)
internal/platform/
  config           variables d'environnement validées
  db               pool, migrations embarquées (schema_migrations, verrou consultatif)
  middleware       auth (JWT + session), CORS, en-têtes de sécurité, limitation de débit, logs
  realtime         hub en mémoire + handler WebSocket
  mailer           SMTP (production) ou log (développement)
  geo              arrondi des coordonnées, haversine, paliers de distance
  logger, errors, handlers
internal/features/
  auth             comptes, sessions, récupération, suppression
  profiles         profil, préférences, centres d'intérêt, profil public
  photos           upload validé + ré-encodage, ordre, remplacement, accès contrôlé
  matching         découverte, swipes, matchs
  chat             conversations, messages, lu/non lu, masquage
  notifications    liste, lecture
  safety           blocages, signalements
```

Chaque feature suit `handler.go → service.go → repository.go` (+ `model.go`, `routes.go`). Les handlers ne contiennent ni SQL ni règle métier.

### Règles métier centralisées

- **Éligibilité d'un profil** (`matching/repository.go`, `eligibleQuery`) : une seule requête SQL définit « qui peut voir / liker qui » (genres réciproques, tranches d'âge réciproques, distance ≤ min des deux maxima, photo présente, compte actif, non bloqué, non déjà traité). La découverte et la validation d'un swipe l'utilisent toutes les deux.
- **Match** : transaction avec verrou consultatif sur la paire (`pg_advisory_xact_lock`), insertion du swipe, détection du like réciproque, création `matches` (paire ordonnée `user_a_id < user_b_id`) + `conversations` + participants + notifications.
- **Visibilité d'une photo / d'un profil** : propriétaire, ou profil découvrable non bloqué, ou match. Sinon `404`.

### Données

Voir `internal/platform/db/migrations/014_dating_schema.sql`. Chemins chauds indexés : `swipes (to_user_id, from_user_id) WHERE like`, `matches (user_a/b_id, created_at)`, `messages (conversation_id, created_at, id)`, `notifications (user_id, created_at)`, `conversation_participants (conversation_id, user_id)`, `photos (user_id, position)`, `profiles (latitude, longitude) WHERE is_discoverable`.

## 3. Mobile (`apps/mobile`)

```text
App.tsx                 providers (SafeArea, React Query, Auth)
src/screens/Root.tsx    chargement → connexion → onboarding → application
src/navigation/         piles et onglets (React Navigation)
src/api/                client REST (refresh de session transparent), un module par domaine
src/hooks/              useAuth, useRealtime (WebSocket + reconnexion), useChat, useDiscoverQueue…
src/components/         cartes de swipe, galerie, dialogues, gestion des photos…
src/shared/             jetons de design, composants UI, retours (chargement/erreur/vide), mise en page
src/lib/                validation, formats, localisation, sélection d'image
```

- Aucun appel réseau dans l'UI : tout passe par `src/api`.
- État serveur dans TanStack Query ; les événements temps réel invalident les requêtes concernées ; à la (re)connexion tout est resynchronisé.
- Découverte : file locale de ~10 profils, rechargée quand il en reste 3 ; swipe optimiste avec retour arrière si l'API échoue.
- Photos : un `<img>` web ne peut pas envoyer `Authorization` ; `AuthImage` récupère l'image avec le jeton et l'affiche depuis un `Blob` (natif : `Image` avec en-têtes).

## 4. Qualité

| Niveau | Outil |
| --- | --- |
| Unitaires Go | validation des profils, images, middleware JWT, limiteur, hub, géo |
| Intégration Go | PostgreSQL réel : auth, profils, photos, découverte, swipes, match/chat, permissions, blocage, suppression, concurrence, WebSocket |
| Unitaires mobile | Jest (`src/lib`) |
| E2E | Playwright : deux navigateurs, parcours inscription → match → chat temps réel → blocage → suppression |
| Statique | `gofmt`, `go vet`, `tsc --noEmit`, ESLint (`--max-warnings 0`), Prettier |
