# Lien — Architecture

Ce document décrit l'organisation du dépôt pour qu'un développeur (ou un agent) puisse trouver sa place, étendre le produit et respecter les frontières. Le README couvre l'installation, l'API et la sécurité ; ici, le « pourquoi » et les flux.

## 1. Monorepo

- `apps/mobile` : application Expo (React Native iOS/Android, et web pour le dev/les tests)
- `services/api` : API Go (un seul binaire HTTP) + deux outils d'exploitation (`cmd/seed`, `cmd/moderate`)
- `docker-compose.yml` (et `infra/`) : PostgreSQL + API
- `scripts/` : vérifications de fumée PowerShell qui délèguent au parcours Node
- `docs/` : ce document et les notes d'agents

## 2. Backend

```text
services/api/
├─ cmd/api/            main.go (composition root), events.go (pont événements)
├─ cmd/seed/           comptes de démo (refuse APP_ENV=production)
├─ cmd/moderate/       signalements, suspension
└─ internal/
   ├─ platform/        config · db (+ migrations, dbtest) · middleware · httpx · logger · errors
   │                   storage · mediaurl · mailer · realtime (hub, tickets) · handlers (health)
   └─ features/        auth · profiles · photos · discover · matching · chat
                       safety · notifications · realtime · account
```

Chaque feature suit `handler → service → repository` (+ `model.go`, `routes.go`). Les handlers valident le transport et traduisent les erreurs ; le service porte les règles métier ; le repository contient tout le SQL.

**Les features ne s'importent pas entre elles.** Quand l'une a besoin d'une autre, elle déclare une petite interface côté consommateur et `cmd/api` fait le branchement :

| Consommateur | Interface | Fournisseur |
| --- | --- | --- |
| discover, matching, chat, safety | `ProfileSource` (`Cards`) | `profiles.Service` |
| matching | `Events.MatchCreated` | `eventBridge` → notifications + temps réel |
| chat | `Events.MessageCreated/MessagesRead` | `eventBridge` |
| account | `Media`, `Sessions` | `photos.Service`, `realtime.Hub` |

### Flux : like → match → message

```text
POST /swipes ──► matching.Service.Swipe
                  └─ repository.Swipe (1 transaction)
                       1. verrou consultatif de la paire (A,B)
                       2. contrôles : profil complet, cible visible, pas de blocage
                       3. match existant ? → renvoyer l'existant (idempotent)
                       4. UPSERT swipe
                       5. like + like inverse ? → INSERT matches (user_a < user_b) ON CONFLICT DO NOTHING
                  └─ si match créé : Events.MatchCreated → notifications (2) + WebSocket "match.new" (2)

POST /conversations/:matchId/messages ─► chat.Service.Send
                  └─ Participant(user, match) sinon 404  ── contrôle d'accès unique, en tête de chaque opération
                  └─ INSERT message + activity_at (1 transaction) → Events.MessageCreated
                       → WebSocket "message.new" (destinataire + autres appareils de l'expéditeur)
                       → notification regroupée par conversation
```

### Découverte

`discover/repository.go` contient une requête unique dont le `WHERE` applique **toutes** les règles dures (préférences mutuelles de genre, d'âge et de distance, profil masqué, sans photo, déjà swipé, bloqué dans un sens ou l'autre). Le classement est isolé dans `ranking.go`. Le service récupère ensuite les cartes en **une seule requête** (`profiles.Cards`, pas de N+1).

### Photos

`photos.Process` est la seule porte d'entrée des octets : contenu → décodage borné → rotation EXIF → réduction → JPEG propre. Le stockage (`storage.Store`) est une interface (disque local fourni). Les URLs servies sont signées (`mediaurl`), regroupées par tranche d'une heure pour rester cacheables.

### Temps réel

`platform/realtime.Hub` : une file bornée par connexion ; un client lent est déconnecté plutôt que de bloquer l'émetteur (il se resynchronise en REST). Authentification de la socket par **ticket à usage unique de 60 s** obtenu via REST.

## 3. Base de données

Migrations embarquées et suivies dans `schema_migrations` (voir README). Principes : contrainte d'unicité plutôt que vérification applicative, `ON DELETE CASCADE` pour que la suppression de compte soit un simple `DELETE FROM users`, index pour chaque tri/filtre fréquent :

| Besoin | Index |
| --- | --- |
| Connexion | `users.email` (unique) |
| Chat | `messages (match_id, created_at DESC, id DESC)`, `messages (match_id, sender_id) WHERE read_at IS NULL` |
| Conversations | `matches (user_a, activity_at DESC)`, `matches (user_b, activity_at DESC)` |
| Photos | `photos (user_id, position)` unique différé |
| Notifications | `notifications (user_id, created_at DESC)`, partiel sur les non lues |
| Découverte | `profiles (latitude) WHERE discoverable`, `swipes (target_id, created_at) WHERE action = 'like'`, `blocks (blocked_id, blocker_id)` |

## 4. Mobile

```text
App.tsx ─ SafeAreaProvider ─ AuthProvider (session + QueryClient)
          ├─ signedOut → AuthScreen
          ├─ signedIn & profil incomplet → OnboardingScreen
          └─ signedIn → RealtimeProvider → RootNavigator (stack) → onglets + écrans
```

- **Pas de `fetch` dans les écrans** : tout passe par `src/api` (`client.ts` ajoute le jeton, renouvelle la session sur 401 avec un seul refresh partagé, convertit les erreurs ; `platform.ts` expose des fonctions typées).
- **React Query** pour l'état serveur ; les événements WebSocket mettent le cache à jour (`setQueryData`) ou l'invalident. À la déconnexion, tout le cache est vidé.
- **Logique pure dans `src/lib`** (validation, formats, file de swipe, construction de la conversation) : c'est ce qui est testé unitairement.
- **Design system** : `shared/ui/tokens.ts` (palette, espacements, typographies) → composants `shared/ui` → composants métier `components/`.

## 5. Où ajouter quoi

- Nouvelle règle de classement → `discover/ranking.go`.
- Nouvel événement temps réel → `RealtimeEvent` (`apps/mobile/src/api/types.ts`), `eventBridge` côté API, `handleEvent` dans `useRealtime.tsx`.
- Nouveau backend de fichiers → implémenter `storage.Store`.
- Nouvel écran → `src/screens`, route dans `navigation/types.ts` et `RootNavigator.tsx`.
