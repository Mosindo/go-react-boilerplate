# Ember — architecture

## Monorepo

- `services/api` — API Go (Gin + pgx). Un binaire `api`, un utilitaire `seed`.
- `apps/mobile` — client Expo (iOS, Android, web), TypeScript strict.
- `docs` — ce document et les guides destinés aux agents de développement.
- `docker-compose.yml` — PostgreSQL + API pour le développement local.

## Backend

```text
cmd/api/{main.go,router.go}   composition root : config → pool → migrations → router
cmd/seed                      comptes de démo (refuse APP_ENV=production)
internal/platform/
  config      variables d'environnement validées au démarrage
  db          pool pgx, migrations embarquées (schema_migrations + verrou consultatif)
  middleware  auth (JWT HS256 strict + session active), CORS, en-têtes de sécurité, rate limiter, logs
  realtime    hub WebSocket (interface Publisher utilisée par les features)
  storage     interface Storage + implémentation disque
  mailer      interface Mailer (log en dev, SMTP en prod)
  validate    e-mail, âge, texte
  httpx       réponses d'erreur et pagination communes
  errors      AppError (statut + message client) ; toute autre erreur devient un 500 opaque
internal/features/<feature>/   handler.go → service.go → repository.go (+ model.go, routes.go)
  auth  profiles  photos  discovery  matches  chat  notifications  safety
```

Flux de dépendances : `handler → service → repository`. Les features se parlent par de petites interfaces :
`profiles.CardLoader` (hydratation de cartes profil en 3 requêtes quelle que soit la taille de page, donc pas de N+1),
`notifications.Notifier`, `realtime.Publisher`, `photos.Viewer`.

### Autorisation (toujours côté serveur)

| Ressource | Règle |
|---|---|
| Profil / photo d'autrui | non bloqué dans aucun sens **et** (profil découvrable **ou** apparié) ; sinon 404 identique à « inexistant » |
| Conversation | l'utilisateur est un participant (sinon 404) et aucun blocage |
| Swipe | profil complet, cible existante, non bloquée, découvrable (ou qui a déjà liké) |
| Photo (écriture) | uniquement les siennes ; limite de 6 appliquée sous verrou de ligne |
| Notification | uniquement les siennes |

### Matching

`POST /swipes` ouvre une transaction, prend `pg_advisory_xact_lock(hash(paire ordonnée))`, enregistre le swipe (`ON CONFLICT DO NOTHING`), et si le like inverse existe crée `matches` (`user_a < user_b`, unique) + `conversations` + 2 `conversation_participants`. Le verrou garantit que deux likes simultanés produisent exactement un match.

### Données

Voir `internal/platform/db/migrations`. Les index critiques : `users.email` (unique), `messages (conversation_id, created_at DESC, id DESC)`, `notifications (user_id, created_at DESC)`, `conversation_participants (user_id, conversation_id)`, `swipes (target_id, action)`, `profiles (latitude, longitude)` partiel (découvrables).

## Mobile

```text
src/api         client HTTP (timeout, refresh single-flight, erreurs typées), endpoints typés, types
src/auth        contexte d'authentification, stockage sécurisé des jetons (SecureStore / localStorage sur web)
src/realtime    client WebSocket (reconnexion exponentielle) + synchronisation du cache React Query
src/hooks       requêtes React Query et clés de cache
src/lib         logique pure testée (date de naissance, formats, fusion de messages, geste)
src/theme       palette claire/sombre, espacements
src/ui          composants réutilisables (Button, TextField, Chip, Sheet, Toast, Photo…)
src/features    auth, discover, chat, notifications, profile, safety
src/navigation  pile racine (déconnecté / onboarding / principal) + onglets
```

Règles : aucun appel réseau dans les composants (uniquement via `src/api`), état serveur dans React Query, photos privées chargées avec le jeton (`expo-image`, cache disque par URL versionnée), aucune valeur secrète dans le client.

## Évolutions prévues (points d'extension)

- Recommandation : remplacer `discovery.rankingOrder`.
- Stockage objet : nouvelle implémentation de `storage.Storage`.
- Push natif : brancher un fournisseur dans `notifications.Service.Notify`.
- Multi-instances : implémenter `realtime.Publisher` au-dessus de LISTEN/NOTIFY ou Redis.
