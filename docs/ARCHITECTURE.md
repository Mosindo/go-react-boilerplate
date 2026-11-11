# Architecture de Seren

Ce document décrit l'organisation du code pour qu'un·e développeur·se (ou un agent) puisse naviguer, comprendre les responsabilités et étendre le produit sans casser ses frontières.

## 1. Vue d'ensemble

```text
 App Expo (iOS / Android / web)
   │  REST JSON (Bearer access token)        WebSocket /realtime?ticket=…
   ▼                                         ▼
 API Go (Gin) ── features ── platform ── PostgreSQL 16
                     │
                     └── stockage photos (disque, interface Storage)
```

Un seul serveur HTTP (Gin) sert l'API REST, les médias signés et le WebSocket. Un seul pool `pgxpool` est partagé par toutes les features.

## 2. Backend (`services/api`)

### 2.1 `cmd/`

- `cmd/api/main.go` — racine de composition : charge la configuration, ouvre le pool, applique les migrations, construit le routeur (`newRouter`) et gère l'arrêt propre. Les tests d'intégration HTTP vivent à côté (`*_integration_test.go`) et réutilisent `newRouter`.
- `cmd/seed/main.go` — données de démonstration (refusé en production).

### 2.2 `internal/platform/` (transverse, sans logique métier)

| Paquet | Rôle |
| --- | --- |
| `config` | Variables d'environnement validées (`JWT_SECRET` robuste, `APP_ENV`, SMTP…) |
| `db` | Connexion avec retry, migrations embarquées, suivies (`schema_migrations`) et sérialisées par verrou consultatif |
| `tokens` | Signature/validation JWT HS256 : access tokens (15 min) et tickets temps réel (60 s), types non interchangeables |
| `middleware` | En-têtes de sécurité, CORS, request id, logs, métriques, `RequireUser` (JWT + session active), limite de corps, rate limiting |
| `ratelimit` | Token bucket en mémoire par clé |
| `httpx` | Format d'erreur unique, parsing pagination/UUID, identité de la requête |
| `storage` | Interface de stockage d'objets + implémentation disque (clés générées, anti path-traversal) |
| `media` | URLs signées HMAC et expirantes pour les photos |
| `mailer` | SMTP, ou logs en développement |
| `realtime` | Hub WebSocket en mémoire (`Publisher`) |
| `logger`, `handlers` | Logs structurés et `/health` |

### 2.3 `internal/features/` (domaines)

Chaque feature contient `model.go`, `repository.go` (SQL uniquement), `service.go` (règles métier), `handler.go` (HTTP fin), `routes.go` (+ tests). Le flux est toujours `handler → service → repository`.

| Feature | Responsabilité | Dépend de |
| --- | --- | --- |
| `auth` | Comptes, sessions, refresh, mot de passe oublié, suppression de compte | `tokens`, `mailer`, nettoyeur de photos, hub (déconnexion) |
| `profiles` | Profil, préférences, intérêts, localisation arrondie, visibilité, **cartes publiques** construites par lot | `media` |
| `photos` | Upload, traitement d'image, ordre, remplacement, suppression, service des médias signés | `storage`, `media`, `profiles` (réponse) |
| `matching` | Découverte (SQL de filtrage + `Ranker`), swipes, création de match, annulation | `profiles` (cartes), `notifications`, `realtime` |
| `chat` | Conversations (= matchs), messages, lecture, masquage | `profiles`, `notifications`, `realtime` |
| `safety` | Blocages et signalements | `realtime` |
| `notifications` | Notifications in-app, regroupement, push temps réel | `realtime` |
| `realtime` | Ticket et connexion WebSocket (pas de repository) | `tokens`, hub, sessions |

Les dépendances entre features passent par de petites interfaces déclarées côté consommateur (`CardProvider`, `Notifier`, `AccountDataCleaner`…), ce qui garde les services testables.

### 2.4 Règles métier clés

- **Découverte** : filtres stricts et réciproques en SQL (genre recherché des deux côtés, tranche d'âge des deux côtés, distance ≤ min des deux distances max, profil découvrable avec au moins une photo, ni déjà swipé, ni bloqué dans un sens ou l'autre). Le `DefaultRanker` ordonne ensuite un pool 4× plus grand que la page. Si l'un des deux membres n'a pas partagé sa position, le filtre de distance ne s'applique pas (la carte n'affiche alors aucune distance) : la localisation reste facultative.
- **Match** : verrou consultatif par paire + `UNIQUE(user_low_id, user_high_id)` ; seul le second like crée le match ; le swiper reçoit le match dans la réponse HTTP, l'autre membre par WebSocket + notification.
- **Chat** : l'insertion verrouille la ligne du match (pas de message après un unmatch/blocage concurrent) ; « lu » = `match_members.last_read_at` ; masquer = `hidden_at`, levé par tout nouveau message.
- **Blocage** : supprime le match (cascade messages) et exclut les deux profils l'un de l'autre partout.
- **Signalement** : instantané de preuves puis blocage automatique ; un seul signalement ouvert par paire.
- **Suppression de compte** : mot de passe requis, cascade SQL complète, fichiers supprimés, connexions temps réel fermées.

### 2.5 Schéma

Voir le tableau dans le README et les fichiers `internal/platform/db/migrations/*.sql`. Les migrations sont append-only : toute évolution passe par un nouveau fichier numéroté.

## 3. Mobile (`apps/mobile`)

| Dossier | Rôle |
| --- | --- |
| `src/api` | `config.ts` (URL de l'API), `client.ts` (fetch typé, timeouts, erreurs, refresh single-flight), `endpoints.ts` (une fonction par route), `types.ts` |
| `src/hooks` | `useAuth` (session + stockage sécurisé), `queries.ts` (clés et hooks React Query, mises à jour optimistes), `useRealtime` (WebSocket → cache) |
| `src/navigation` | `RootNavigator` : non connecté → pile Auth ; profil incomplet → Onboarding ; sinon onglets + pile (profil, chat, signalement, réglages) |
| `src/screens` | Écrans par domaine : `auth`, `onboarding`, `discover`, `chat`, `notifications`, `profile` |
| `src/features` | Blocs réutilisés par l'onboarding et l'édition : formulaires, gestion des photos, localisation |
| `src/ui`, `src/theme` | Design system thémé (clair/sombre) : texte, boutons, champs, puces, sections, états, toasts/dialogues |
| `src/utils` | Formatage FR et validations partagées (testées) |

Principes : aucun appel réseau dans les écrans (tout passe par `src/api`), état serveur dans React Query, temps réel injecté dans le cache, repli en polling quand le WebSocket est coupé, images via `expo-image` (cache mémoire + disque), listes virtualisées avec pagination par curseur.

## 4. Étendre le produit

1. Ajouter une migration numérotée.
2. Créer `internal/features/<feature>/` avec les cinq fichiers et ses tests.
3. Brancher la feature dans `newRouter` (`cmd/api/main.go`) avec le bon rate limit.
4. Ajouter les fonctions d'API dans `apps/mobile/src/api/endpoints.ts`, les hooks dans `src/hooks/queries.ts`, puis l'écran.
5. Couvrir le scénario dans `cmd/api/flow_integration_test.go` (et `e2e/` si c'est un parcours critique).
