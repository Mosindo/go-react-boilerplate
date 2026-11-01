# Architecture d'Alma

## Vue d'ensemble

```text
Mobile (Expo) ──REST──▶ API Gin ──pgxpool──▶ PostgreSQL
      └─────WebSocket──▶ hub en mémoire (événements poussés)
                          └── disque local (photos, hors racine web)
```

Un seul serveur HTTP (`net/http` + Gin), une seule pool de connexions partagée. Pas d'ORM.

## Backend (`services/api`)

`handler -> service -> repository` par feature ; pas de SQL dans les handlers ; préoccupations transverses dans
`internal/platform`.

| Feature | Responsabilité |
| --- | --- |
| `auth` | comptes, JWT (15 min) + refresh token rotatif (30 j, hash SHA-256 en base), récupération de mot de passe, suppression de compte |
| `profiles` | profil, réglages (préférences + confidentialité), catalogue d'intérêts, profil public |
| `photos` | traitement d'image, stockage disque, ordre, accès contrôlé |
| `matching` | flux de découverte, swipes, matchs, annulation de match |
| `chat` | conversations, messages (pagination keyset), lecture, suppression locale |
| `moderation` | blocages et signalements |
| `notifications` | liste / compteur / lecture (créées par `matching` et `chat` dans leurs transactions) |

`platform` : `config`, `db` (migrations embarquées, table `schema_migrations`, verrou advisory), `middleware`
(JWT HS256 strict, CORS, en-têtes de sécurité, rate limiting, limite de corps), `realtime` (hub + endpoint `/ws`),
`geo`, `mailer` (log en dev, SMTP en prod), `httpx` (erreurs JSON, pagination, UUID), `logger`.

### API (résumé)

| Méthode | Route | Rôle |
| --- | --- | --- |
| POST | `/auth/register` `/auth/login` `/auth/refresh` `/auth/logout` | sessions |
| POST | `/auth/password/forgot` `/auth/password/reset` | récupération |
| GET / DELETE | `/me` | compte / suppression (mot de passe requis) |
| GET / PUT | `/me/profile` `/me/settings` | profil, préférences + confidentialité |
| GET | `/interests` | catalogue |
| GET POST PUT DELETE | `/me/photos[...]`, `/photos/:id` | gestion et lecture des photos |
| GET | `/discover?limit=` | lot de profils (≤ 20) |
| GET | `/profiles/:id` | profil complet d'un autre membre |
| POST | `/profiles/:id/like` `/profiles/:id/pass` | décision |
| GET / DELETE | `/matches`, `/matches/:id` | liste paginée / annulation |
| GET | `/conversations` | liste (dernier message + non-lus, une requête) |
| GET / POST | `/conversations/:id/messages` | historique paginé / envoi |
| POST / DELETE | `/conversations/:id/read`, `/conversations/:id` | lu / suppression locale |
| POST / DELETE / GET | `/users/:id/block`, `/blocks` | blocage |
| POST | `/users/:id/report` | signalement |
| GET / POST | `/notifications`, `/notifications/read` | notifications |
| GET (WS) | `/ws` | temps réel (`ready`, `message`, `read`, `match`, `unmatch`) |

Erreurs : `{"error": "..."}` ; pagination : `limit` + `cursor` opaque (`nextCursor` vide = fin).

### Modèle de données (`001_init.sql`)

`users`, `sessions`, `password_resets`, `profiles`, `preferences`, `interests`, `user_interests`, `photos`, `swipes`
(table des likes/pass), `matches`, `conversations`, `conversation_participants`, `messages`, `blocks`, `reports`,
`notifications`.

- Toutes les clés étrangères sont `ON DELETE CASCADE` (suppression de compte cohérente), sauf `reports.reporter_id`
  (`SET NULL` : un signalement survit à la suppression de son auteur).
- Contraintes : email en minuscules unique, 18 ans minimum, genre/motifs/statuts par `CHECK`, `matches.user_a <
  user_b` + `UNIQUE(user_a, user_b)`, `UNIQUE(user_id, position)` différable pour les photos.
- Index : `users(email)`, `messages(conversation_id, created_at DESC, id DESC)`, `notifications(user_id, created_at
  DESC)` + partiel sur les non-lues, `conversation_participants(user_id, conversation_id)`, `profiles(latitude,
  longitude)` (pré-filtre géographique), `swipes(target_id, swiper_id) WHERE action='like'`, etc.

### Règles métier (toutes vérifiées côté serveur)

- **Éligibilité** (`eligibleFrom`) : le genre recherché, la tranche d'âge et la distance maximale s'appliquent **dans
  les deux sens** ; profil découvrable ; aucun blocage ; ni soi-même. La même condition protège `like`/`pass` : un
  client ne peut pas swiper un profil qu'il ne pourrait pas voir.
- **Swipe** : une décision par paire ordonnée (`PRIMARY KEY (swiper_id, target_id)`), répétition = 409.
- **Match** : transaction + `pg_advisory_xact_lock` sur la paire ; création du match, de la conversation, de ses deux
  participants et des notifications ; événements WebSocket après commit.
- **Chat** : seul un participant lit/écrit ; un tiers reçoit 404 (pas de fuite d'existence). Envoi refusé en cas de
  blocage. Suppression locale = `cleared_at` par participant (le message suivant fait réapparaître la conversation,
  sans les anciens messages).
- **Blocage** : supprime le match (donc conversation et messages), masque profils et photos dans les deux sens.
- **Photos** : types réels détectés par contenu (pas par extension), 8 Mo max, 40 Mpx max, ≥ 200 px, recompression
  JPEG 1080 px ; nom de fichier aléatoire validé par regex (pas de traversée de chemin).
- **Notifications de message** : au plus une notification non lue par conversation (anti-flood).

### Sécurité

JWT HS256 uniquement, expiration obligatoire, `uid`/`sid` requis ; bcrypt ; mot de passe 8-72 caractères avec lettres
et chiffres ; temps de réponse égalisé si l'email est inconnu ; rate limiting (auth/IP, swipes, messages, uploads,
signalements) ; corps JSON ≤ 1 Mo ; en-têtes de sécurité ; logs expurgés des secrets ; `reports` conservés après
suppression du plaignant ; aucun secret côté mobile (seul `EXPO_PUBLIC_API_URL`, public par nature).

## Mobile (`apps/mobile`)

- `src/api` : client HTTP unique (refresh single-flight sur 401, erreurs typées) — aucun `fetch` dans les écrans.
- `src/realtime` : client WebSocket (reconnexion exponentielle, resynchronisation au retour au premier plan) et
  `RealtimeProvider` qui met à jour le cache TanStack Query.
- `src/state` : session (tokens dans `expo-secure-store`) et hooks de requêtes.
- `src/components` : design system (Alma : crème / aubergine / rose terracotta, clair + sombre), `SwipeDeck`
  (PanResponder + boutons), gestion des photos, formulaires de préférences.
- `src/lib` : logique pure testée avec Vitest (dates 18+, validation, fusion de messages, backoff).
