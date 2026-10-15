// Maps stable API error codes to French messages. Unknown codes fall back to a generic text:
// server messages are in English and must never be shown to users as-is.

const messages: Record<string, string> = {
  // auth
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  email_exists: "Un compte existe déjà avec cet e-mail.",
  invalid_email: "Cette adresse e-mail n'est pas valide.",
  invalid_password: "Le mot de passe doit contenir entre 8 et 72 caractères.",
  wrong_password: "Le mot de passe est incorrect.",
  invalid_reset_code: "Ce code est invalide ou a expiré.",
  invalid_refresh_token: "Votre session a expiré, reconnectez-vous.",
  session_expired: "Votre session a expiré, reconnectez-vous.",
  unauthorized: "Votre session a expiré, reconnectez-vous.",
  user_not_found: "Ce compte est introuvable.",
  // profile
  invalid_first_name: "Le prénom doit contenir entre 1 et 40 caractères.",
  invalid_birth_date: "La date de naissance n'est pas valide.",
  underage: "Alma est réservée aux personnes de 18 ans et plus.",
  birth_date_locked: "La date de naissance ne peut pas être modifiée.",
  invalid_gender: "Ce genre n'est pas valide.",
  invalid_bio: "La présentation est trop longue (500 caractères maximum).",
  invalid_interests: "Choisissez au plus 10 centres d'intérêt.",
  invalid_preferences: "Ces préférences ne sont pas valides.",
  invalid_age_range: "La tranche d'âge doit être comprise entre 18 et 99 ans.",
  invalid_distance: "La distance doit être comprise entre 1 et 500 km.",
  invalid_location: "Cette position n'est pas valide.",
  profile_incomplete: "Terminez votre profil (infos, préférences et au moins une photo) pour continuer.",
  profile_not_found: "Ce profil n'est plus disponible.",
  // photos
  photo_limit: "Vous pouvez ajouter jusqu'à 6 photos.",
  unsupported_image: "Seules les images JPEG, PNG ou WebP sont acceptées.",
  image_too_large: "Cette image est trop volumineuse.",
  image_too_small: "Cette image est trop petite (200 px minimum).",
  invalid_order: "L'ordre des photos n'est pas valide.",
  photo_not_found: "Cette photo est introuvable.",
  // matching & chat
  already_swiped: "Vous avez déjà fait un choix pour cette personne.",
  match_not_found: "Ce match n'existe plus.",
  conversation_not_found: "Cette conversation n'existe plus.",
  empty_message: "Écrivez un message avant d'envoyer.",
  message_too_long: "Ce message est trop long (2000 caractères maximum).",
  // safety
  invalid_reason: "Choisissez un motif de signalement.",
  invalid_details: "Les précisions sont trop longues (1000 caractères maximum).",
  invalid_target: "Cette action n'est pas possible.",
  // transport
  rate_limited: "Trop de requêtes, patientez un instant.",
  network: "Impossible de joindre le serveur. Vérifiez votre connexion.",
  timeout: "Le serveur met trop de temps à répondre. Réessayez.",
  not_configured: "L'adresse du serveur n'est pas configurée (EXPO_PUBLIC_API_URL).",
  invalid_request: "Certaines informations ne sont pas valides.",
  unavailable: "Le service est momentanément indisponible.",
  internal: "Une erreur est survenue de notre côté. Réessayez dans un instant."
};

export const GENERIC_ERROR = "Une erreur est survenue. Réessayez.";

type CodedError = { code?: unknown };

export function errorCodeOf(error: unknown): string | undefined {
  const code = (error as CodedError | null)?.code;
  return typeof code === "string" ? code : undefined;
}

export function errorMessage(error: unknown, fallback: string = GENERIC_ERROR): string {
  const code = errorCodeOf(error);
  return (code && messages[code]) || fallback;
}
