# Architecture

Monorepo :

```text
apps/mobile/        Client Expo / React Native (TypeScript) — « Liane »
services/api/       API Go (Gin + pgx + PostgreSQL)
docs/               API.md (contrat), ce document
.github/workflows/  CI
scripts/            smoke-api.sh (test E2E HTTP)
docker-compose.yml  PostgreSQL + API
```

## Backend

```text
services/api/
├─ cmd/api/        point d'entrée + routeur (composition root) + tests d'intégration
├─ cmd/seed/       données de démonstration (refuse APP_ENV=production)
└─ internal/
   ├─ platform/    config, db (migrations suivies), middleware (auth, CORS, rate limit…),
   │               httpx (erreurs/validation HTTP), errors, logger, mailer, media (photos),
   │               geo (règles de confidentialité), realtime (hub WebSocket)
   └─ features/    auth · account · profiles · photos · discovery · chat · notifications · safety
```

Chaque feature suit `handler → service → repository` (+ `model.go`, `routes.go`) : handlers fins, règles métier dans le service,
SQL explicite uniquement dans le repository. Un seul serveur HTTP (Gin dans `http.Server`), un seul pool pgx partagé.

### Flux de dépendances
`discovery` → `profiles` (hydratation des profils publics), `notifications` (Notifier), `realtime` (Publisher) ;
`chat` → `notifications`, `realtime` ; `account` → `auth` (validation du mot de passe), `media` ; les features ne s'importent pas autrement.

### Décisions clés
- **Migrations** : fichiers SQL embarqués, appliqués dans l'ordre, chacun dans une transaction, suivis dans `schema_migrations`, verrou advisory pour les démarrages concurrents.
- **Sessions** : access token JWT HS256 (15 min, `exp` obligatoire, `uid`+`sid`), refresh token opaque haché et tourné. `RequireUser` vérifie en base que la session n'est pas révoquée : logout, reset de mot de passe et suppression de compte prennent effet immédiatement.
- **Match** : table `swipes` (PK `(from,to)`) → un utilisateur n'est jugé qu'une fois. Le like réciproque crée `matches` (paire ordonnée `user_a < user_b`, unique) + `conversations` + participants dans la même transaction, sous un verrou advisory par paire (deux likes simultanés → exactement un match, testé).
- **Recommandation** : une seule requête SQL (`discovery/repository.go`, `candidatesSQL`) ; le `ORDER BY` est le point d'extension (aujourd'hui : ceux qui vous ont liké, puis proximité, puis activité récente).
- **Géolocalisation** : coordonnées arrondies à 2 décimales (~1 km) avant stockage ; distance affichée par paliers de 5 km ; jamais de coordonnées dans une réponse.
- **Photos** : type vérifié sur le contenu, anti décompression‑bomb, ré‑encodage JPEG (EXIF supprimé), clés de stockage aléatoires, accès uniquement via URL signée HMAC expirante (stable ~6‑12 h pour le cache). Stockage disque derrière `media.Store` (remplaçable par du stockage objet).
- **Temps réel** : WebSocket authentifié par première trame ; `realtime.Publisher` est une interface (hub mémoire aujourd'hui, Redis/NATS pour plusieurs instances). Les évènements ne sont qu'un accélérateur : tout est récupérable via REST.
- **Blocage** : filtré dans toutes les requêtes (découverte, profils, conversations, envoi). Les données sont conservées pour permettre la modération des signalements, et supprimées avec les comptes.
- **Rate limiting** : seaux à jetons en mémoire par IP (auth) ou par utilisateur (swipes, messages, uploads, signalements) ; garde‑fous anti‑abus, pas des limites produit. `TRUSTED_PROXIES` doit être défini derrière un proxy.

## Mobile
Voir `apps/mobile/README.md`. Tout accès réseau passe par `src/api` (client typé, refresh de jeton, WebSocket) ; l'UI ne fait jamais de `fetch`.
