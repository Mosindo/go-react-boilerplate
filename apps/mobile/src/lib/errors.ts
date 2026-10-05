import { ApiError, NetworkError } from "../api/client";

const MESSAGES: Record<string, string> = {
  invalid_credentials: "Email ou mot de passe incorrect.",
  email_exists: "Un compte existe déjà avec cet email.",
  weak_password: "Choisissez un mot de passe d'au moins 8 caractères.",
  invalid_email: "Cette adresse email n'est pas valide.",
  invalid_reset_token: "Ce lien a expiré ou a déjà été utilisé. Demandez-en un nouveau.",
  underage: "Vous devez avoir au moins 18 ans pour utiliser l'application.",
  birth_date_locked: "La date de naissance ne peut plus être modifiée.",
  profile_incomplete: "Terminez d'abord votre profil.",
  already_swiped: "Vous avez déjà réagi à ce profil.",
  not_available: "Ce profil n'est plus disponible.",
  photo_limit: "Vous pouvez ajouter 6 photos au maximum.",
  unsupported_type: "Seules les images JPEG et PNG sont acceptées.",
  invalid_image: "Cette image est illisible ou trop petite.",
  file_too_large: "Cette photo est trop lourde (8 Mo maximum).",
  rate_limited: "Trop de tentatives. Patientez un instant puis réessayez.",
  empty_message: "Écrivez un message avant d'envoyer.",
  message_too_long: "Ce message est trop long (2000 caractères maximum)."
};

/** Turns any thrown value into a message that is safe and useful to show to a person. */
export function errorMessage(error: unknown, fallback = "Une erreur est survenue. Réessayez."): string {
  if (error instanceof NetworkError) return error.message;
  if (error instanceof ApiError) {
    if (error.code in MESSAGES) return MESSAGES[error.code];
    if (error.status === 429) return MESSAGES.rate_limited;
    if (error.status >= 500) return "Le serveur rencontre un problème. Réessayez dans un instant.";
    if (error.status === 404) return "Élément introuvable.";
    return error.message || fallback;
  }
  return fallback;
}
