# Décisions produit et techniques

Ce document explique brièvement les choix faits lors de la transformation du boilerplate en application de rencontre, et les limites connues.

## Audit initial (résumé)

| Constat | Décision |
| --- | --- |
| API Go/Gin/pgx bien structurée (handler → service → repository), JWT HS256 strict, bcrypt, refresh tokens rotatifs | **Conservée** et étendue |
| Billing Stripe | **Retiré** : contraire au principe « 100 % gratuit » |
| Multi-tenant `organizations` | **Retiré** : sans objet pour une app de rencontre, complexifiait chaque requête |
| Posts, commentaires, votes | **Retirés** : hors produit |
| `/users` listait les emails de tous les utilisateurs | **Retiré** : fuite de données personnelles |
| Module `files` acceptant une `storage_key` fournie par le client | **Remplacé** par un module photos sûr |
| Migrations rejouées intégralement à chaque démarrage, sans suivi | Suivi par version (`schema_migrations`) + verrou consultatif |
| `c.ClientIP()` faisant confiance à tout `X-Forwarded-For` | Proxies de confiance explicites (`TRUSTED_PROXIES`) |
| Front : Tamagui installé mais inutilisé, pas de refresh automatique du jeton (TTL 15 min), polling du chat | Nouveau design system léger, client API avec refresh transparent, WebSocket |

## Produit

- **Âge** : 18 ans minimum, vérifié côté serveur. La date de naissance est **verrouillée une fois saisie** pour empêcher la manipulation de l'âge ; seul l'âge est exposé.
- **Profil complet** (condition pour découvrir et être découvert) : prénom, date de naissance, genre, au moins un genre recherché, au moins une photo. Bio, intérêts, métier, ville et localisation sont facultatifs.
- **Préférences mutuelles** : un profil n'apparaît que si chacun correspond aux critères de l'autre (genres, tranche d'âge, distance).
- **Swipe définitif** : un like ou un « passer » n'est pas réversible, ce qui garantit qu'un profil ne réapparaît jamais. Annuler un match transforme le like en « passer ».
- **« Qui m'a liké »** : non exposé en liste (pression sociale, confidentialité). À la place, les personnes qui vous ont liké sont classées en tête de votre découverte, gratuitement.
- **Signaler bloque automatiquement** la personne pour protéger immédiatement l'auteur du signalement. Un profil signalé par 3 personnes distinctes est masqué de la découverte jusqu'à revue.
- **Blocage** : supprime le match et la conversation ; la personne bloquée n'est pas notifiée (elle voit simplement le match disparaître).
- **Suppression locale d'une conversation** : la conversation disparaît de votre liste jusqu'au prochain message, et les anciens messages restent masqués pour vous seulement.
- **Notifications de message** dédupliquées : une seule notification non lue par conversation.
- **Mise en pause** : un profil non « découvrable » disparaît de la découverte, mais les matchs existants restent accessibles.
- **Rate limiting** : uniquement anti-abus (ex. 120 swipes/min, 60 messages/min), jamais un quota produit.

- **Modération** : le rôle modérateur n'est attribuable qu'en ligne de commande (`cmd/admin`), jamais via l'API ; il est revérifié en base à chaque requête. L'identité du signaleur n'est pas montrée au modérateur. Suspendre un compte révoque ses sessions, refuse la connexion, le retire de la découverte et clôture ses signalements ouverts ; les modérateurs ne peuvent pas être suspendus via l'API.

- **Push** : envoyés en arrière-plan (jamais bloquants pour la requête), uniquement pour les notifications créées (donc dédupliquées pour les messages), sans le contenu des messages ; en premier plan, l'app n'affiche pas de bannière système (le temps réel et les badges suffisent). Les jetons signalés `DeviceNotRegistered` sont supprimés, et le jeton de l'appareil est retiré à la déconnexion.

## Technique

- **Temps réel** : WebSocket authentifié par un ticket JWT de 60 s (type distinct du jeton d'accès, pour ne jamais placer un jeton longue durée dans une URL). La diffusion passe par PostgreSQL `LISTEN/NOTIFY`, ce qui fonctionne avec plusieurs instances sans Redis. Si un événement dépasse la limite de `NOTIFY` (8 Ko), il est envoyé tronqué et le client recharge la ressource.
- **Révocation immédiate** : chaque requête authentifiée vérifie que la session existe encore (déconnexion, réinitialisation du mot de passe et suppression de compte prennent effet sans attendre l'expiration du jeton).
- **Photos** : validation par le contenu (pas l'extension ni le type déclaré), protection contre les bombes de décompression, correction de l'orientation EXIF, redimensionnement (1440 px max), ré-encodage JPEG qui supprime toutes les métadonnées (dont la position GPS) et neutralise les fichiers polyglottes. Stockées hors de toute racine web avec des noms aléatoires, servies uniquement via des URLs signées HMAC valables 1 à 2 h (fenêtres alignées pour garder le cache d'images efficace). Le nombre de décodages simultanés est borné.
- **Localisation** : coordonnées arrondies à 2 décimales (~1,1 km) côté appareil et côté serveur ; distance affichée arrondie (≥ 2 km, paliers de 5 km au-delà de 10 km), masquable.
- **Matching concurrent** : un verrou transactionnel par paire d'utilisateurs garantit un match unique même si deux likes arrivent simultanément (testé).
- **Pas de N+1** : les cartes de profils sont construites en un nombre constant de requêtes (profils, photos, intérêts chargés par lot).
- **Pagination** : par curseur (keyset) pour les messages et conversations.
- **Classement** : interface `Scorer` (réciprocité, intérêts communs, proximité, activité récente, petit bonus nouveaux membres). La requête SQL applique les filtres stricts et renvoie un pool de 100 candidats.
- **Récupération de compte** : code à 6 chiffres haché, valable 30 min, 5 essais max, un envoi par minute et par compte ; la réponse ne révèle jamais si l'email existe ; toutes les sessions sont révoquées après changement.
- **Frontend** : le web (react-native-web) est supporté pour permettre des tests E2E réels dans un navigateur ; sur le web, la session est gardée en `sessionStorage` (jamais `localStorage`).

## Limites connues et pistes

- **Notifications push natives** : implémentées via Expo Push et testées contre un faux serveur Expo ; la livraison réelle sur appareil nécessite un projet EAS et des identifiants APNs/FCM, et n'a pas pu être vérifiée dans cet environnement.
- **Modération** : outillage volontairement minimal (file de signalements, suspension). Pas d'historique d'actions en interface (les actions sont journalisées côté serveur), pas de sanctions graduées.
- **Vérification d'email** à l'inscription non implémentée (l'email sert à la récupération de compte).
- **Stockage des photos** sur disque local (volume). Pour plusieurs instances, implémenter `storage.Store` vers un stockage objet (S3, GCS…).
- **Rate limiting** en mémoire, par instance.
- Une URL de photo déjà signée reste valable jusqu'à son expiration (≤ 2 h), même après un blocage.
- Sur le web, la couleur du curseur des interrupteurs suit le style par défaut de react-native-web.
