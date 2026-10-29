// The API speaks English (stable, machine friendly); the app shows French.
// Unknown messages fall back to the server text.

const EXACT: Record<string, string> = {
  "an account with this email already exists": "Un compte existe déjà avec cette adresse e-mail.",
  "invalid email or password": "Adresse e-mail ou mot de passe incorrect.",
  "invalid email address": "Cette adresse e-mail n'est pas valide.",
  "invalid or expired code": "Code invalide ou expiré.",
  "invalid refresh token": "Votre session a expiré. Reconnectez-vous.",
  "invalid token": "Votre session a expiré. Reconnectez-vous.",
  "missing token": "Votre session a expiré. Reconnectez-vous.",
  "incorrect password": "Mot de passe incorrect.",
  "password cannot be blank": "Le mot de passe ne peut pas être vide.",
  "keep at least one photo on your profile": "Gardez au moins une photo sur votre profil.",
  "you already answered this profile": "Vous avez déjà répondu à ce profil.",
  "complete your profile before discovering people":
    "Terminez votre profil avant de découvrir des personnes.",
  "profile not available": "Ce profil n'est plus disponible.",
  "profile not found": "Ce profil est introuvable.",
  "profile not created yet": "Votre profil n'est pas encore créé.",
  "preferences not set yet": "Vos préférences ne sont pas encore définies.",
  "conversation not found": "Cette conversation n'existe plus.",
  "match not found": "Ce match n'existe plus.",
  "photo not found": "Photo introuvable.",
  "notification not found": "Notification introuvable.",
  "user not found": "Utilisateur introuvable.",
  "message cannot be empty": "Le message est vide.",
  "message contains invalid characters": "Le message contient des caractères non autorisés.",
  "image is too small (minimum 200x200)": "L'image est trop petite (200 × 200 px minimum).",
  "image dimensions are too large": "Les dimensions de l'image sont trop grandes.",
  "unsupported image: use JPEG, PNG or WebP":
    "Format non pris en charge : utilisez JPEG, PNG ou WebP.",
  "the image file is corrupted": "Le fichier image est corrompu.",
  "empty file": "Le fichier est vide.",
  "could not read the uploaded file": "Impossible de lire le fichier envoyé.",
  "birth date is not valid": "La date de naissance n'est pas valide.",
  "coordinates are out of range": "La position n'est pas valide.",
  "choose at least one gender you are interested in": "Choisissez au moins une option.",
  "max distance must be between 1 and 500 km":
    "La distance maximale doit être comprise entre 1 et 500 km.",
  "first name contains invalid characters": "Le prénom contient des caractères non autorisés.",
  "unknown interest": "Centre d'intérêt inconnu.",
  "unknown report reason": "Motif de signalement inconnu.",
  "you cannot swipe on yourself": "Action impossible sur votre propre profil.",
  "you cannot do this to yourself": "Action impossible sur votre propre compte.",
  "too many requests, please slow down":
    "Trop de requêtes, patientez un instant avant de réessayer.",
  "internal error": "Une erreur est survenue côté serveur. Réessayez dans un instant."
};

const PATTERNS: [RegExp, (match: RegExpMatchArray) => string][] = [
  [
    /^you must be at least (\d+) years old/,
    (m) => `Vous devez avoir au moins ${m[1]} ans pour utiliser Sillage.`
  ],
  [
    /^password must be at least (\d+) characters/,
    (m) => `Le mot de passe doit contenir au moins ${m[1]} caractères.`
  ],
  [/^password must be at most/, () => "Le mot de passe est trop long."],
  [/^you can upload at most (\d+) photos/, (m) => `Vous pouvez ajouter ${m[1]} photos au maximum.`],
  [/^photo must be smaller than (\d+) MB/, (m) => `La photo doit peser moins de ${m[1]} Mo.`],
  [/^bio must be at most (\d+)/, (m) => `La bio est limitée à ${m[1]} caractères.`],
  [/^city must be at most (\d+)/, (m) => `La ville est limitée à ${m[1]} caractères.`],
  [/^message must be at most (\d+)/, (m) => `Les messages sont limités à ${m[1]} caractères.`],
  [
    /^you can pick at most (\d+) interests/,
    (m) => `Vous pouvez choisir ${m[1]} centres d'intérêt au maximum.`
  ],
  [/^first name must be between/, () => "Le prénom doit contenir entre 1 et 40 caractères."],
  [/^age range must be within/, () => "La tranche d'âge doit être comprise entre 18 et 99 ans."],
  [/^details must be at most (\d+)/, (m) => `Les détails sont limités à ${m[1]} caractères.`]
];

export function translateApiError(message: string): string {
  const exact = EXACT[message];
  if (exact) {
    return exact;
  }
  for (const [pattern, build] of PATTERNS) {
    const match = message.match(pattern);
    if (match) {
      return build(match);
    }
  }
  return message;
}
