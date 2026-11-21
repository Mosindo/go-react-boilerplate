# Architecture de Lueur

Ce document décrit l'organisation du dépôt, le modèle de données, les flux principaux et les choix de sécurité. Il complète le [README](../README.md).

## 1. Vue d'ensemble

```text
 Client Expo (iOS / Android / Web)                     API Go (Gin)                         PostgreSQL
 ┌───────────────────────────────┐   HTTPS (REST)   ┌───────────────────────────┐   pgxpool   ┌───────────┐
 │ features/* (écrans)           │ ───────────────▶ │ middleware (auth, CORS,   │ ──────────▶ │ tables    │
 │ api/* (client + endpoints)    │                  │   rate limit, en-têtes)   │             │ + index   │
 │ React Query (cache)           │ ◀─── WSS ─────── │ features: handler→service │             └───────────┘
 │ realtime/ (WebSocket)         │   /realtime      │   →repository             │   disque / S3
 └───────────────────────────────┘                  │ realtime hub, storage     │ ──────────▶ photos
                                                    └───────────────────────────┘
```

- Un seul serveur HTTP Gin (pas de mélange `net/http` / Gin pour le routage), un seul pool de connexions partagé.
- Les handlers sont fins (parsing / mapping HTTP), la logique métier vit dans les services, le SQL uniquement dans les repositories.

## 2. Backend (`services/api`)

```text
cmd/api/main.go          composition : config, pool, migrations, storage, mailer, hub, routes
cmd/seed/main.go         données de démonstration (refusé en production)
internal/platform/
  config/                variables d'environnement validées
  db/                    connexion + migrations embarquées (schema_migrations, verrou consultatif)
  middleware/            RequireUser (JWT HS256 strict + session active), CORS, en-têtes de sécurité,
                         limite de taille JSON, rate limiter, logs/metrics
  httpx/                 erreurs JSON homogènes, pagination, validation d'UUID
  storage/               interface Storage + implémentation disque (anti path traversal, écriture atomique)
  media/                 URLs de photos signées HMAC et expirantes
  realtime/              hub WebSocket en mémoire (interface Publisher)
  mailer/                SMTP (STARTTLS), journal en développement, désactivé sinon
  geo/                   arrondi des coordonnées, distance, boîte englobante
internal/features/
  auth/                  inscription, connexion, refresh, logout, récupération, suppression de compte
  profiles/              profil, préférences, intérêts, position, réglages, cartes publiques
  photos/                pipeline d'images, CRUD, ordre, service des médias
  discovery/             filtrage SQL + Ranker (score)
  matching/              swipes, matchs, annulation
  chat/                  conversations liées aux matchs, messages, lecture, effacement local
  safety/                blocages, signalements, suspension automatique
  notifications/         notifications persistées + poussées en temps réel
  users/ posts/ comments/ files/   modules génériques (ENABLE_COMMUNITY_MODULES)
```

Dépendances entre features : `photos`, `discovery`, `matching`, `chat` et `safety` utilisent `profiles.Service` pour construire les cartes publiques (chargement groupé, sans N+1) ; `matching` et `chat` utilisent `notifications.Service` ; tout ce qui pousse des événements dépend de l'interface `realtime.Publisher`.

## 3. Modèle de données

| Table | Contenu | Contraintes / index notables |
| --- | --- | --- |
| `users` | e-mail (unique, normalisé), hash bcrypt, organisation | index `email` |
| `sessions` | refresh tokens hachés (SHA-256), expiration, révocation | unique `token_hash` |
| `password_reset_codes` | code HMAC, tentatives (max 5), expiration 15 min | |
| `profiles` | prénom, date de naissance, genre, bio, ville, métier, taille, objectif, **lat/lng arrondies**, `is_visible`, `show_distance`, `is_suspended`, `is_demo`, `last_active_at` | `CHECK` sur chaque champ, index `(gender, birthdate)` partiel, index `(latitude, longitude)` |
| `preferences` | genres recherchés, âge min/max, distance max | `CHECK` 18–99, 1–500 km |
| `photos` | clés de stockage aléatoires (pleine taille + miniature), position 0–5 | unique `(user_id, position)` différable |
| `interests` / `user_interests` | catalogue (40 entrées) / sélection (≤ 10) | PK `(user_id, interest_id)` |
| `swipes` | like ou pass, définitif | PK `(actor, target)`, `CHECK actor <> target`, index `(target, action, actor)` |
| `matches` | paire ordonnée `(user_low_id, user_high_id)` | unique, `CHECK low < high` |
| `conversations` | 1 par match (`match_id` unique, `ON DELETE CASCADE`) | |
| `conversation_participants` | `last_read_at` (lu/non lu), `cleared_at` (effacement local) | PK `(conversation, user)` |
| `messages` | contenu ≤ 2000 caractères | index `(conversation_id, created_at)` |
| `blocks` | bloqueur → bloqué | PK + index inverse |
| `reports` | motif, détails, extrait de conversation (`evidence`), statut | unique (rapporteur, signalé) tant que « open », FK `ON DELETE SET NULL` |
| `notifications` | type, titre, corps, `data` JSON, lu | index `(user_id, created_at)`, index partiel non-lues |

Suppression de compte : la ligne `users` est supprimée et toutes les données personnelles suivent par `ON DELETE CASCADE` (profil, préférences, photos, intérêts, swipes, matchs → conversations → messages, notifications, sessions, blocages). Les fichiers photo sont supprimés du stockage. Les signalements sont conservés pour la modération avec la référence anonymisée.

## 4. Flux principaux

### Découverte

1. `LoadViewer` : profil + préférences du membre (409 si profil incomplet).
2. `ListCandidates` (une requête) : même tenant, visible, non suspendu, au moins une photo, genre et âge compatibles **dans les deux sens**, distance ≤ rayon des deux membres (préfiltre par boîte englobante puis `haversine_km`), jamais swipé par le membre, aucun blocage dans un sens ou l'autre, hors `exclude` (cartes déjà en main côté client). Pool de 100 candidats.
3. `Ranker.Rank` (Go) : score = a liké (40) + intérêts communs (6 × ≤5) + activité récente (25 × demi-vie 72 h) + proximité (15) + complétude (8) + nouveau membre (6). Départage déterministe.
4. Chargement groupé des cartes (3 requêtes : profils, photos, intérêts) et signature des URLs.

### Like → match

Transaction : verrou consultatif sur la paire, vérification que l'auteur a un profil complet et que la cible est un candidat légitime (même tenant, visible, non bloqué), insertion du swipe (`ON CONFLICT DO NOTHING` → 409 si déjà traité), puis si like réciproque : création du match, de la conversation et des participants. Après commit : événement `match.new` et notification pour les deux membres.

### Chat

Chaque appel résout la conversation via un **match actif sans blocage** ; sinon 404. Envoi → message + `updated_at` + `last_read_at` de l'expéditeur, puis `message.new` (aux deux membres) et notification regroupée pour le destinataire. Lecture → `last_read_at`, notifications de messages marquées lues, `message.read` à l'autre membre. Pagination par curseur (`before=<messageId>`).

### Temps réel

`GET /realtime` (WebSocket) : le client envoie `{"type":"auth","token":"…"}` en premier message (délai 10 s), le serveur vérifie le JWT **et** la session, répond `ready`, puis pousse les événements. Pings toutes les 25 s, fermeture (code 4401) à l'expiration du jeton ; le client rafraîchit son jeton et se reconnecte avec backoff exponentiel, puis invalide ses caches pour rattraper ce qu'il a manqué.

## 5. Client (`apps/mobile`)

- `src/api/client.ts` : `fetch` avec délai, erreurs typées (`ApiError`), rafraîchissement **unique et partagé** du jeton sur 401 puis nouvel essai ; messages d'erreur traduits.
- `src/store/tokenStore.ts` : jetons dans le trousseau (`expo-secure-store`) sur mobile, `sessionStorage` sur le web.
- `src/features/auth/AuthProvider.tsx` : restauration de session, connexion / inscription / déconnexion, expiration.
- `src/navigation/RootNavigator.tsx` : trois espaces — non connecté, onboarding (décidé une fois par session pour ne pas interrompre le parcours), application (onglets Découvrir, Matchs, Activité, Profil + écrans empilés).
- `src/realtime/RealtimeProvider.tsx` : applique les événements au cache React Query (ajout de message, invalidations), toasts pour les messages hors conversation ouverte.
- `src/theme` + `src/ui` : jetons clair/sombre, typographie (serif pour les titres), composants accessibles (rôles, labels, états, cibles ≥ 44 px), feuilles d'actions et confirmations compatibles web.
- Découverte : `PanResponder` + `Animated` (aucune dépendance de geste), actions d'accessibilité « J'aime » / « Passer » / « Voir le profil », préchargement des images suivantes.

## 6. Sécurité (résumé)

| Menace | Mesure |
| --- | --- |
| Vol / rejeu de jeton | Accès 15 min, refresh rotatif haché, vérification de session à chaque requête, révocation à la réinitialisation |
| Falsification JWT | HS256 uniquement, expiration obligatoire, claims stricts, secret ≥ 32 caractères |
| Brute force / abus | Rate limiting par IP (auth) et par utilisateur (messages, uploads, signalements, swipes), proxies de confiance explicites, codes de récupération limités à 5 essais |
| Énumération de comptes | Réponse identique à « mot de passe oublié », temps de connexion égalisé, 404 identique pour toute conversation inaccessible |
| Accès non autorisé | Toutes les règles vérifiées côté serveur (match requis, blocages, tenant, propriété des photos) |
| Fuite de localisation | Arrondi ~1 km, coordonnées jamais renvoyées, distance arrondie ≥ 1 km, option de masquage, EXIF supprimé |
| Upload dangereux | Détection par contenu, formats limités (JPEG/PNG/WebP), limites de taille et de pixels, ré-encodage, noms aléatoires, `nosniff`, Content-Type forcé |
| Exposition des médias | URLs signées HMAC liées à la photo, la variante et l'expiration |
| Payloads excessifs | Limite JSON 1 Mo, upload 10 Mo, messages WebSocket 4 Ko |
| Harcèlement | Blocage silencieux, signalement + blocage, preuve conservée, suspension automatique |
