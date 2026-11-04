# API Liane

Base URL : `http://localhost:18080` (Docker) — JSON partout, UTF‑8. Authentification : `Authorization: Bearer <accessToken>`
(sauf routes marquées *public*). Les erreurs ont toujours la forme `{"error": "message", "code": "snake_case"}`.

Codes courants : `400 validation`, `401 unauthorized`, `403` (interdit : âge, profil incomplet, mauvais mot de passe, bloqué),
`404 not_found` (ressource inexistante **ou** à laquelle vous n'avez pas droit), `409` (doublon), `429 rate_limited` (+ `Retry-After`).

## Auth
| Méthode | Route | Description |
|---|---|---|
| POST | `/auth/register` *public* | `{email, password}` → `{accessToken, refreshToken, user}`. Mot de passe 8‑72 car., 1 lettre + 1 chiffre. |
| POST | `/auth/login` *public* | idem |
| POST | `/auth/refresh` *public* | `{refreshToken}` → nouveaux jetons (le refresh token est **tourné**, l'ancien est invalidé). |
| POST | `/auth/logout` *public* | `{refreshToken}` → 204, la session est révoquée immédiatement. |
| POST | `/auth/password/forgot` *public* | `{email}` → toujours 202 (n'indique pas si le compte existe). Envoie un code à 8 chiffres (15 min, 5 essais). |
| POST | `/auth/password/reset` *public* | `{email, code, newPassword}` → 204, toutes les sessions sont révoquées. |
| GET | `/me` | Compte courant. |
| PUT | `/me/password` | `{currentPassword, newPassword}` → 204 (les autres sessions sont révoquées). |
| DELETE | `/me` | `{password}` → 204. Supprime définitivement le compte et toutes ses données (profil, photos + fichiers, likes, matchs, messages, notifications, blocages, signalements). |

Access token : JWT HS256, 15 min. Refresh token : opaque, 30 jours, stocké haché.

## Profil
| Méthode | Route | Description |
|---|---|---|
| GET | `/profile/me` | Profil privé + `isComplete` / `missing` (`preferences`, `photos`). |
| PUT | `/profile/me` | `{firstName, birthDate (YYYY-MM-DD, requis à la création, 18+, ensuite immuable), gender (woman/man/nonbinary), bio ≤500, city ≤80, interests: [slug] ≤10, isVisible}` |
| PUT | `/profile/preferences` | `{interestedIn: [..], minAge ≥18, maxAge ≤99, maxDistanceKm 1‑500}` |
| PUT | `/profile/location` | `{latitude, longitude, city?}` — **arrondi à ~1 km côté serveur**, la position exacte n'est jamais stockée. |
| GET | `/interests` | Catalogue des centres d'intérêt `{slug, label}`. |
| GET | `/profiles/:id` | Profil public d'un autre utilisateur (âge, distance approximative par pas de 5 km, photos, intérêts). 404 si bloqué (dans un sens ou l'autre), masqué (hors match) ou inexistant. |

## Photos
`url` est une URL relative signée et expirante (`/photos/:id/file?exp=…&sig=…`), à préfixer par la base URL ; elle ne s'obtient que par les réponses de l'API.
| Méthode | Route | Description |
|---|---|---|
| GET | `/photos` | Mes photos (ordre d'affichage). |
| POST | `/photos` | `multipart/form-data`, champ `photo` (JPEG/PNG vérifié sur le contenu, ≤ 8 Mo, ≥ 200 px). Ré-encodé en JPEG ≤ 1280 px, EXIF supprimé. Max 6. |
| PUT | `/photos/:id` | Remplace le fichier en gardant la position. |
| PUT | `/photos/order` | `{ids: [..]}` — permutation complète ; la première est la photo principale. |
| PUT | `/photos/:id/primary` | Passe la photo en première position. |
| DELETE | `/photos/:id` | Supprime (fichier inclus) et recompacte les positions. |
| GET | `/photos/:id/file?exp&sig` *public (signé)* | Octets de l'image. |

## Découverte et matching
| Méthode | Route | Description |
|---|---|---|
| GET | `/discover?limit=10` | Prochains profils (max 20). Filtre : préférences mutuelles (genre, âge), distance, non encore swipés, non bloqués, avec photo. Un profil jugé ne revient jamais. |
| POST | `/swipes` | `{userId, action: "like"\|"pass"}` → `{result: liked\|passed\|matched, alreadyProcessed, match?: {matchId, conversationId, profile}}`. Idempotent. 403 `profile_incomplete` si votre profil n'est pas complet. |
| GET | `/likes/received` | Personnes qui vous ont liké et que vous n'avez pas encore jugées (gratuit). |

## Conversations
Une conversation est créée automatiquement au match.
| Méthode | Route | Description |
|---|---|---|
| GET | `/conversations?limit&cursor` | Liste (plus récente d'abord) avec `unread`, `lastMessage`, `with`. Les matchs sans message sont inclus. |
| GET | `/conversations/:id/messages?limit&before` | Historique paginé (ordre chronologique), `nextBefore`, `peerLastReadId` (accusé de lecture). |
| POST | `/conversations/:id/messages` | `{body}` 1‑2000 caractères → 201. 403 `blocked` si blocage. |
| POST | `/conversations/:id/read` | Marque tout comme lu. |
| DELETE | `/conversations/:id` | Suppression locale (masquée pour vous ; réapparaît à un nouveau message, sans l'ancien historique). |
| POST | `/conversations/:id/unmatch` | Défait le match pour les deux (conversation supprimée). |

## Temps réel — `GET /ws` (WebSocket)
1. Connexion sur `ws(s)://<base>/ws`.
2. Première trame (< 5 s) : `{"type":"auth","token":"<accessToken>"}` ; réponse `{"type":"ready"}`. Sinon fermeture `4401`.
3. Keep‑alive : `{"type":"ping"}` → `{"type":"pong"}`.
4. Évènements serveur : `message.new` (message), `conversation.read` `{conversationId, readerId, lastReadId}`, `conversation.removed` `{conversationId}`, `match.new` `{matchId, conversationId, userId}`, `notification.new` (notification).
La session est revérifiée toutes les 5 min (fermeture `4401` si révoquée). Le hub est en mémoire (une instance) ; voir `internal/platform/realtime`.

## Notifications
| Méthode | Route | Description |
|---|---|---|
| GET | `/notifications?limit&cursor` | `{notifications, nextCursor, unread}` — types : `match`, `like`, `new_message`. |
| POST | `/notifications/:id/read` | 204 |
| POST | `/notifications/read-all` | 204 |

## Sécurité / modération
| Méthode | Route | Description |
|---|---|---|
| GET | `/blocks` | Utilisateurs bloqués. |
| POST | `/blocks` | `{userId}` → 204. Masque mutuellement profils, conversations et découverte. Idempotent. |
| DELETE | `/blocks/:userId` | Débloque. |
| POST | `/reports` | `{userId, reason: spam\|fake\|harassment\|inappropriate\|underage\|other, details ≤1000, alsoBlock}` → 201. Un seul signalement ouvert par couple. |

## Santé
`GET /health` *public* → `{"status":"ok"}` (503 si la base est injoignable).
