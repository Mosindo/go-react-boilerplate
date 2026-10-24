# Alba — application de rencontre 100 % gratuite

Alba permet à deux adultes de se rencontrer : créer un compte et un profil, ajouter des photos, découvrir des profils, liker ou passer, matcher quand l'intérêt est réciproque, discuter en temps réel, gérer sa confidentialité, bloquer / signaler, et supprimer son compte.

**Aucun abonnement, aucun paywall, aucun boost payant, aucune limite de likes, aucune publicité.** C'est une règle produit : elle est écrite dans [`AGENTS.md`](AGENTS.md) et ne doit pas être contournée.

Le dépôt est un monorepo issu du boilerplate `go-react-saas` :

| Dossier | Contenu |
| --- | --- |
| `services/api` | API Go (Gin, PostgreSQL via pgxpool, JWT HS256, WebSocket), sans ORM |
| `apps/mobile` | Client Expo / React Native (iOS, Android, web), TypeScript |
| `docker-compose.yml` | PostgreSQL + API |
| `scripts/smoke.mjs` | Test de fumée de bout en bout de l'API |
| `docs/ARCHITECTURE.md` | Architecture détaillée et décisions techniques |

## Fonctionnalités

- **Compte** : inscription (18 ans minimum), connexion, sessions à refresh token rotatif, récupération de mot de passe par code e-mail, suppression définitive du compte (mot de passe requis).
- **Profil** : prénom, âge (date de naissance non modifiable), genre, bio, ville, centres d'intérêt (10 max), jusqu'à 6 photos (ordre, photo principale, remplacement, suppression), préférences (genres recherchés, tranche d'âge, distance maximale), position approximative optionnelle.
- **Découverte** : paquet de cartes (swipe ou boutons), navigation entre photos, profil complet, like / passer. Un profil traité ne revient jamais.
- **Matchs** : like réciproque = match, sans doublon même en cas de likes simultanés. Se désapparier est possible à tout moment.
- **Chat** : une conversation par match, messages texte, heure d'envoi, lu / non lu, temps réel (WebSocket), suppression locale d'une conversation.
- **Notifications** : nouveaux matchs (in-app + temps réel), compteur de non-lus.
- **Confidentialité & sécurité** : profil masquable, distance masquable, blocage, signalement (avec blocage optionnel), autorisations vérifiées côté serveur, limitation de débit.
- **Interface** : identité visuelle propre (prune + abricot), mode clair/sombre automatique, états de chargement / vide / erreur, retours visuels, libellés d'accessibilité.

## Prérequis

- Go 1.24+, Node 20+ (testé avec Node 22), Docker (ou un PostgreSQL 15+ local).

## Démarrage rapide (Docker)

```bash
cp .env.example .env
# renseignez JWT_SECRET (>= 32 caractères aléatoires) :
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env   # ou éditez .env à la main

docker compose up --build -d          # PostgreSQL + API sur http://localhost:18080
curl http://localhost:18080/health    # {"status":"ok"}

# données de démonstration (comptes @seed.invalid, jamais en production)
docker compose exec api ./seed
```

Comptes de démo : `demo01@seed.invalid` (Camille, déjà matchée avec Lucas) … `demo12@seed.invalid`, mot de passe `Demo-password-1`. Supprimez-les avec `docker compose exec api ./seed --purge`.

> Port : l'API est publiée sur `18080` (et non `8080`) pour éviter les conflits avec d'autres services locaux. Si ce port est pris, changez-le dans `docker-compose.yml` **et** dans `EXPO_PUBLIC_API_URL`.

## Lancer l'application mobile

```bash
cd apps/mobile
npm ci

# Web / simulateur : l'API par défaut est http://localhost:18080 (10.0.2.2 sur émulateur Android)
npm run web

# Téléphone physique : n'utilisez JAMAIS localhost, mais l'IP LAN de votre machine
npm run start:phone                   # détecte l'IP LAN et la passe à Expo
# ou : EXPO_PUBLIC_API_URL=http://192.168.1.20:18080 npx expo start
```

Vérifiez avant de tester sur téléphone : `docker compose ps`, puis `curl http://<IP_LAN>:18080/health`.
Pour un build web servi depuis un navigateur, ajoutez son origine à `ALLOWED_ORIGINS` côté API.

## Variables d'environnement

Voir [`.env.example`](.env.example). Aucune valeur secrète n'est versionnée.

| Variable | Obligatoire | Description |
| --- | --- | --- |
| `JWT_SECRET` | oui | secret HS256, 32 caractères minimum, les valeurs « exemple » sont refusées |
| `DATABASE_URL` | oui | URL PostgreSQL |
| `PORT` | non | port d'écoute (défaut `8080`) |
| `APP_ENV` | non | `development` (défaut) ou `production` (exige `SMTP_*`, désactive le seed) |
| `UPLOAD_DIR` | non | stockage des photos (défaut `./data/uploads`, volume Docker en conteneur) |
| `ALLOWED_ORIGINS` | non | origines CORS / WebSocket autorisées (builds web) |
| `TRUSTED_PROXIES` | non | IP/CIDR des reverse proxies dont `X-Forwarded-For` est cru |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` | prod | e-mail de récupération. Sans `SMTP_HOST`, le code est affiché dans les logs de l'API (développement uniquement) |
| `DATABASE_URL_TEST` | tests | base dédiée aux tests d'intégration Go |
| `EXPO_PUBLIC_API_URL` | mobile | URL publique de l'API (jamais de secret dans une variable `EXPO_PUBLIC_*`) |

## Base de données et migrations

Les migrations SQL (`services/api/internal/platform/db/migrations`) sont embarquées dans le binaire et appliquées **automatiquement au démarrage**, une seule fois chacune, dans une transaction, sous verrou consultatif (plusieurs instances peuvent démarrer ensemble). L'historique est dans la table `schema_migrations`.

Une base issue de l'ancien boilerplate SaaS est migrée sur place : les tables obsolètes (organisations, abonnements, posts, commentaires, votes, fichiers) sont supprimées et les utilisateurs existants conservés.

Sans Docker : `createdb app && DATABASE_URL=postgres://… JWT_SECRET=… go run ./cmd/api` depuis `services/api`.

## Tests et qualité

```bash
# API (depuis services/api) — les tests d'intégration utilisent DATABASE_URL_TEST
createdb app_test
export DATABASE_URL_TEST='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'
gofmt -l .            # doit ne rien afficher
go vet ./...
go test -race ./...
go build ./cmd/api ./cmd/seed

# Mobile (depuis apps/mobile)
npm ci
npm run typecheck && npm run lint && npm run format:check && npm test

# Test de fumée de l'API en fonctionnement
node scripts/smoke.mjs http://localhost:18080

# Test navigateur de bout en bout (inscription → profil → découverte → like → match → conversation → suppression)
# l'API doit tourner avec ALLOWED_ORIGINS=http://localhost:8099
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web
npx playwright install chromium     # une fois
npm run e2e
```

La CI (`.github/workflows/ci.yml`) exécute tout cela, y compris la pile Docker et le test navigateur.

## Build et déploiement

- **API** : `docker build -t alba-api services/api` (image Alpine, utilisateur non-root, binaires `api` et `seed`). Fournissez `DATABASE_URL`, `JWT_SECRET`, `APP_ENV=production`, `SMTP_*`, un volume persistant sur `UPLOAD_DIR`, et placez l'API derrière un proxy TLS (renseignez `TRUSTED_PROXIES`). Une seule réplique est supportée telle quelle (hub temps réel et limiteur en mémoire — voir l'architecture pour monter en charge).
- **Web** : `EXPO_PUBLIC_API_URL=https://api.example.org npm run build:web` produit `apps/mobile/dist` (fichiers statiques).
- **iOS / Android** : builds via EAS (`eas build`) avec `EXPO_PUBLIC_API_URL` défini pour l'environnement ciblé.

## Principales décisions

- **Gratuit par construction** : aucune notion de plan, quota ou paiement dans le modèle de données ni dans l'API.
- **Éligibilité unique** : le fil de découverte et l'autorisation d'un swipe partagent la même requête SQL ; le client ne peut pas agir sur un profil que le serveur ne lui montrerait pas.
- **Vie privée de la localisation** : coordonnées arrondies à ~1 km à l'écriture, jamais renvoyées ; les autres voient une distance par tranches de 5 km. Position, distance et visibilité sont désactivables.
- **Photos** : validation par le contenu, limites de taille et de pixels avant décodage, ré-encodage JPEG (métadonnées/GPS supprimées), accès réservé aux personnes autorisées.
- **Une conversation = un match** : se désapparier ou bloquer ferme la conversation pour les deux personnes ; « supprimer » une conversation la masque seulement pour soi.
- **Premier choix définitif** : liker puis passer (ou l'inverse) sur la même personne est refusé ; rejouer la même action est idempotent.
- **Date de naissance immuable** après création, pour éviter de contourner la règle des 18 ans.
- **Modération** : les signalements sont stockés dans `reports` (statut `open` / `reviewed` / `dismissed`). Il n'y a pas d'interface d'administration ; exemple de revue et de suspension :

  ```sql
  SELECT r.created_at, r.reason, r.details, u.email FROM reports r JOIN users u ON u.id = r.reported_id WHERE r.status = 'open' ORDER BY r.created_at;
  UPDATE users SET status = 'suspended' WHERE id = '<uuid>';   -- connexion et découverte refusées
  UPDATE reports SET status = 'reviewed' WHERE reported_id = '<uuid>';
  ```
- **Limites connues** : pas de vérification d'adresse e-mail à l'inscription (la récupération de compte repose sur l'e-mail déclaré) ; un jeton d'accès (15 min) reste valide jusqu'à son expiration après une suspension ; les notifications push natives ne sont pas implémentées (temps réel tant que l'app est ouverte, notifications consultables ensuite).

## Contribution

Voir [`CONTRIBUTING.md`](CONTRIBUTING.md) et [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md).
