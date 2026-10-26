# Architecture

## Vue d'ensemble

```text
 Expo client (iOS / Android / web)
   │  REST (JSON, multipart pour les photos)        WebSocket /ws
   ▼                                                ▼
 Gin router ── middleware: sécurité, CORS, limite de corps, rate limit, auth JWT
   │
   ├─ features/auth          comptes, sessions, récupération, suppression
   ├─ features/profiles      profil, préférences, localisation, confidentialité, centres d'intérêt
   ├─ features/photos        pipeline d'images + URLs signées
   ├─ features/matching      découverte, swipes, matchs
   ├─ features/chat          conversations, messages, accusés de lecture
   ├─ features/notifications liste et compteurs
   └─ features/safety        blocages et signalements
   │
 platform/{config,db,middleware,realtime,mailer,signing,logger,handlers}
   │
 PostgreSQL (pgxpool partagé)
```

Chaque module suit `handler -> service -> repository` (+ `model.go`, `routes.go`). Les modules ne s'appellent pas entre eux, sauf deux contrats explicites :
`profiles.Loader` (vue publique d'un profil, chargée par lot en 3 requêtes, sans N+1) et `notifications.Create` (écriture dans la transaction de l'appelant).

## Flux critiques

**Swipe** : `POST /discover/swipes` → vérifie que le profil est éligible (même SQL que la découverte) → transaction + verrou consultatif de la paire → insertion du swipe (unique) → si like réciproque, création du match unique et des deux notifications → événement `match` temps réel à l'autre personne.

**Message** : `POST /conversations/:id/messages` → appartenance au match actif vérifiée dans la requête SQL → insertion + notification (une seule non lue par conversation) → événement `message` aux deux participants.

**Authentification** : access token JWT HS256 de 15 min (claims `uid`, `sid`, `exp` obligatoires) + refresh token opaque de 30 jours, haché en base et rotatif (une seule utilisation). Le client mutualise les rafraîchissements concurrents.

## Sécurité

Autorisations toujours côté serveur ; entrées validées et normalisées (email, texte, dates, pagination, uploads) ; bcrypt ; réponses d'authentification sans énumération (login, récupération) ; rate limiting (auth par IP, swipes/messages/uploads/signalements/localisation par utilisateur) ; corps JSON limités à 1 Mo ; en-têtes de sécurité ; CORS sur liste blanche ; `X-Forwarded-For` ignoré sauf `TRUSTED_PROXIES` ; secrets uniquement via variables d'environnement.

## Performance

Pagination par curseur (conversations, notifications, messages), lots de découverte plafonnés, index sur les chemins chauds (`messages(match_id, created_at)`, `notifications(user_id, created_at)`, profils visibles, swipes, matchs), listes sans photos binaires, URLs de photos stables 6 h (cache client).

## Frontend

`src/api` (client typé + refresh) · `src/auth` (session, stockage sécurisé) · `src/realtime` (WebSocket → cache React Query) · `src/hooks` · `src/components` (design system : thème clair/sombre automatique) · `src/screens` · `src/navigation`. Aucune requête réseau hors de `src/api` et du socket.

## Étendre

- **Recommandation** : modifier le `ORDER BY` / les filtres de `matching/repository.go` (ou extraire un scoreur).
- **Plusieurs instances** : remplacer `realtime.Hub` (interface `Publisher`) par un transport partagé.
- **Modération** : consommer `reports` ; basculer `users.status` ; l'API refuse déjà connexion et rafraîchissement aux comptes suspendus.
