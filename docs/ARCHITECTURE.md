# Architecture de Lueur

## Vue d'ensemble

```text
 App Expo (iOS / Android / Web)
   │  REST JSON (Bearer JWT)            WebSocket /ws (événements)
   ▼                                        ▲
 API Go (Gin) ── features ── platform ──────┘
   │
   ├─ PostgreSQL (pgxpool partagé, migrations embarquées)
   └─ Stockage photos (disque local, interface Store)
```

Un seul serveur HTTP Gin sert l'API REST, les médias signés et le WebSocket.

## API (`services/api`)

### Composition (`cmd/api/main.go`)

Charge la configuration, ouvre **un pool Postgres unique**, applique les migrations, construit le stockage, le mailer et le hub temps réel, puis assemble les features et leurs limites de débit. Serveur `http.Server` avec timeouts et arrêt gracieux.

### Plateforme (`internal/platform`)

| Paquet | Rôle |
| --- | --- |
| `config` | Variables d'environnement, validation (secret fort, `APP_ENV`, SMTP) |
| `db` | Connexion avec retry, migrations suivies (`schema_migrations` + verrou consultatif) |
| `httpx` | Enveloppe d'erreur `{error, code}`, `ValidationError`, parsing strict (UUID, limit, curseur keyset) |
| `middleware` | Auth (JWT strict + session active), CORS, en-têtes de sécurité, request id, logs, métriques, rate limiting |
| `realtime` | Hub WebSocket en mémoire, interface `Publisher` |
| `storage` | Stockage local des photos (clés générées, anti path traversal), URLs signées HMAC expirantes |
| `mailer` | SMTP, mailer de dev (logs), mailer désactivé (prod sans SMTP) |
| `logger`, `handlers` | Logs structurés avec masquage des secrets, `/health` |

### Features (`internal/features`)

Chaque feature : `model.go` (types + payloads), `repository.go` (SQL, interface + implémentation pgx), `service.go` (règles métier), `handler.go` (HTTP fin), `routes.go`.

| Feature | Responsabilités | Dépendances |
| --- | --- | --- |
| `auth` | Inscription, connexion, refresh rotatif, logout, reset par code, suppression de compte, vérification de session | `mailer`, hook de suppression (photos) |
| `profiles` | Profil privé, préférences, intérêts, position arrondie, confidentialité, cartes publiques en lot, visibilité | signeur d'URL |
| `photos` | Upload/remplacement/suppression/ordre, pipeline d'image, service des médias signés, nettoyage à la suppression | `storage` |
| `matching` | Découverte (règles dures en SQL + `Ranker`), swipes, création de match, boîte de réception, annulation | `profiles` (cartes), `notifications`, `realtime` |
| `chat` | Messages d'un match (membres uniquement), envoi idempotent, marqueur de lecture | `realtime` |
| `safety` | Blocages, signalements (instantané de conversation), masquage automatique | `realtime` |
| `notifications` | Création interne (non exposée aux clients), liste, compteur, lecture | `realtime` |

Les dépendances entre features passent par des **interfaces** déclarées côté consommateur (`CardLoader`, `Notifier`, `AccountDeletionHook`).

### Règles de matching

1. **Éligibilité (SQL)** : même organisation, profil visible, non masqué par la modération, au moins une photo ; genre et âge compatibles **dans les deux sens** ; jamais swipé par le spectateur ; aucun blocage dans un sens ou l'autre ; pas déjà matché ; distance ≤ min(distance max des deux) si le spectateur a une position (pré-filtre par boîte englobante indexée).
2. **Classement (Go)** : `ScoreRanker` sur un lot de 100 candidats ; remplaçable via l'interface `Ranker`.
3. **Swipe** : transaction avec `pg_advisory_xact_lock` sur la paire → insertion idempotente du swipe → si like réciproque, création du match (`UNIQUE (user_low_id, user_high_id)`), de la conversation et des participants. Les likes simultanés produisent exactement un match (testé).

### Temps réel

`/ws` : le client envoie `{"type":"auth","token":…}` dans les 10 s ; le serveur répond `ready`, envoie un ping toutes les 25 s et revérifie la session toutes les 5 min. Événements : `message.new`, `message.read`, `match.new`, `match.removed`, `notification.new`. Un client lent est déconnecté plutôt que de bloquer l'émetteur ; le client se resynchronise par REST à la reconnexion.

### Performance

- Pool unique ; aucune connexion par requête.
- Pas de N+1 : cartes, photos et intérêts chargés en lot (`= ANY($1)`) pour une page entière.
- Pagination keyset (`(created_at, id) < (…)`) sur messages, matchs et notifications.
- Index : `users.email`, `profiles (gender, birthdate)` partiel, `profiles (latitude, longitude)`, `profiles (last_active_at)`, `swipes` (PK + likes reçus), `matches` (paire + `user_high_id`), `conversation_participants (conversation_id, user_id)`, `messages (conversation_id, created_at)`, `notifications (user_id, created_at)`, `blocks` (deux sens), `reports (reported_user_id, created_at)`, `photos (user_id, position)`.
- Photos redimensionnées (1600 px) et compressées ; URLs stables par fenêtre de 6 h pour le cache client.

## Application mobile (`apps/mobile`)

- **Données** : TanStack Query (cache, pagination infinie, refetch au premier plan) ; mutations optimistes pour les swipes et les messages.
- **Client API** (`src/api/client.ts`) : seul point d'accès réseau ; ajoute le jeton, **rafraîchit automatiquement** une seule fois en cas de 401 (single flight), erreurs typées `ApiError {status, code}`.
- **Temps réel** (`src/api/realtime.ts`, `src/hooks/useRealtime.ts`) : met à jour le cache (messages, accusés de lecture, matchs, notifications) et bascule sur un polling léger hors connexion.
- **Navigation** : pile d'authentification → assistant d'onboarding (si profil incomplet) → onglets (Découvrir, Matchs, Activité, Moi) + écrans empilés (chat, profil, réglages).
- **Design system** : `src/theme` (palettes claire/sombre, typographie serif/sans-serif), `src/ui` (composants accessibles). Aucune couleur codée en dur dans les écrans.

## Évolutions prévues par l'architecture

- Scalabilité horizontale : implémenter `realtime.Publisher` avec Postgres LISTEN/NOTIFY ou Redis, rate limiting partagé.
- Stockage objet (S3, GCS) : implémenter `storage.Store`.
- Notifications push : brancher Expo Push dans `notifications.Service.Notify`.
- Recommandation : nouveau `matching.Ranker`.
- Console de modération : traiter la table `reports` et `profiles.moderation_hidden_at`.
