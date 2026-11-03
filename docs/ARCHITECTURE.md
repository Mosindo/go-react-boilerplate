# Architecture de Braise

## Monorepo

- `apps/mobile` — client Expo (React Native + web)
- `services/api` — API Go
- `infra` — Docker Compose
- `docs` — documentation et rôles d'agents

## Backend (`services/api`)

```text
cmd/api            point d'entrée, câblage (setupRouter), tests d'intégration
cmd/seed           données de démonstration
internal/platform  config · db (+migrations) · middleware · ratelimit · realtime · httpx · errors · logger · handlers
internal/features  auth · profiles · photos · matching · chat · notifications · safety
```

Règles : un seul serveur HTTP (Gin) ; `handler → service → repository` ; aucun SQL dans les handlers ; erreurs métier typées (`platformerrors.AppError`) traduites par `httpx.Fail` ; préoccupations partagées dans `platform`.

`profiles` expose `CardColumns`, `ScanCard` et `VisibleToViewer` : fragments SQL réutilisés par matching, chat et photos pour que la visibilité d'un profil (blocs, découvrabilité, match, like reçu) soit définie à un seul endroit. Les listes agrègent intérêts et photos dans la requête (pas de N+1).

### Modèle de données

`users`, `sessions`, `password_resets` · `profiles`, `preferences`, `interests`, `user_interests`, `photos` · `swipes`, `matches`, `conversations`, `conversation_participants`, `messages` · `blocks`, `reports`, `notifications`.

Contraintes notables : `matches.user_a < user_b` (une seule ligne par paire), `swipes` PK `(actor_id, target_id)`, `photos (user_id, position)` unique différé (réordonnancement atomique), CHECK sur genres, âges, longueurs, `lat/lng` simultanément nuls. Toutes les clés étrangères vers `users` sont `ON DELETE CASCADE`.

### Flux critiques

- **Swipe** : transaction → verrou consultatif de paire → profil complet ? cible visible ? insertion unique → si like réciproque : match + conversation + participants + 2 notifications → commit → événements WebSocket.
- **Chat** : l'appartenance à la conversation est vérifiée à chaque lecture/écriture ; un non-participant reçoit 404 (pas de fuite d'existence). Les curseurs sont les identifiants de message ; `last_read_message_id` et `cleared_up_to_id` portent l'état lu et la suppression locale.
- **Photos** : `Process` valide (taille, signature de contenu, dimensions), décode, redimensionne, ré-encode en JPEG ; `Storage` abstrait le stockage (disque local aujourd'hui).
- **Temps réel** : `realtime.Hub` (en mémoire) ; auth par premier message ; `Publish` ne bloque jamais.

## Client (`apps/mobile/src`)

```text
config.ts · theme.ts · storage.ts
api/        client (refresh automatique, timeouts), types, endpoints typés
auth/       session (jetons), AuthContext
realtime/   WebSocket + reconnexion
features/   profile (brouillon + formulaires + photos), discover (SwipeCard), match (overlay), safety (signalement)
screens/    écrans · navigation/ RootNavigator · ui/ composants
```

Aucun accès réseau direct depuis les écrans : tout passe par `api/`. L'état serveur est géré par TanStack Query ; les événements WebSocket invalident les caches concernés.
