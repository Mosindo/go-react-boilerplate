# Architecture de Lien

Voir le README pour la vue d'ensemble, les variables et les commandes. Ce document fixe les règles de structure.

## Backend (`services/api`)

- `handler → service → repository`, un dossier par fonctionnalité (`handler.go`, `service.go`, `repository.go`, `model.go`, `routes.go`). Pas de SQL dans les handlers, pas d'`net/http` mélangé à Gin pour le routage, un seul serveur.
- Les erreurs destinées au client sont des `platformerrors.Error` (statut + code stable + message) ; tout le reste est journalisé et masqué en 500. Réponse d'erreur : `{"error": "...", "code": "..."}`.
- Identifiants de chemin validés comme UUID avant d'atteindre la base (`ParamUUID`), listes paginées par curseur keyset (`platform/pagination`), limites explicites (`limit` borné).
- Les dépendances entre fonctionnalités passent par de petites interfaces (`CardLoader`, `Viewers`, `FilePurger`, `realtime.Publisher`).
- `internal/server.New` assemble tout ; `main` et les tests d'intégration l'utilisent.

### Routes

Publiques : `GET /health`, `POST /auth/{register,login,refresh,logout}`, `POST /auth/password/{forgot,reset}`, `GET /ws?ticket=`.
Authentifiées : `GET /me`, `PUT /me/{profile,preferences,location}`, `DELETE /me/location`, `DELETE /me`, `POST /me/password`, `POST|PUT|DELETE /me/photos…`, `GET /photos/:id`, `GET /profiles/:id`, `GET /interests`, `GET /discover`, `POST /swipes`, `GET /matches`, `DELETE /matches/:id`, `GET /conversations`, `…/:id/messages`, `…/:id/read`, `DELETE /conversations/:id`, `GET /conversations/unread`, `GET /notifications…`, `GET|POST /blocks`, `DELETE /blocks/:id`, `POST /reports`, `POST /ws/ticket`.

## Frontend (`apps/mobile`)

- Aucun accès réseau depuis l'UI : tout passe par `src/api` (client avec refresh automatique, erreurs typées, messages en français).
- État serveur dans React Query (`src/hooks/data.ts`), événements WebSocket appliqués au cache par `src/realtime/events.ts` (fonction pure testée).
- Navigation : pile d'authentification / onboarding / pile principale + onglets ; thème clair/sombre automatique (`src/theme`).
- Web : photos protégées chargées via `fetch` + blob (une balise `<img>` ne peut pas envoyer d'en-tête Authorization).
