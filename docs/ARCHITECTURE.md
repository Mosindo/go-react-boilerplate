# Architecture

Tendre est un monorepo : une API Go (`services/api`) et une application Expo/React Native (`apps/mobile`) qui sert aussi
de build web.

```text
 mobile / web ──REST + WebSocket──▶ Gin (un seul serveur HTTP) ──▶ PostgreSQL
                                          │
                                          └──▶ stockage de fichiers (disque local, interface remplaçable)
```

## Backend

`handler → service → repository`, un dossier par domaine dans `internal/features`, chacun avec `model.go`, `handler.go`,
`service.go`, `repository.go`, `routes.go`. `internal/app` assemble les features ; `cmd/api` démarre le serveur ;
`cmd/seed` crée des comptes de démonstration.

| Feature         | Responsabilité                                                                                         |
| --------------- | ------------------------------------------------------------------------------------------------------ |
| `auth`          | inscription, connexion, sessions (refresh rotatif), récupération et changement de mot de passe, suppression de compte |
| `profiles`      | profil, préférences, centres d'intérêt, confidentialité, hydratation des profils publics (2 requêtes groupées) |
| `photos`        | upload, traitement d'image, ordre, remplacement, suppression, URLs signées                              |
| `matching`      | découverte, swipes, matchs (transaction + verrou), consultation d'un profil, annulation d'un match       |
| `chat`          | conversations, messages (pagination par curseur), lu/non lu, masquage local                             |
| `safety`        | blocages et signalements                                                                                |
| `notifications` | liste, non-lus, lecture ; `notifications.Insert` utilisable dans la transaction d'une autre feature     |

`internal/platform` : `config`, `db` (+ migrations versionnées, `dbtest` pour les tests), `middleware` (auth, CORS,
en-têtes de sécurité, rate limiting, limite de corps), `realtime` (hub + WebSocket), `storage`, `mailer`, `httpx`,
`logger`, `handlers` (santé).

### Règle d'éligibilité (cœur du produit)

Défini une seule fois en SQL (`matching/repository.go`, constante `eligible`) et réutilisé pour la découverte, le
like et la consultation d'un profil : profil visible, genre voulu par chacun, âge dans la plage de chacun, distance
dans la limite de chacun, au moins une photo, aucun blocage dans un sens ou l'autre. La découverte exclut en plus les
profils déjà traités. Aucune de ces règles n'est vérifiée côté client.

### Temps réel

`POST /realtime/ticket` délivre un JWT de 30 s (audience dédiée) échangé contre une connexion WebSocket. Le `Hub`
(`user_id → connexions`) publie sans jamais bloquer (un client lent perd des événements et se resynchronise via REST).

### Évolutions prévues

- `Ranker` (matching) : nouvel algorithme de recommandation sans toucher au SQL.
- `storage.Storage` : S3/GCS à la place du disque.
- `realtime.Publisher` : LISTEN/NOTIFY ou Redis pour plusieurs instances.
- PostGIS pour la recherche géographique à grande échelle.

## Mobile

- `src/api` : un client HTTP (`client.ts`) — injection du token, refresh automatique mono-requête, erreurs typées — et un
  module par domaine. Les écrans n'appellent jamais `fetch`.
- `src/hooks` : `useAuth` (session, démarrage, expiration), `queries` (clés et hooks TanStack Query),
  `useRealtime` (WebSocket → mise à jour du cache), `deckBus`.
- `src/navigation` : porte d'entrée unique — non connecté → pile d'authentification ; connecté sans profil complet →
  onboarding ; sinon onglets + écrans empilés.
- `src/components` : composants de domaine (deck, carte, photos, préférences…) ; `src/shared` : design system.
- Performance : requêtes paginées, préchargement du deck (10 profils, rechargement à 3 restants), vignettes pour les
  listes, URLs de photos stables pendant une heure (cache HTTP), mises à jour de cache ciblées plutôt que rechargements.

## Tests

Voir le README. Les tests d'intégration créent une base PostgreSQL jetable par test (`dbtest.New`) : ils tournent en
parallèle sans se marcher dessus.
