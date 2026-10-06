type ErrorLike = { name?: string; message?: string; status?: number; code?: string };

const MESSAGES_BY_API_MESSAGE: Record<string, string> = {
  "invalid credentials": "E-mail ou mot de passe incorrect.",
  "email already exists": "Un compte existe déjà avec cet e-mail.",
  "invalid email": "Cette adresse e-mail n'est pas valide.",
  "password too long": "Mot de passe trop long (72 caractères maximum).",
  "invalid or expired code": "Code invalide ou expiré.",
  "wrong password": "Mot de passe incorrect.",
  "you must be at least 18 years old": "Vous devez avoir 18 ans ou plus.",
  "photo limit reached (max 6)": "Vous avez atteint la limite de 6 photos.",
  "file too large (max 8 MB)": "Cette photo est trop volumineuse (8 Mo maximum).",
  "image too small (min 200px per side)": "Cette photo est trop petite (200 px minimum).",
  "unsupported image type (JPEG, PNG or WebP only)": "Format non pris en charge (JPEG, PNG ou WebP).",
  "message too long": "Message trop long (2000 caractères maximum)."
};

/** Turns API / network failures into a short, user-facing French message. */
export function describeError(error: unknown, fallback = "Une erreur est survenue. Réessayez."): string {
  const e = (typeof error === "object" && error !== null ? error : {}) as ErrorLike;

  if (e.name === "NetworkError") {
    return "Connexion impossible. Vérifiez votre réseau puis réessayez.";
  }
  if (e.status === 429) {
    return "Trop de tentatives. Patientez un instant avant de réessayer.";
  }
  if (e.status === 401) {
    return "Votre session a expiré. Reconnectez-vous.";
  }
  if (e.status !== undefined && e.status >= 500) {
    return "Le service rencontre un problème. Réessayez dans un instant.";
  }
  if (e.code === "profile_incomplete") {
    return "Terminez d'abord votre profil.";
  }
  if (e.status !== undefined && e.message) {
    return MESSAGES_BY_API_MESSAGE[e.message] ?? fallback;
  }
  return fallback;
}
