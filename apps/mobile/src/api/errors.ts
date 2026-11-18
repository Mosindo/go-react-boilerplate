import { ApiError, NetworkError } from "./client";

const messages: Record<string, string> = {
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  email_exists: "Un compte existe déjà avec cet e-mail.",
  invalid_email: "Cette adresse e-mail n’est pas valide.",
  weak_password: "Le mot de passe doit contenir 8 caractères minimum, dont une lettre et un chiffre.",
  invalid_reset_code: "Code invalide ou expiré.",
  underage: "Lueur est réservée aux personnes majeures (18 ans et plus).",
  birthdate_locked: "La date de naissance ne peut plus être modifiée.",
  profile_required: "Complétez d’abord votre profil.",
  photo_limit: "Vous pouvez ajouter jusqu’à 6 photos.",
  unsupported_type: "Format non pris en charge. Utilisez une photo JPEG, PNG ou WebP.",
  image_too_large: "Cette photo est trop lourde (10 Mo maximum).",
  image_too_small: "Cette photo est trop petite (200 × 200 pixels minimum).",
  invalid_image: "Ce fichier n’est pas une image valide.",
  rate_limited: "Doucement ! Réessayez dans quelques instants.",
  already_swiped: "Vous avez déjà répondu à ce profil.",
  not_found: "Ce contenu n’est plus disponible.",
  session_expired: "Votre session a expiré. Reconnectez-vous."
};

/** Turns any thrown value into a short French message for the UI. */
export function errorMessage(error: unknown, fallback = "Une erreur est survenue. Réessayez."): string {
  if (error instanceof ApiError) {
    if (messages[error.code]) {
      return messages[error.code];
    }
    if (error.code === "validation_failed" || error.code === "invalid_request") {
      return `Vérifiez les informations saisies (${error.message}).`;
    }
    return error.status >= 500 ? "Le service est momentanément indisponible." : fallback;
  }
  if (error instanceof NetworkError) {
    return error.message;
  }
  return fallback;
}

export function isApiError(error: unknown, code: string): boolean {
  return error instanceof ApiError && error.code === code;
}
