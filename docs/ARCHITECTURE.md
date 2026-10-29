# Architecture de Sillage

## Vue d'ensemble

```text
 Mobile (Expo)  ──REST/JSON──▶  API Go (Gin)  ──pgxpool──▶  PostgreSQL
        ▲                           │
        └───── WebSocket /ws ◀──────┘  (événements temps réel)
                                    └──▶ disque (photos)  ·  SMTP (codes de récupération)
```

- Un seul serveur HTTP (Gin). Le WebSocket est une route de ce même serveur.
- Un seul pool de connexions `pgxpool` partagé.
- Pas d'ORM : SQL explicite dans les `repository.go`.

## Backend

```text
services/api/
├─ cmd/api/main.go            composition root (câblage, routes, arrêt gracieux)
├─ cmd/seed/main.go           comptes de démonstration @demo.invalid (refuse APP_ENV=production)
└─ internal/
   ├─ platform/               préoccupations transverses
   │  ├─ config/              variables d'environnement validées
   │  ├─ db/                  connexion, migrations versionnées embarquées
   │  ├─ errors/              AppError (Invalid/Unauthorized/Forbidden/NotFound/Conflict)
   │  ├─ httpx/               Fail() traduit AppError → statut HTTP, BindJSON borné, pagination
   │  ├─ middleware/          auth JWT + session, CORS, en-têtes de sécurité, rate limit, logs
   │  ├─ realtime/            hub WebSocket en mémoire + interface Publisher
   │  ├─ storage/             Storage (disque) avec clés aléatoires validées
   │  ├─ mailer/              SMTP, ou log en développement
   │  └─ logger/ handlers/    logs JSON expurgés, /health
   └─ features/               un dossier par domaine : handler · service · repository · model · routes
      ├─ auth/                inscription, sessions, récupération, suppression de compte
      ├─ profiles/            profil, préférences, centres d'intérêt, profil public
      ├─ photos/              upload validé et ré-encodé, ordre, accès autorisé
      ├─ discovery/           candidats, swipes, matchs, annulation de match
      ├─ chat/                conversations, messages, lu/non-lu, masquage local
      ├─ notifications/       liste, lecture ; Insert() partagé par discovery et chat
      └─ safety/              blocages et signalements
```

Règles : handler fin → service (règles métier, validation) → repository (SQL). Les services renvoient des `AppError` ; les handlers ne font que `httpx.Fail(c, op, err)`. Les erreurs inattendues sont journalisées et renvoyées en 500 générique.

### Sécurité

- JWT HS256 uniquement (`WithValidMethods`, expiration obligatoire, claims `uid`/`sid` stricts), 15 minutes ; refresh token aléatoire haché (SHA-256), rotation à chaque usage, 30 jours.
- `RequireUser` vérifie aussi la session en base : logout, reset de mot de passe ou suppression de compte invalident immédiatement les jetons.
- bcrypt (mot de passe 8–72 octets) ; comparaison à temps constant pour les inconnus (anti-énumération et anti-timing).
- Limitation de débit : auth (30/min/IP), récupération (10/15 min/IP), swipes (240/min/utilisateur), messages (60/min), uploads (20/10 min), signalements (10/h).
- Entrées : corps JSON ≤ 64 Ko, validation de chaque champ, identifiants UUID validés avant SQL, paramètres de pagination bornés, texte nettoyé (espaces, caractères de contrôle).
- Fichiers : taille ≤ 8 Mo, détection par décodage réel (JPEG/PNG/WebP), limites de dimensions (anti-bombe de décompression), ré-encodage JPEG, nom de stockage généré par le serveur.
- Autorisation : toute lecture/écriture est filtrée par l'identité du JWT (participant d'une conversation, propriétaire d'une photo, absence de blocage). Un tiers reçoit 404, pas 403, pour ne pas révéler l'existence de la ressource.
- WebSocket : authentification par première trame (pas de jeton dans l'URL), origine contrôlée, lecture limitée, ping/pong, consommateurs lents déconnectés.

### Modèle de données

| Table | Rôle | Contraintes notables |
| --- | --- | --- |
| `users` | identifiants | e-mail normalisé unique (`CHECK email = lower(btrim(email))`) |
| `sessions`, `password_resets` | refresh tokens hachés, codes de récupération | index unique sur le hash ; cascade |
| `profiles` | prénom, naissance, genre, bio, ville, coordonnées ~1 km, visibilité | `CHECK` longueurs, genre, plages de coordonnées, couple lat/lng |
| `preferences` | genres recherchés, âge 18–99, distance 1–500 km | `CHECK` d'intégrité |
| `photos` | métadonnées des fichiers | `UNIQUE (user_id, position) DEFERRABLE` → réordonner en un `UPDATE` ; position 0–5 |
| `interests`, `user_interests` | catalogue (24) et choix | clé primaire composite |
| `swipes` | like / pass | PK `(from, to)` : un seul avis par personne ; `from <> to` |
| `matches` | paire non ordonnée | `CHECK low < high` + `UNIQUE` → aucun doublon possible |
| `conversations`, `conversation_participants`, `messages` | chat | `messages.id` identité (curseur stable) ; `cleared_at`/`hidden_at` par participant |
| `blocks`, `reports` | sécurité | motifs et statuts contraints |
| `notifications` | match / message | index `(user_id, created_at DESC)` et partiel sur les non lues |

Index requis par le projet : `users.email`, `conversation_participants (conversation_id, user_id)`, `messages (conversation_id, created_at)`, `notifications (user_id, created_at)`, plus ceux de découverte (`profiles (latitude, longitude)`, `swipes (to_user_id, from_user_id)`).

### Découverte et matching

`discovery.candidateQuery` ne retourne que des profils : visibles, avec photo et position, dont le genre est recherché **et** qui recherchent mon genre, dont l'âge est dans ma tranche **et** dont la tranche contient mon âge, à une distance ≤ min(ma distance max, la leur), jamais swipés par moi, sans blocage dans aucun sens. Une page = 1 requête + 2 requêtes d'hydratation (photos, centres d'intérêt) quel que soit le nombre de profils.

`Swipe` : transaction → verrou consultatif de la paire → contrôles (profil complet, blocage, cible disponible) → insertion idempotente → si like réciproque : `matches` + `conversations` + participants + notifications, puis publication temps réel après commit.

### Temps réel

`realtime.Publisher` est la seule dépendance des features. Événements : `ready`, `message`, `read`, `match`, `notification`, `unmatch`. Le client récupère aussi l'état par REST à chaque reconnexion, donc un événement manqué n'est jamais définitif.

## Mobile

- Navigation : stack racine (`Welcome/SignIn/SignUp/…` → `Onboarding` → `Main` + écrans empilés) piloté par l'état d'authentification ; onglets Découvrir / Messages / Activité / Profil.
- `api/client.ts` : seul point d'accès réseau (aucun `fetch` dans l'UI). Rafraîchit le jeton sur 401 (un seul refresh partagé), déconnecte si le refresh token est refusé, expire après 15 s.
- Caches TanStack Query ; le temps réel met à jour directement le cache des messages (`messageCache.ts`) et invalide les listes.
- `useDeck` : file locale de profils, rechargement quand ≤ 3, restauration en cas d'échec réseau, exclusion des profils traités depuis une fiche (`discoverStore`).
- Design system maison (`components/`, `theme/`) : clair/sombre automatique, rôles d'accessibilité, états chargement / vide / erreur, retours par toasts.

## Étendre le produit

1. Nouvelle migration `00N_*.sql`.
2. Nouveau dossier `internal/features/<nom>` avec les 5 fichiers, câblé dans `cmd/api/main.go`.
3. Tests d'intégration dans `cmd/api/*_integration_test.go`, puis un module `src/api/<nom>.ts` côté mobile.
4. Améliorer la recommandation : modifier `discovery/ranking.go` seulement.
