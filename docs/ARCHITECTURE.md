# Architecture de Lueur

Ce document décrit l'organisation du dépôt, le modèle de données, l'API et les flux critiques, pour pouvoir étendre le produit sans casser ses frontières.

## 1. Monorepo

| Dossier | Rôle |
| --- | --- |
| `services/api` | API Go (Gin, PostgreSQL, pgxpool, JWT HS256, WebSocket) et commande `cmd/seed` |
| `apps/mobile` | Client Expo / React Native (iOS, Android, web) |
| `infra/` | Variante de `docker compose` |
| `scripts/` | Scripts de QA PowerShell (délèguent aux scripts Node de `apps/mobile/scripts`) |
| `docs/` | Documentation (ce fichier, notes d'agents) |

## 2. Backend

```text
cmd/api/main.go                 assemblage : config → pool → migrations → services → routes
cmd/seed                        comptes de démonstration (@seed.invalid), jamais en production
internal/platform
  config                        variables d'environnement validées
  db                            pool, migrations embarquées et suivies
  middleware                    auth JWT, CORS, en-têtes de sécurité, limite de corps, rate limiting, logs
  httpx                         erreurs JSON, validation d'UUID
  logger                        logs structurés avec masquage des secrets
  mailer                        SMTP ou journal (développement)
  realtime                      hub d'événements par utilisateur (Publisher)
internal/features/<feature>     handler.go · service.go · repository.go · model.go · routes.go
  auth · profiles · photos · discover · matches · chat · notifications · safety
```

Règles : un seul serveur HTTP (Gin) ; handlers fins ; logique métier dans les services ; SQL uniquement dans les repositories ; les dépendances entre features passent par de petites interfaces (`discover.Eligibility` consommée par `matches`, `profiles.CanView` consommée par `photos`).

### Responsabilités

| Feature | Rôle |
| --- | --- |
| `auth` | Inscription, connexion, rotation des jetons, déconnexion, récupération de mot de passe, suppression de compte |
| `profiles` | Profil, préférences, localisation arrondie, centres d'intérêt, vue publique (jamais de coordonnées ni de date de naissance) |
| `photos` | Upload validé et réencodé, ordre, remplacement, suppression, diffusion protégée |
| `discover` | **Unique** définition de l'éligibilité (préférences mutuelles, distance, blocages, déjà traités) et ordre de recommandation (`Strategy`) |
| `matches` | Like/pass atomique, création du match + conversation + notifications, liste, annulation |
| `chat` | Conversations, messages, lecture, masquage local, WebSocket `/ws` |
| `notifications` | Lecture et acquittement (créées par `matches` et `chat` dans la même transaction que l'événement) |
| `safety` | Blocages et signalements |

### Endpoints

Tous exigent `Authorization: Bearer <accessToken>` sauf ceux marqués ○. Les erreurs sont `{"error": "message"}`.

| Méthode | Route | Description |
| --- | --- | --- |
| GET | `/health` ○ | État de l'API et de la base |
| POST | `/auth/register` `/auth/login` `/auth/refresh` `/auth/logout` ○ | Session |
| POST | `/auth/password/forgot` `/auth/password/reset` ○ | Récupération de compte |
| GET / DELETE | `/me` | Compte courant / suppression (mot de passe requis) |
| GET | `/interests` | Centres d'intérêt disponibles |
| GET / PUT | `/profile` | Mon profil (création et mise à jour) |
| PUT | `/profile/preferences` `/profile/location` | Préférences / position (arrondie) |
| GET | `/profiles/:id` | Profil public d'un membre visible |
| GET / POST / PUT / DELETE | `/photos`, `/photos/order`, `/photos/:id` | Gestion de mes photos |
| GET | `/photos/:id/file?size=thumb` | Image protégée |
| GET | `/discover?limit=` | Prochains profils (1-20) |
| POST | `/likes` `{userId, action: like\|pass}` | Swipe ; `409` si déjà traité, `404` si non éligible |
| GET / DELETE | `/matches`, `/matches/:id` | Matchs / annulation |
| GET | `/conversations`, `/conversations/:id/messages?before=` | Conversations / historique paginé |
| POST | `/conversations/:id/messages` `/conversations/:id/read` | Envoyer / marquer lu |
| DELETE | `/conversations/:id` | Masquer pour moi |
| GET / POST | `/notifications`, `/notifications/read-all`, `/notifications/:id/read` | Notifications |
| GET / POST / DELETE | `/blocks`, `/blocks/:userId` | Blocages |
| POST | `/reports` | Signalement (`alsoBlock` optionnel) |
| GET | `/ws` ○ | WebSocket ; première trame `{"type":"auth","token":"…"}` |

Événements serveur → client : `ready`, `message`, `read`, `match`, `unmatch`, `notification`, `pong`.

Pagination : `limit` borné partout ; curseurs `before` (horodatage ou identifiant de message) pour les listes qui grandissent ; `/discover` n'a pas de curseur car chaque swipe retire le profil du flux.

## 3. Modèle de données

```text
users ─┬─ sessions            jeton de rafraîchissement haché, rotation
       ├─ password_resets     code haché, tentatives, expiration
       ├─ profiles            prénom, naissance, genre, bio, ville, lat/lon arrondies, visibilité
       ├─ preferences         genres recherchés, âge min/max, distance max (NULL = partout)
       ├─ user_interests ─── interests (référentiel)
       ├─ photos              position 0-5, image + miniature (bytea)
       ├─ likes               (from, to) unique, action like|pass
       ├─ matches             user_a < user_b, unique ; unmatched_at
       │    └─ conversations ─┬─ conversation_participants (last_read_at, hidden_at)
       │                      └─ messages (id identité, body 1-2000)
       ├─ blocks              (blocker, blocked) unique
       ├─ reports             motif, détails, statut open|reviewed|dismissed
       └─ notifications       match|message, data JSONB, read_at
```

Tout dépend de `users` en `ON DELETE CASCADE` : supprimer un utilisateur supprime ses données.

Index : `users(email)` unique, `profiles(latitude)`, `profiles(gender, birth_date)`, `photos(user_id, position)`, `likes(to_user_id, action)`, `matches(user_b)`, `conversation_participants(conversation_id, user_id)` (clé) et `(user_id, conversation_id)`, `messages(conversation_id, created_at, id)`, `notifications(user_id, created_at DESC)`, index unique partiel « une notification de message non lue par conversation », `reports(status, created_at)`.

Performance : aucune requête N+1 — la découverte, les matchs et les conversations chargent leurs profils en **une** requête (`PublicByIDs`, agrégats JSON) ; tous les endpoints de liste sont bornés.

## 4. Flux critiques

**Swipe → match** (`matches.Service.Swipe`) : (1) déjà traité → `409` ; (2) `discover.Eligible` → sinon `404` ; (3) transaction : verrou consultatif sur la paire (ordre canonique), re-vérification du blocage, `INSERT` du like (`ON CONFLICT DO NOTHING`), et si un like réciproque existe : match + conversation + participants + notifications ; (4) après commit, événement `match` aux deux. Le verrou garantit qu'avec deux likes simultanés un seul match est créé.

**Message** (`chat.Service.Send`) : normalisation du texte → transaction (accès : participant, match actif, aucun blocage) → insertion, `last_message_at`, lecture côté émetteur, notification agrégée du destinataire → événements `message` aux deux participants.

**Lecture** : `last_read_at` par participant ; `read` d'un message envoyé = `created_at <= last_read_at` de l'autre ; le non-lu est calculé à la demande.

**Photo** : lecture bornée → type détecté sur le contenu → dimensions avant décodage → décodage → aplatissement de la transparence → réduction (1280 px / 360 px) → JPEG ; positions maintenues contiguës sous verrou de la ligne `users`.

**Suppression de compte** : vérification du mot de passe → `DELETE FROM users` (cascade) → fermeture des WebSockets.

## 5. Client

- **Données** : TanStack Query ; le WebSocket injecte les messages dans le cache et invalide les listes. À la reconnexion, tout est rechargé ; si la socket est coupée, les listes se rafraîchissent par polling.
- **Session** : `tokenStore` (SecureStore, `localStorage` sur web) est la source de vérité ; le client HTTP rafraîchit le jeton une seule fois en cas de `401` (appel unique partagé) et déconnecte si le refresh est refusé.
- **Navigation** : `AuthScreen` → `OnboardingScreen` (5 étapes, conservé jusqu'à la fin) → pile (`Tabs` + écrans) ; onglets Découvrir / Messages / Alertes / Profil.
- **Aucun accès réseau direct dans l'UI** : écrans et composants passent par `src/api` et `src/hooks`.
- **Photos protégées** : `PhotoImage` envoie le jeton en en-tête (natif) ou récupère l'image en blob (web).
- **Thème** : clair/sombre selon le système (`useTheme`), accessibilité (rôles, libellés, états, régions live), mise en page contrainte sur grand écran.

## 6. Extensions prévues

- Nouvelle recommandation : ajoutez une `discover.Strategy` et son `ORDER BY` ; le filtre d'éligibilité reste commun.
- Multi-instances : implémentez `realtime.Publisher` au-dessus de Redis ou `LISTEN/NOTIFY` ; partagez le rate limiting.
- Stockage d'objets pour les photos : remplacez `photos.Repository`.
- Notifications push : consommez les mêmes événements que le hub.
