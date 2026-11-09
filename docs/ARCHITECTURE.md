# Architecture

Monorepo : `apps/mobile` (Expo) et `services/api` (Go). Voir le [README](../README.md) pour l'installation.

## Backend (`services/api`)

```
cmd/api/main.go            composition root : config → pool pgx unique → migrations → router Gin
cmd/seed/main.go           comptes de démonstration (@demo.invalid), refuse APP_ENV=production
internal/platform/
  config/      variables d'environnement validées (JWT_SECRET fort, SMTP requis en prod)
  db/          pool pgx, migrations SQL embarquées + schema_migrations
  middleware/  JWT HS256 strict (alg + expiration obligatoires), CORS, en-têtes de sécurité, logs
  ratelimit/   limiteur à fenêtre fixe (IP ou utilisateur)
  realtime/    hub WebSocket en mémoire (interface Publisher)
  access/      règle SQL unique « X peut-il voir Y ? » (blocages, visibilité, match)
  geo/         arrondi des coordonnées, distance approximative
  mailer/      email de récupération (SMTP ou log en dev)
  httpx/       enveloppe d'erreur {"error": ...}, parsing des paramètres
internal/features/
  auth/ profiles/ photos/ matching/ chat/ safety/ notifications/ users/
```

Règles : `handler → service → repository`, pas de SQL dans les handlers, validation côté serveur, un seul serveur HTTP (Gin).
Le pool pgx est partagé ; aucune requête N+1 (les cartes de découverte, conversations et notifications utilisent
`ANY($1)` / `LATERAL`).

### Flux d'un match

1. `POST /swipes` (like) : transaction + verrou consultatif de la paire → insertion du swipe (clé primaire = idempotence).
2. Si l'autre personne a déjà liké : insertion `matches` (paire ordonnée), `conversations`, deux `notifications`, commit.
3. Après commit : événement WebSocket `match` aux deux utilisateurs.

### Accès aux données privées

- Conversations : toute lecture/écriture vérifie la participation, un match actif et l'absence de blocage ; sinon 404.
- Photos : `GET /photos/:id` applique `access.CanView` (soi-même, ou ni bloqué ni caché, ou matché).
- Coordonnées et date de naissance ne sont jamais présentes dans les réponses publiques.

## Mobile (`apps/mobile/src`)

```
api/        client HTTP (refresh automatique mono-vol), endpoints typés, types
auth/       AuthContext (session, SecureStore)
features/   formulaires (profil, préférences, photos), SwipeCard, ReportSheet, hooks, clés de cache
nav/        pile de navigation minimale (onglets + écrans empilés, bouton retour Android)
realtime/   WebSocket → mises à jour du cache TanStack Query, reconnexion exponentielle
screens/    Auth, Onboarding, Discover, Messages, Chat, ProfileDetail, Me, Notifications, réglages
ui/         composants de base (Button, Input, Chip, PhotoImage, états vide/erreur/chargement, Toast)
theme/      palette « Ember » clair/sombre
```

Aucun accès réseau direct depuis les écrans : tout passe par `api/`.
