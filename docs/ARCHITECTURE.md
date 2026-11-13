# Architecture d'Aimant

Ce document décrit l'organisation du code, le modèle de données et les flux clés (matching, chat, temps réel) pour que les développeurs et agents puissent faire évoluer le produit sans casser ses frontières.

## 1. Vue d'ensemble

```text
 App Expo (iOS / Android / web)
   │  REST JSON (Bearer access token, refresh auto)
   │  WebSocket /realtime (événements serveur)
   ▼
 API Go (Gin) ── pgxpool ──► PostgreSQL
   │
   └─ storage.Store (disque local, extensible S3) pour les photos
```

Un seul serveur HTTP, un seul pool de connexions partagé, migrations embarquées appliquées au démarrage.

## 2. Backend (`services/api`)

```text
cmd/api/main.go            composition : config, pool, migrations, wiring, serveur HTTP + arrêt propre
cmd/seed/main.go           profils de démonstration (refuse APP_ENV=production)
internal/platform/
  config/                  variables d'environnement validées
  db/                      connexion, migrations suivies (schema_migrations + verrou consultatif)
  middleware/              en-têtes de sécurité, CORS, request id, logs, JWT + session, limite de corps
  httpx/                   erreurs JSON homogènes, pagination, validation d'UUID
  ratelimit/               limiteur à fenêtre fixe (en mémoire)
  storage/                 interface Store + implémentation disque (anti-traversée)
  media/                   URLs signées HMAC pour les photos
  realtime/                hub WebSocket par utilisateur (interface Publisher)
  mailer/                  e-mails (journal en dev, SMTP en prod)
  logger/                  logs JSON avec masquage des secrets
internal/features/
  auth/                    comptes, sessions, récupération, suppression
  profiles/                profil, préférences, localisation, centres d'intérêt, cartes publiques
  photos/                  upload, traitement d'image, ordre, service des médias
  matching/                découverte + Ranker, swipes, matchs, annulation
  chat/                    conversations, messages, lectures, effacement local
  notifications/           notifications in-app + push temps réel
  safety/                  blocages, signalements, modération automatique
  realtime/                endpoint WebSocket
```

Règles :

- `handler` : parse / valide l'entrée HTTP, appelle le service, traduit les erreurs en codes HTTP. Pas de SQL.
- `service` : règles métier, autorisations, orchestration (notifications, événements temps réel).
- `repository` : SQL explicite et transactions. Aucune logique HTTP.
- Les dépendances entre features passent par des interfaces étroites définies côté consommateur (`matching.CardLoader`, `chat.Notifier`, `auth.AccountCleaner`…), câblées dans `main.go`.

## 3. Modèle de données

| Table | Rôle | Clés / contraintes |
| --- | --- | --- |
| `users` | identifiants, hash bcrypt, `is_demo` | e-mail unique |
| `sessions` | refresh tokens (SHA-256), révocation | index token, utilisateur |
| `password_resets` | codes HMAC, essais, expiration | un code actif par utilisateur |
| `profiles` | prénom, naissance, genre, bio, métier, taille, objectif, ville, lat/lon arrondies, pause, distance visible, modération, complétude, dernière activité | CHECK sur chaque champ ; index partiel de découverte |
| `preferences` | genres recherchés, âge min/max, distance | CHECK (18–99, 1–500 km, genres connus) |
| `interests`, `user_interests` | catalogue et choix | PK composite |
| `photos` | clé de stockage, position 0–5, dimensions | `UNIQUE(user_id, position)` différée |
| `swipes` | like / pass | PK `(swiper, cible)`, pas d'auto-swipe |
| `matches` | paire matchée, annulation | `user_a_id < user_b_id`, `UNIQUE(paire)` |
| `conversations`, `conversation_participants`, `messages` | fil par paire, lecture, effacement local | `direct_key` unique, index `(conversation_id, created_at)` |
| `blocks` | blocages | PK `(bloqueur, bloqué)` |
| `reports` | signalements | un signalement ouvert par paire |
| `notifications` | in-app, données JSON | index `(user_id, created_at)` |

La fonction SQL `refresh_profile_completeness(user_id)` recalcule `profiles.is_complete` (prénom, naissance, genre, localisation, ≥ 1 photo, ≥ 1 genre recherché) après chaque modification concernée.

## 4. Flux de matching

1. `GET /discovery` : le service vérifie que le profil du demandeur est complet, non mis en pause et non masqué par la modération (sinon 409 `profile_incomplete` / `profile_paused` ou 403 `profile_in_review`), puis le repository applique tous les filtres durs en une requête (préférences et âges **mutuels**, distance ≤ min des deux rayons via une bande de latitude indexée + haversine, profils visibles, pas de swipe déjà fait, pas de « pass » reçu, pas de blocage) sur un pool de 200.
2. Le `Ranker` (pondération récence / proximité / intérêts communs) trie le pool ; seule la page retournée est hydratée (3 requêtes groupées : cartes, photos, intérêts).
3. `POST /swipes` : transaction avec `pg_advisory_xact_lock` sur la paire → insertion du swipe (unique) → si like réciproque, insertion du match (unique). Rejouer la même action est idempotent ; changer d'avis renvoie 409.
4. Nouveau match : notification + événement `match.created` pour les deux membres.

## 5. Chat et temps réel

- Autorisation unique `CanChat` : match actif, aucun blocage **et** aucun des deux profils masqué par la modération. Sinon 404, quelle que soit la raison.
- Messages paginés par curseur (`before=<id>`), ordre chronologique dans la page.
- Lecture : `last_read_at` par participant ; `peerLastReadAt` permet d'afficher « Vu ». Effacement local : `cleared_at` par participant.
- Le hub (`platform/realtime`) garde les connexions par utilisateur ; les services publient `message.created`, `conversation.read`, `match.created`, `match.removed`, `notification.created`. Les clients lents sont déconnectés plutôt que de bloquer l'émetteur.
- Le client (`src/api/realtime.ts`) s'authentifie par premier message, se reconnecte avec backoff et rafraîchit son token sur fermeture 4001 ; `src/state/realtime.tsx` applique les événements au cache React Query.

## 6. Frontend (`apps/mobile`)

- `api/` : seul endroit qui fait du réseau. `client.ts` gère timeout, erreurs typées et refresh unique partagé (single-flight).
- `state/` : session (stockage sécurisé), hooks React Query, pont temps réel.
- `navigation/` : pile non connectée, onboarding (décidé une fois par session), application (onglets + écrans modaux).
- `features/*` : écrans par domaine ; composants partagés dans `ui/` (thème clair / sombre via `useTheme`).
- Les images passent par `ui/Photo` (`expo-image`, cache mémoire + disque, URLs signées stables pendant une heure).

## 7. Points d'extension

| Besoin | Où brancher |
| --- | --- |
| Nouvel algorithme de recommandation | `matching.Ranker` |
| Stockage objet (S3, GCS) | `storage.Store` |
| Plusieurs instances API | `realtime.Publisher` (Redis / LISTEN-NOTIFY), `ratelimit` |
| Notifications push natives | `notifications.Service.Notify` |
| Back-office de modération | table `reports` (statuts `open` / `reviewed` / `dismissed`), `profiles.moderation_hidden` |
