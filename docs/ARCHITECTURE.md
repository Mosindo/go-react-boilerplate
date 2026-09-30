# Architecture de Lueur

## Vue d'ensemble

```text
 App Expo (iOS / Android / Web)
   │  REST (JSON, JWT)         WebSocket (ticket 60 s)
   ▼                            ▼
 API Go (Gin) ── internal/app : composition root
   ├─ platform/  config, db (migrations), middleware (sécurité, CORS, auth),
   │             authtoken, ratelimit, realtime (hub WS + LISTEN/NOTIFY),
   │             storage (fichiers privés), media (URLs signées), geo, mailer,
   │             httpx (réponses/erreurs), errors (AppError)
   └─ features/  auth · profiles · photos · discovery · matching · chat ·
                 notifications · safety
   ▼
 PostgreSQL 16  (+ dossier privé des photos)
```

## Backend

Chaque feature contient `model.go`, `repository.go` (SQL explicite, pgx), `service.go` (règles métier, erreurs `AppError`), `handler.go` (liaison HTTP mince), `routes.go`. Les handlers ne contiennent ni SQL ni logique métier ; `httpx.Fail` transforme une `AppError` en réponse `{error, code}` et masque toute autre erreur derrière un 500 journalisé.

Dépendances entre features (par interfaces) :

- `profiles` fournit les cartes publiques et résumés (`Cards`, `Summaries`) à `discovery`, `matching`, `chat`, `safety`, ainsi que les fragments SQL partagés (`CompleteSQL`, `EligibleSQL`, `NotBlockedSQL`, `DistanceSQL`) : une seule définition de « profil complet », « visible » et « bloqué ».
- `notifications.Notifier` est utilisé par `matching` et `chat`.
- `realtime.Publisher` est utilisé par `notifications`, `matching`, `chat`, `safety`.
- `photos` implémente `auth.AccountCleaner` pour supprimer les fichiers après suppression du compte.

### Modèle de données

| Table | Rôle |
| --- | --- |
| `users` | Compte (email unique, hash bcrypt, `last_active_at`, `is_demo`) |
| `sessions` | Refresh tokens hachés, rotation, révocation |
| `password_reset_codes` | Code haché, essais, expiration |
| `profiles` | Informations affichées, coordonnées arrondies, confidentialité |
| `preferences` | Genres recherchés, tranche d'âge, distance max |
| `photos` | Clé de stockage privée, position 0–5 (unique par utilisateur, contrainte différable) |
| `interests`, `user_interests` | Catalogue et choix (max 10) |
| `swipes` | Like / pass, clé primaire (auteur, cible) |
| `matches` | Paire canonique (low < high), unique |
| `conversations`, `conversation_participants`, `messages` | Chat lié au match, lecture (`last_read_at`), suppression locale (`hidden_at`) |
| `blocks`, `reports` | Sécurité et modération |
| `notifications` | Notifications in-app |

Toutes les relations utilisent des clés étrangères avec `ON DELETE CASCADE` (sauf `reports`, conservés avec `SET NULL` pour l'historique de modération). Supprimer un compte supprime donc profil, photos (fichiers compris), swipes, matchs, conversations, messages, blocages et notifications.

Index principaux : `users(email)` unique, `users(last_active_at)`, `profiles(gender, birthdate) WHERE discoverable`, `profiles(latitude)`, `swipes(target_id, action, created_at)`, `matches(user_high_id)`, `messages(conversation_id, created_at DESC, id DESC)`, `conversation_participants(user_id, conversation_id)`, `notifications(user_id, created_at)`, `blocks(blocked_id)`, `reports(reported_id, status)`.

### Flux clés

- **Swipe** : transaction avec verrou consultatif sur la paire → vérifications (profil de l'auteur complet, cible visible et non bloquée) → insertion du swipe (conflit = déjà traité) → si like réciproque : match + conversation + participants → après commit : événements temps réel et notifications.
- **Message** : vérification de participation (et absence de blocage) → insertion + `last_message_at` + lecture implicite de l'expéditeur → publication `message.created` aux deux participants → notification dédupliquée au destinataire.
- **Temps réel** : `POST /realtime/ticket` → `GET /realtime?ticket=…` → le hub garde les connexions locales ; `Publish` fait un `pg_notify`, chaque instance écoute et relaie à ses clients.

## Frontend

```text
src/
├─ design/        theme.ts (palettes clair/sombre, espacements, typo), ThemeProvider,
│                 components/ (Text, Button, IconButton, TextField, Chip, Avatar, Card,
│                 ListRow, Stepper, ActionSheet, Screen, états Loading/Empty/Error)
├─ lib/
│  ├─ api/        config, client (refresh transparent single-flight), session (SecureStore),
│  │              endpoints typés, types, realtime (client WebSocket avec backoff)
│  ├─ session/    SessionProvider (restauration, connexion, déconnexion, expiration)
│  ├─ realtime/   RealtimeProvider (événements → cache React Query)
│  ├─ queryClient.ts, format.ts (+ tests), toast.tsx
├─ features/      écrans et composants par domaine
└─ navigation/    RootNavigator : Auth → Onboarding → App (onglets + écrans empilés)
```

Règles : aucun appel réseau direct depuis un écran (tout passe par `lib/api/endpoints.ts`), cache et synchronisation via React Query, temps réel appliqué au cache (pas de polling tant que la connexion est active), styles via le thème (mode sombre automatique).
