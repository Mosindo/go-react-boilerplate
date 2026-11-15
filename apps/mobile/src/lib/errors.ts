import { ApiError } from "../api/client";

// Server error codes mapped to user-facing French copy.
const messages: Record<string, string> = {
  network: "Connexion impossible. Vérifiez votre réseau et réessayez.",
  config: "L'application n'est pas configurée pour joindre le serveur.",
  invalid_request: "Certaines informations sont invalides.",
  invalid_email: "Adresse e-mail invalide.",
  email_exists: "Un compte existe déjà avec cet e-mail.",
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  weak_password: "Le mot de passe doit contenir au moins 8 caractères, dont une lettre et un chiffre.",
  terms_required: "Vous devez confirmer avoir 18 ans ou plus.",
  invalid_reset_code: "Code invalide ou expiré.",
  wrong_password: "Mot de passe incorrect.",
  rate_limited: "Trop de tentatives. Patientez un instant.",
  invalid_first_name: "Le prénom ne doit contenir que des lettres (40 max).",
  invalid_birthdate: "Date de naissance invalide.",
  birthdate_locked: "La date de naissance ne peut plus être modifiée.",
  underage: "Lueur est réservée aux personnes majeures (18 ans et plus).",
  invalid_gender: "Genre invalide.",
  bio_too_long: "La bio est limitée à 500 caractères.",
  too_many_interests: "10 centres d'intérêt maximum.",
  invalid_location: "Position invalide.",
  invalid_preferences: "Préférences invalides.",
  unsupported_image_type: "Formats acceptés : JPEG, PNG ou WebP.",
  image_too_large: "Image trop lourde (10 Mo maximum).",
  invalid_image_dimensions: "Image trop petite (200 × 200 pixels minimum).",
  invalid_image: "Ce fichier n'est pas une image valide.",
  too_many_photos: "6 photos maximum.",
  profile_incomplete: "Complétez votre profil pour découvrir d'autres personnes.",
  profile_unavailable: "Ce profil n'est plus disponible.",
  already_swiped: "Vous avez déjà répondu à ce profil.",
  conversation_not_found: "Cette conversation n'existe plus.",
  empty_message: "Le message est vide.",
  message_too_long: "Message trop long (2000 caractères maximum).",
  profile_not_found: "Ce profil n'est plus disponible.",
  internal: "Une erreur est survenue. Réessayez dans un instant."
};

export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError) {
    return messages[error.code] ?? (error.status >= 500 ? messages.internal : error.message || fallback);
  }
  if (error instanceof Error && error.message) return fallback;
  return fallback;
}

export function isApiError(error: unknown, code: string): boolean {
  return error instanceof ApiError && error.code === code;
}
