# Architecture de Lueur

## Vue d'ensemble

```text
┌──────────────────────────┐  HTTPS (REST, JSON)   ┌──────────────────────────────┐
│ App Expo (iOS/Android/Web)│ ────────────────────▶ │ API Go (Gin)                  │
│  React Query (cache)      │ ◀──── WebSocket ───── │  features/* (handler→service→ │
│  SecureStore (jetons)     │   événements temps    │  repository) + platform/*     │
└──────────────────────────┘   réel                └──────────────┬───────────────┘
                                                                   │ pgxpool (pool unique)
                                                     ┌─────────────▼──────────┐   ┌───────────────┐
                                                     │ PostgreSQL              │   │ Stockage photos│
                                                     │ (migrations embarquées) │   │ (UPLOAD_DIR)   │
                                                     └─────────────────────────┘   └───────────────┘
```

Un seul serveur HTTP Gin (pas de `net/http` mixte), un seul pool de connexions, une seule composition des dépendances dans `cmd/api/app.go` (`newRouter`), réutilisée par les tests d'intégration.

## Backend

### Plateforme (`internal/platform`)

| Paquet | Rôle |
| --- | --- |
| `config` | Lecture et validation de l'environnement (secret JWT fort, SMTP obligatoire en production) |
| `db` | Connexion avec retry, migrations versionnées (`schema_migrations`, verrou consultatif, refus d'un schéma legacy) |
| `middleware` | En-têtes de sécurité, CORS, request id, logs, métriques, limite de corps (1 Mo JSON), rate limiting, authentification JWT + vérification de session |
| `httpx` | Enveloppe d'erreur unique `{error, code}`, pagination bornée, validation des UUID |
| `realtime` | Hub WebSocket en mémoire (`Publisher`), authentification par premier message, ping, 5 connexions max par utilisateur |
| `media` | URL signées HMAC-SHA256 pour les photos (clé dérivée du secret JWT, expiration par tranche d'heure) |
| `storage` | Interface `Storage` + implémentation disque (clés validées, écriture atomique, aucun chemin fourni par l'utilisateur) |
| `mailer` | Interface `Mailer` : SMTP (`net/smtp`, STARTTLS) ou journalisation en développement |
| `logger` | Logs JSON avec masquage des jetons et mots de passe |

### Fonctionnalités (`internal/features`)

| Paquet | Responsabilité | Dépend de |
| --- | --- | --- |
| `auth` | Comptes, sessions, réinitialisation, suppression | `mailer`, hook de suppression (photos + temps réel) |
| `profiles` | Profil, préférences, position, confidentialité, centres d'intérêt, cartes publiques hydratées en 3 requêtes | `media` |
| `photos` | Pipeline d'upload, ordre, remplacement, service des fichiers | `storage`, `media` |
| `matching` | Découverte, likes/passes, création de match, annulation | `profiles` (cartes), `notifications`, `realtime` |
| `chat` | Conversations, messages, lu/non-lu, effacement local | `profiles`, `notifications`, `realtime` |
| `notifications` | Notifications match/message (regroupées), lecture | `realtime` |
| `safety` | Blocages, signalements, suspension automatique | `realtime` |

Les dépendances entre fonctionnalités passent par de petites interfaces déclarées côté consommateur (`CardProvider`, `MatchNotifier`, `MessageNotifier`, `realtime.Publisher`), ce qui garde les paquets découplés et testables.

### Flux clés

**Like → match**
1. `POST /swipes` : le service vérifie l'action et que le profil de l'auteur est complet.
2. Le repository ouvre une transaction, prend `pg_advisory_xact_lock(paire)`, vérifie que la cible est disponible (profil + photo, non suspendue, pas de blocage), insère le swipe (`ON CONFLICT DO NOTHING` → 409 si déjà traité).
3. Si like et like réciproque existant : insertion du match (paire ordonnée, unique) et des deux participants, dans la même transaction.
4. Après commit : notifications aux deux membres, événement `match.created` à la cible ; l'auteur reçoit le match dans la réponse.

**Message**
1. `POST /conversations/:matchId/messages` : normalisation (1–2000 caractères, caractères de contrôle retirés), contrôle d'appartenance (`conversation_participants`, sinon 404).
2. Transaction : insertion, `matches.last_message_at`, lecture implicite pour l'expéditeur.
3. Événement `message.created` aux deux membres (multi-appareils) ; notification « message » mise à jour plutôt que dupliquée tant qu'elle n'est pas lue.

**Découverte** (`matching/repository.go`, `Candidates`) : filtres mutuels (genre recherché des deux côtés, tranches d'âge des deux côtés, distance maximale des deux côtés si les deux positions sont connues, avec pré-filtre par latitude), exclusion des profils masqués, suspendus, bloqués (dans les deux sens), sans photo ou déjà traités ; tri par score (`a liké l'utilisateur` +3, intérêts communs +0,8 chacun jusqu'à 5, activité récente +1,5/+0,5, distance −1 par 100 km), puis identifiant pour la stabilité. Lots de 10 à 30, sans offset : les profils traités sortent naturellement du résultat.

## Modèle de données

| Table | Clés et contraintes principales | Index |
| --- | --- | --- |
| `users` | e-mail unique en minuscules, `is_demo`, `suspended_at`, `last_active_at` | `users(email)` unique |
| `sessions` | refresh token haché (unique), expiration, révocation | `token_hash` unique, `user_id` |
| `password_resets` | code haché, tentatives, expiration, usage unique | `(user_id, created_at)` |
| `profiles` | PK `user_id`, genre ∈ {woman, man, nonbinary}, longueurs bornées, taille 100–250, lat/lng appariées et bornées | `(gender, birthdate) WHERE discoverable`, `(latitude, longitude)` |
| `preferences` | `interested_in` ⊂ genres (1–3), âges 18–99 ordonnés, distance 1–500 km | PK |
| `interests` / `user_interests` | catalogue (slug unique) / PK `(user_id, interest_id)`, 10 max (service) | `(interest_id, user_id)` |
| `photos` | type `image/jpeg`, position 0–5, `UNIQUE (user_id, position) DEFERRABLE` (réordonnancement atomique), clé de stockage unique | — |
| `swipes` | PK `(actor_id, target_id)`, action ∈ {like, pass}, pas d'auto-like | `(target_id, action, actor_id)` |
| `matches` | paire ordonnée `user_a_id < user_b_id`, unique | `user_b_id` (+ unique `(a, b)`) |
| `conversation_participants` | PK `(match_id, user_id)`, `last_read_at`, `cleared_at` | `(user_id, match_id)` |
| `messages` | corps 1–2000 caractères | `(match_id, created_at DESC, id DESC)` |
| `blocks` | PK `(blocker_id, blocked_id)`, pas d'auto-blocage | `(blocked_id, blocker_id)` |
| `reports` | motif contrôlé, statut, `ON DELETE SET NULL` | unique ouvert par paire, `(reported_id, status)` |
| `notifications` | type ∈ {match, message, system} | `(user_id, created_at DESC, id DESC)`, non-lus partiel |

Toutes les tables personnelles référencent `users(id) ON DELETE CASCADE` : supprimer l'utilisateur supprime toutes ses données en une instruction. Fonctions SQL : `distance_km` (haversine) et `age_years`.

## Application

- **Navigation** : `RootNavigator` choisit entre la pile d'authentification, l'onboarding (décidé une fois à partir de `GET /me/profile`, terminé uniquement par l'assistant) et l'application (onglets Découvrir / Matchs / Activité / Profil + écrans empilés).
- **Données** : React Query (cache, retry sur erreurs transitoires seulement), hooks dans `state/queries.ts`. Le client HTTP rafraîchit le jeton une seule fois même si plusieurs requêtes échouent en même temps, puis rejoue la requête.
- **Temps réel** : `realtime/socket.ts` (reconnexion exponentielle, rafraîchissement du jeton si refus) ; `RealtimeProvider` met à jour le cache (insertion de message, accusés de lecture, matchs, notifications) et affiche des toasts, sauf pour la conversation ouverte.
- **Swipe** : `Animated` + `PanResponder` (aucune dépendance native supplémentaire) ; les boutons déclenchent la même animation, donc l'écran reste utilisable sans geste (lecteurs d'écran, web).
- **Images** : `expo-image` (cache mémoire + disque, clé de cache = id de photo car les URL signées tournent).
- **Design system** : `theme/` (palettes claire/sombre, police serif pour les titres), `ui/` (composants accessibles). Aucune couleur codée en dur dans les écrans hors superpositions sur photo.

## Sécurité (récapitulatif)

- Mots de passe bcrypt (8–72 octets, lettre + chiffre), comparaison à temps constant y compris pour les e-mails inconnus.
- JWT HS256 uniquement (`alg` vérifié, `exp` obligatoire, `iat` validé), session vérifiée à chaque requête.
- Autorisation côté serveur sur chaque ressource (propriété des photos, appartenance aux conversations, visibilité des profils, blocages).
- Validation stricte des entrées (UUID, pagination bornée, longueurs, énumérations, coordonnées), corps JSON limités à 1 Mo, uploads à 10 Mo.
- Rate limiting par IP (identifiants, refresh) et par utilisateur (uploads, swipes, messages, signalements), `Retry-After` renvoyé.
- En-têtes `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, HSTS en HTTPS, CSP stricte sur les photos.
- Données minimales : aucune coordonnée ni date de naissance ni e-mail exposés aux autres membres ; métadonnées photo supprimées.
