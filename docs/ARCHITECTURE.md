# Alma — architecture

Ce document explique comment le dépôt est organisé et pourquoi, pour pouvoir le faire évoluer sans casser ses frontières.
Les commandes d'installation et d'exploitation sont dans le [README](../README.md).

## 1. Vue d'ensemble

```text
 Mobile (Expo)                      API (Go / Gin)                   PostgreSQL
┌──────────────────┐   REST/JSON   ┌───────────────────────┐        ┌───────────┐
│ écrans           │──────────────▶│ handlers (fins)       │        │ users…    │
│ TanStack Query   │               │ services (règles)     │──SQL──▶│ profiles… │
│ client API typé  │◀── WebSocket ─│ repositories (SQL)    │        │ matches…  │
└──────────────────┘   événements  │ platform (transverse) │        └───────────┘
                                   └───────────┬───────────┘
                                               │ fichiers
                                          volume photos
```

- Un seul serveur HTTP (Gin), une seule pool de connexions PostgreSQL (`pgxpool`), pas d'ORM.
- Le client mobile ne parle à l'API que par `src/api` ; aucun écran n'appelle `fetch` directement.
- Le serveur est l'autorité : âge, éligibilité, doublons, permissions, limites sont revérifiés à chaque requête.

## 2. Backend

```text
services/api/
├─ cmd/api/main.go              composition root: config, base, migrations, services, routes, arrêt propre
├─ cmd/seed/main.go             données de démonstration (refuse APP_ENV=production)
└─ internal/
   ├─ platform/
   │  ├─ config/                variables d'environnement, validation du secret JWT
   │  ├─ db/ (+ migrations/)    connexion, migrations suivies (schema_migrations), purge périodique
   │  ├─ middleware/            auth (JWT HS256 + session active), CORS, en-têtes, limitation de débit, journaux
   │  ├─ httpx/                 réponses d'erreur, JSON borné, pagination, UUID
   │  ├─ errors/                AppError : erreurs métier affichables (statut + code stable)
   │  ├─ realtime/              hub WebSocket, tickets à usage unique
   │  ├─ blob/                  stockage de fichiers (disque local, interface remplaçable)
   │  ├─ mailer/                SMTP, mailer de développement, mailer désactivé
   │  └─ logger/, handlers/     journaux JSON, /health
   └─ features/                 auth, users, files, matching, chat, notifications, safety
```

Règles : `handler → service → repository` ; pas de SQL dans les handlers ; les préoccupations partagées vivent dans
`platform`, les préoccupations métier dans `features` ; un module n'importe pas le repository d'un autre — il dépend d'une
**interface** définie chez lui (ex. `matching.Profiles`, `chat.Notifier`, `users.PhotoSource`), câblée dans `main.go`.

### Erreurs

Les services renvoient des `*platformerrors.AppError{Status, Code, Message}` ; `httpx.Fail` les traduit en
`{"error","code"}`. Toute autre erreur devient un `500 internal` opaque (détails dans les journaux seulement). Les codes
sont stables : l'application mobile les traduit en français (`src/i18n/errors.ts`).

### Authentification

- Mot de passe : bcrypt, 8 caractères minimum, 72 octets maximum (limite de bcrypt, refusée plutôt que tronquée).
- Jeton d'accès : JWT HS256 (méthode HMAC vérifiée), expiration obligatoire, 15 min, claims `uid` et `sid`.
- À chaque requête, `sid` est comparé à la table `sessions` (non révoquée, non expirée) : déconnexion, changement de
  mot de passe et suppression de compte agissent immédiatement.
- Refresh token : 32 octets aléatoires, haché (SHA-256) en base, **rotation** à chaque usage.
- Connexion : comparaison bcrypt factice si l'e-mail est inconnu (temps constant), message identique dans les deux cas.
- Récupération : code à 6 chiffres, haché par HMAC, 15 min, 5 essais, usage unique, révoque toutes les sessions.

### Découverte et matching

Les règles tiennent en une requête SQL commune (`matching/repository.go`) : la personne est visible, a au moins une photo,
n'est pas bloquée (dans aucun sens), correspond au genre/âge/distance recherchés par le demandeur **et** réciproquement.

- `Candidates` ajoute : jamais décidé (`likes`) et absent de la liste `exclude` envoyée par le client.
- `Candidate` (consultation d'un profil) : éligible *ou* déjà matché.
- `Swipe` (transaction) : verrou consultatif du couple → éligibilité → `INSERT` de la décision (la clé primaire interdit
  les doublons) → si like réciproque, création du match (`user_a < user_b`, `UNIQUE`) et de la conversation.

Pour changer l'algorithme de recommandation, on modifie l'`ORDER BY` de `Candidates` (ou on le remplace par une requête
de score) sans toucher au reste.

### Messagerie et temps réel

- Une conversation est liée à un match (`conversations.match_id`, suppression en cascade) : annuler le match, bloquer ou
  supprimer un compte efface la conversation.
- Chaque endpoint de conversation commence par `PeerOf`, qui ne répond qu'aux participants : tous les autres obtiennent
  `404`, sans fuite d'existence.
- L'état de lecture est `conversation_participants.last_read_at` ; le « Lu » de l'expéditeur compare ce repère à la date
  du message. Écrire dans une conversation vaut lecture de ce qui précède.
- `realtime.Hub` : `POST /ws/ticket` (authentifié) → ticket aléatoire de 30 s, à usage unique → `GET /ws?ticket=`. Les
  clients lents perdent un événement plutôt que de bloquer l'émetteur et se resynchronisent par REST à la reconnexion.

### Photos

Pipeline `files` : taille ≤ 10 Mo → détection du type par le contenu → dimensions lues avant décodage (≤ 12 000 px,
≤ 40 Mpx, ≥ 200 px) → décodage → orientation EXIF → aplatissement du fond → JPEG 1080 px + miniature 360 px → clés
aléatoires dans `blob.Store`. Les positions sont denses et modifiées sous un verrou consultatif par utilisateur.
`GET /photos/:id/file` est autorisé par une signature HMAC (clé dérivée du secret) qui expire par fenêtre de 24 h.

### Migrations

Fichiers `NNN_nom.sql` embarqués, exécutés dans l'ordre, chacun dans une transaction, une seule fois (`schema_migrations`),
sous `pg_advisory_lock`. Les migrations ne se modifient jamais après publication : on en ajoute une nouvelle.

## 3. Mobile

```text
apps/mobile/src/
├─ api/            client.ts (base URL, refresh automatique), endpoints.ts, platform.ts, auth.ts, types.ts
├─ hooks/          useAuth (session), queries (clés + hooks TanStack Query), useRealtime (WebSocket)
├─ navigation/     RootNavigator : déconnecté → Auth ; connecté → Onboarding tant que le profil est incomplet → Tabs
├─ screens/        Auth, ForgotPassword, Onboarding, Home (découverte), Matches, Chat, Notifications, Profile, …
├─ components/     SwipeDeck, PhotoCarousel, PhotoManager, ProfileBody, MatchOverlay, ActionSheet, …
├─ shared/         ui (Text, Button, Input, Card, Chip…), layout, feedback (toasts, vues vide/erreur/chargement)
├─ theme/          palettes claire/sombre, jetons d'espacement, de rayon et de typographie
├─ lib/            fonctions pures testées : dates, validation, pile de découverte, photos
└─ i18n/           textes et messages d'erreur en français
```

- **État serveur** : TanStack Query (clés dans `hooks/queries.ts`). Le temps réel met le cache à jour (`appendMessage`)
  ou l'invalide ; à la (re)connexion l'application se resynchronise.
- **Session** : jetons dans `expo-secure-store`. Sur `401`, le client tente un refresh unique partagé puis rejoue la
  requête ; en cas d'échec, `AuthProvider` déconnecte.
- **Pile de découverte** : file locale rechargée quand il reste ≤ 3 cartes, avec l'`exclude` des cartes déjà détenues ;
  une décision est appliquée de façon optimiste et la carte revient si l'enregistrement échoue.
- **Performance** : `expo-image` avec cache disque, préchargement de la photo suivante, miniatures dans les listes,
  pagination des messages et des conversations, aucune liste entière chargée d'avance.
- **Accessibilité** : rôles et libellés, zones tactiles ≥ 44 px, texte redimensionnable (plafonné), contraste vérifié sur
  les deux palettes, états annoncés (`accessibilityState`, régions live).

## 4. Qualité et tests

| Niveau | Où | Quoi |
| --- | --- | --- |
| Unitaire Go | `internal/**/**_test.go` | Âge, arrondis, normalisation de texte, traitement d'image, limiteur de débit, config |
| Intégration Go | `cmd/api/*_integration_test.go`, `internal/features/auth` | HTTP complet sur PostgreSQL réel |
| Unitaire mobile | `src/lib/*.test.ts` (Vitest) | Dates, validation, pile de découverte, photos |
| Parcours API | `apps/mobile/scripts/e2e-smoke.js` | Inscription → match → chat → suppression, sur une API en marche |
| Parcours UI | `apps/mobile/e2e/maestro` | Connexion → like → match → message (appareil requis) |

## 5. Étendre sans casser

- Nouveau module : `features/<nom>/{handler,service,repository,model,routes}.go`, interfaces vers les autres modules,
  câblage dans `cmd/api/main.go`, migration SQL, tests d'intégration.
- Nouvelle règle de visibilité : l'ajouter dans les fragments SQL de `matching/repository.go` (une seule fois).
- Passage à plusieurs instances : implémenter `blob.Store` (S3), publier les événements via Redis/`LISTEN`+`NOTIFY`
  derrière `Hub.Publish`, partager le limiteur de débit.
