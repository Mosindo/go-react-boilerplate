# Architecture de Lueur

Ce document décrit l'organisation du code, le modèle de données et les flux critiques. Le [README](../README.md) couvre l'installation et les décisions produit.

## Vue d'ensemble

```text
App mobile (Expo)  ──REST JSON──▶  API Go (Gin)  ──pgxpool──▶  PostgreSQL
        ▲                             │
        └────── WebSocket (events) ───┘──▶ stockage photos (disque / volume)
```

Un seul serveur HTTP Gin (pas de `net/http` mélangé), un seul pool PostgreSQL partagé, SQL explicite sans ORM.

## API (`services/api`)

### Racine de composition

`internal/app/app.go` construit le moteur Gin : middlewares globaux (recovery, en-têtes de sécurité, CORS, request id, métriques, logs, limite globale par IP), services de plateforme et features. `cmd/api/main.go` charge la configuration, applique les migrations et sert l'application avec arrêt gracieux. Les tests d'intégration utilisent exactement la même fonction `app.New`.

### Plateforme (`internal/platform`)

| Paquet | Rôle |
| --- | --- |
| `config` | Lecture et validation des variables d'environnement |
| `db` | Connexion avec retry, migrations embarquées suivies dans `schema_migrations` |
| `authtoken` | Signature et validation stricte des JWT HS256 (`iss`, `sub`, `exp`, `typ`) |
| `middleware` | Authentification (avec vérification de session en base), limitation de débit, sécurité, logs |
| `httpx` | Format d'erreur unique, parsing strict de la pagination et des identifiants |
| `realtime` | Hub WebSocket en mémoire, interface `Publisher` |
| `storage` | Interface `Store` + implémentation disque (clés aléatoires, anti path traversal) |
| `mediaurl` | URLs de photos signées HMAC et expirantes |
| `mailer` | SMTP (STARTTLS) ou journalisation en développement |
| `logger` | Logs JSON avec masquage des secrets |

### Features (`internal/features`)

Chaque feature contient `handler.go` (HTTP uniquement), `service.go` (règles métier, validation), `repository.go` (SQL), `model.go`, `routes.go`. Les dépendances entre features passent par des interfaces déclarées côté consommateur :

- `auth` : comptes, sessions, récupération, changement de mot de passe, suppression de compte (dépend de `AccountPurger` implémenté par `photos` et de `Disconnecter` implémenté par le hub).
- `profiles` : profil, préférences, localisation grossière, centres d'intérêt, visibilité ; fournit `Cards`, l'hydratation en lot (3 requêtes quel que soit le nombre de profils) utilisée par la découverte, les likes et le chat.
- `photos` : upload, traitement d'image, ordre, remplacement, suppression, service des médias signés.
- `matching` : découverte (`Candidates` en SQL + `Ranker`), swipes, matchs, likes reçus, unmatch.
- `chat` : conversations, messages paginés par curseur, accusés de lecture, effacement local.
- `notifications` : notifications in-app regroupées, publiées en temps réel.
- `safety` : blocages et signalements (seuil de revue).

## Modèle de données

| Table | Points clés |
| --- | --- |
| `users` | e-mail unique normalisé (contrainte), `is_demo`, `moderation_status` (`active`, `under_review`, `suspended`), `last_active_at` |
| `sessions` | hash SHA-256 du refresh token (unique), expiration, révocation |
| `password_reset_tokens` | hash du code, essais, expiration, usage unique |
| `profiles` | 1–1 avec `users` ; contraintes : prénom 1–40, genre, bio ≤ 500, âge ≥ 18 ans, taille, objectif, coordonnées appariées et bornées ; index partiel `(gender, birthdate) WHERE discoverable`, index `(location_lat, location_lng)` |
| `preferences` | genres recherchés (1–3 valeurs connues), âges 18–99 ordonnés, distance 1–500 km |
| `interests`, `user_interests` | catalogue de référence (58 entrées) + table de liaison, index inverse `(interest_id, user_id)` |
| `photos` | position 0–5 unique par membre (contrainte différée pour réordonner), clé de stockage unique, JPEG uniquement |
| `swipes` | PK `(swiper_id, target_id)` : une seule décision par paire ; index partiel des likes reçus |
| `matches` | paire canonique `user_a_id < user_b_id`, unique ; index par membre |
| `conversations` | 1–1 avec un match (`ON DELETE CASCADE`), `last_message_at` |
| `conversation_participants` | PK `(conversation_id, user_id)`, `last_read_at`, `cleared_at` ; index `(user_id, conversation_id)` |
| `messages` | 1–2000 caractères, index `(conversation_id, created_at DESC, id DESC)` |
| `blocks` | PK `(blocker_id, blocked_id)`, index inverse |
| `reports` | motif énuméré, détails ≤ 1000, statut, index par signalé et index partiel des signalements ouverts |
| `notifications` | type énuméré, index `(user_id, created_at DESC)`, index partiel des non lues, unicité partielle `(user_id, dedupe_key)` pour le regroupement |

Toutes les clés étrangères vers `users` sont en `ON DELETE CASCADE` : supprimer un compte efface toutes ses données.

## Flux critiques

### Authentification

1. `register`/`login` → session en base + access token (15 min) + refresh token (30 j, stocké haché).
2. Chaque requête : JWT validé strictement **puis** session vérifiée en base (révocation immédiate).
3. `refresh` fait tourner le refresh token de façon atomique ; un ancien token ne fonctionne plus.
4. Le client mobile rafraîchit automatiquement et une seule fois en cas de 401 (`apiRequest`).

### Découverte et matching

1. `GET /discovery` charge le contexte du membre (profil + préférences + photo) puis un lot de candidats éligibles en SQL (préférences mutuelles, bounding box + haversine, ni vus ni bloqués, compte actif avec photo).
2. Le `Ranker` ordonne le lot ; les cartes sont hydratées en 3 requêtes.
3. `POST /swipes` : verrou consultatif par paire, vérification d'éligibilité, insertion unique, détection de réciprocité, création du match, de la conversation et des participants dans la même transaction.
4. Notifications + événements `match.created` aux deux membres.

### Chat

Envoi : contrôle d'appartenance, insertion, mise à jour de `last_message_at` et du curseur de lecture de l'expéditeur, événement `message.created` aux deux participants, notification regroupée au destinataire. Lecture : curseur positionné sur le dernier message existant (et non sur `NOW()`), événement `conversation.read` vers l'autre membre.

### Temps réel

`POST /realtime/ticket` délivre un JWT `typ=realtime` de 60 s ; `GET /realtime?ticket=` vérifie le ticket et la session puis ouvre la WebSocket. Le hub borne le nombre de connexions par membre, déconnecte les clients lents et ferme les connexions à la suppression du compte ou à la réinitialisation du mot de passe. Côté mobile, `RealtimeProvider` injecte les événements dans le cache React Query et les écrans repassent en polling si la connexion est perdue.

## Application mobile (`apps/mobile`)

- **Navigation** : non connecté → pile d'authentification ; connecté sans profil complet → onboarding en 5 étapes ; sinon onglets (Découvrir, Likes, Matchs, Profil) + écrans empilés (profil détaillé, chat, édition, réglages, signalement…).
- **Données** : tous les appels réseau passent par `src/api/client.ts` (aucun `fetch` dans l'UI). TanStack Query gère cache, états de chargement et invalidations ; clés centralisées dans `providers/queryClient.ts`.
- **Jetons** : stockés dans le trousseau via `expo-secure-store` (mémoire uniquement sur le web).
- **Design system** : `src/theme` (palettes claire/sombre, typographie, espacements) et `src/ui` (Button, TextField, Chip, Stepper, ActionSheet, Toast, états vides/erreurs/chargement…).
- **Performance** : images mises en cache disque (`expo-image`), listes virtualisées, pagination par curseur des messages, lot de découverte préchargé avant épuisement, mises à jour ciblées du cache plutôt que des rechargements complets.
