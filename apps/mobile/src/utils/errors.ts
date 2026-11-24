import { ApiError } from "../api/client";

// Server error codes mapped to user-facing French copy.
const messages: Record<string, string> = {
  network: "Connexion impossible. Vérifie ton réseau et réessaie.",
  timeout: "Le serveur met trop de temps à répondre. Réessaie.",
  missing_api_url: "L'application n'est pas configurée (EXPO_PUBLIC_API_URL).",
  rate_limited: "Doucement ! Patiente quelques secondes avant de réessayer.",
  internal_error: "Un problème est survenu de notre côté. Réessaie dans un instant.",
  invalid_request: "Certaines informations sont invalides.",
  invalid_email: "Cette adresse e-mail n'est pas valide.",
  email_taken: "Un compte existe déjà avec cet e-mail.",
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  weak_password: "Le mot de passe doit faire au moins 8 caractères, avec une lettre et un chiffre.",
  invalid_reset_code: "Ce code est invalide ou a expiré.",
  session_revoked: "Ta session a expiré. Reconnecte-toi.",
  invalid_token: "Ta session a expiré. Reconnecte-toi.",
  invalid_first_name: "Ton prénom doit contenir uniquement des lettres (40 max).",
  invalid_birthdate: "Cette date de naissance n'est pas valide.",
  underage: "Lueur est réservée aux personnes majeures (18 ans et plus).",
  invalid_gender: "Choisis un genre.",
  bio_too_long: "Ta bio doit faire 500 caractères maximum.",
  job_title_too_long: "Ton métier doit faire 80 caractères maximum.",
  city_too_long: "Le nom de ville doit faire 80 caractères maximum.",
  invalid_height: "La taille doit être comprise entre 120 et 230 cm.",
  invalid_location: "Position invalide.",
  profile_required: "Crée d'abord ton profil.",
  invalid_age_range: "La tranche d'âge doit être comprise entre 18 et 99 ans.",
  invalid_distance: "La distance doit être comprise entre 1 et 300 km.",
  invalid_interested_in: "Choisis au moins un genre recherché.",
  too_many_interests: "Choisis 10 centres d'intérêt maximum.",
  unsupported_image: "Formats acceptés : JPEG, PNG ou WebP.",
  image_too_large: "La photo doit peser 10 Mo maximum.",
  invalid_image_dimensions: "La photo est trop petite ou trop grande.",
  corrupt_image: "Cette photo est illisible.",
  photo_limit_reached: "Tu peux ajouter jusqu'à 6 photos.",
  photo_required: "Sélectionne une photo.",
  profile_incomplete: "Complète ton profil (photo et localisation) pour découvrir des personnes.",
  already_swiped: "Tu as déjà répondu à ce profil.",
  profile_unavailable: "Ce profil n'est plus disponible.",
  profile_not_found: "Ce profil n'est plus disponible.",
  conversation_not_found: "Cette conversation n'existe plus.",
  match_not_found: "Ce match n'existe plus.",
  empty_message: "Ton message est vide.",
  message_too_long: "Ton message doit faire 2000 caractères maximum.",
  invalid_reason: "Choisis un motif de signalement.",
  details_too_long: "Les détails doivent faire 1000 caractères maximum."
};

export function errorMessage(error: unknown, fallback = "Une erreur est survenue. Réessaie."): string {
  if (error instanceof ApiError) {
    return messages[error.code] ?? fallback;
  }
  return fallback;
}

export function errorCode(error: unknown): string | null {
  return error instanceof ApiError ? error.code : null;
}
