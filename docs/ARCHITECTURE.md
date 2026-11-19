# Architecture de Lueur

## Vue d'ensemble

```text
App Expo (iOS / Android / web)
  │  REST JSON (Bearer access token)      WebSocket /realtime (ticket à usage unique)
  ▼                                        ▼
API Go (Gin) ── features ── platform ── PostgreSQL (pool pgx unique)
                     │
                     └── stockage photos (UPLOAD_DIR) servi par URLs signées
```

## Backend (`services/api`)

- `cmd/api` : charge la config, se connecte (retry), applique les migrations, démarre un `http.Server` (Gin) avec arrêt propre.
- `cmd/seed` : données de démonstration (refusé en production) et purge.
- `internal/app` : composition root (`app.New`) partagée par `cmd/api` et les tests d'intégration ; câble les features, les middlewares et les limites de débit.
- `internal/features/<feature>` : `model.go`, `repository.go` (SQL uniquement), `service.go` (règles métier), `handler.go` (HTTP fin), `routes.go`.

| Feature | Responsabilité |
| --- | --- |
| `auth` | inscription, connexion, refresh avec rotation, logout, récupération par code, `/me`, suppression du compte, validation des sessions pour le middleware |
| `profiles` | profil, préférences, intérêts, position arrondie, cartes publiques chargées par lot (3 requêtes quel que soit le nombre de profils), règle de visibilité |
| `photos` | upload / remplacement / ordre / suppression, service des médias signés |
| `matching` | candidats de découverte, `Ranker`, like/pass, création atomique du match et de sa conversation, likes reçus, annulation de match |
| `chat` | liste des conversations, messages paginés par curseur, envoi, lecture, suppression locale |
| `notifications` | création (messages regroupés), liste, compteurs, lecture |
| `safety` | blocages, signalements (qui bloquent aussi) |
| `realtime` | ticket + endpoint WebSocket |

`internal/platform` : `config`, `db` (pool + migrations suivies), `middleware` (JWT + session, CORS, en-têtes de sécurité, limite de corps, rate limiting, logs), `httpx` (erreurs homogènes, validation d'ID/pagination), `geo`, `media` (traitement d'image, stockage, signature), `mailer`, `realtime` (hub), `logger` (journalisation avec masquage des secrets).

Les dépendances entre features passent par des interfaces déclarées côté consommateur (`CardLoader`, `MatchNotifier`, `MessageNotifier`, `realtime.Publisher`, `AccountHooks`).

## Modèle de données

| Table | Clés et contraintes notables | Index |
| --- | --- | --- |
| `users` | e-mail en minuscules, unique ; `is_demo` | `email` (unique), `is_demo` partiel |
| `sessions` | hash SHA-256 du refresh token, révocation | `token_hash` unique, `(user_id, created_at)` |
| `password_resets` | HMAC du code, tentatives, expiration, usage unique | `(user_id, created_at)` |
| `profiles` | 1-1 avec `users` ; genre/intention en liste fermée ; longueurs bornées ; lat/lng arrondies, toutes deux nulles ou non | `(gender, birthdate)`, `(latitude, longitude)` partiel, `last_active_at` |
| `preferences` | genres recherchés ⊂ {woman, man, nonbinary}, 18 ≤ âge_min ≤ âge_max ≤ 99, distance 1–500 km | PK |
| `interests` / `user_interests` | catalogue / association | PK + `interest_id` |
| `photos` | position 0–5, unicité `(user_id, position)` différée (réordonnancement atomique), `storage_key` unique | via contrainte |
| `swipes` | PK `(swiper_id, target_id)` ⇒ une seule décision par paire ; pas de swipe sur soi | `(target_id, decision, created_at)` |
| `matches` | paire ordonnée `user_a_id < user_b_id`, unique ; `unmatched_at` | `user_b_id` (+ unicité couvrant `user_a_id`) |
| `conversations` | 1-1 avec `matches`, `last_message_at` | `match_id` unique |
| `conversation_participants` | PK `(conversation_id, user_id)`, `last_read_at`, `cleared_at` | `(user_id, conversation_id)` |
| `messages` | 1–2000 caractères | `(conversation_id, created_at DESC, id DESC)` |
| `blocks` | PK `(blocker_id, blocked_id)` | `(blocked_id, blocker_id)` |
| `reports` | motif en liste fermée, statut de modération ; `ON DELETE SET NULL` pour conserver l'historique | `(reported_id, created_at)`, `(status, created_at)` |
| `notifications` | types `match` / `message` | `(user_id, created_at DESC, id DESC)`, non-lus partiel |

Toutes les données personnelles sont en `ON DELETE CASCADE` depuis `users`.

## Règles métier clés (toutes vérifiées côté serveur)

- **Découverte** : préférences de genre et d'âge **mutuelles**, distance dans le rayon des deux personnes (si le demandeur a une position ; préfiltre par boîte englobante indexée puis haversine), profils découvrables avec au moins une photo, jamais soi-même, jamais un profil déjà traité, jamais un blocage dans un sens ou l'autre, comptes de démo exclus en production. Le dépôt renvoie un lot borné (≤ 100), classé ensuite par `Ranker`.
- **Swipe** : verrou consultatif transactionnel par paire → insertion du swipe (conflit = 409) → si réciprocité, insertion du match (unique) + conversation + participants dans la même transaction. Deux likes simultanés produisent exactement un match (test d'intégration dédié).
- **Accès au chat** : une seule requête de « membership » vérifie que l'utilisateur participe, que le match est actif et qu'aucun blocage n'existe ; un non-participant reçoit 404 (indiscernable d'une conversation inexistante).
- **Lu / non-lu** : un horodatage `last_read_at` par participant ; un message est « lu » s'il est antérieur au marqueur du partenaire.
- **Visibilité d'un profil** : pas de blocage, et match actif ou profil découvrable avec photo. Les réponses ne contiennent jamais e-mail, date de naissance ni coordonnées.

## Sécurité

- JWT HS256 uniquement (`WithValidMethods`), expiration obligatoire, claims `uid`/`sid` requis, session vérifiée à chaque requête.
- Mots de passe bcrypt (coût 12), comparaison à temps constant y compris pour un e-mail inconnu ; politique : 8–72 caractères, lettre + chiffre.
- Récupération : code à 6 chiffres HMAC, 15 min, 5 essais, 3 demandes/h, réponse identique que le compte existe ou non, envoi asynchrone, toutes les sessions révoquées après réinitialisation.
- Rate limiting par IP (identifiants, récupération) et par utilisateur (uploads, swipes, messages, signalements) ; `X-Forwarded-For` uniquement depuis `TRUSTED_PROXIES`.
- Corps JSON limités à 64 Ko, uploads à 8 Mo ; validation de tous les identifiants (UUID), paginations bornées, textes nettoyés (caractères de contrôle).
- Photos : détection du type par contenu, limite de pixels, ré-encodage (métadonnées supprimées), clés aléatoires validées par regex (pas de traversée de chemin), URLs HMAC expirantes, `Content-Security-Policy: default-src 'none'`.
- En-têtes : `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, HSTS derrière HTTPS ; CORS sur liste blanche.
- Logs JSON avec masquage des tokens, mots de passe et secrets.

## Frontend (`apps/mobile`)

- `src/api` : `client.ts` (timeouts, erreurs typées et traduites, refresh « single-flight » sur 401), `endpoints.ts` (seul point d'accès réseau), `tokenStore.ts` (SecureStore natif / sessionStorage web).
- `src/state` : `session.tsx` (statut, utilisateur, profil), `realtime.tsx` (WebSocket avec reconnexion exponentielle, application des événements au cache TanStack Query, resynchronisation REST à chaque connexion), `matchCelebration.tsx`.
- `src/navigation/RootNavigator.tsx` : non connecté → pile d'auth ; profil absent/incomplet → onboarding ; sinon onglets (Découvrir, Likes, Messages, Profil) + écrans empilés.
- `src/features` : un dossier par domaine, sans logique réseau directe dans les composants.
- `src/ui` + `src/theme` : design system thémé (clair/sombre), composants accessibles, `ActionSheet` et dialogues compatibles web (où `Alert` n'existe pas).

Performance : images via `expo-image` (cache mémoire + disque, clé de cache stable malgré la signature), listes virtualisées et paginées par curseur, cartes de découverte préchargées par lots de 10, invalidations ciblées plutôt que rechargements globaux, aucune liste chargée en entier.
