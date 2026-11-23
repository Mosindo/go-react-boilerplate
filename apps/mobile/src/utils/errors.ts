import { isApiError } from "../api/client";

const messages: Record<string, string> = {
  invalid_email: "Adresse email invalide.",
  weak_password: "Le mot de passe doit faire au moins 8 caractères avec une lettre et un chiffre.",
  email_exists: "Un compte existe déjà avec cet email.",
  invalid_credentials: "Email ou mot de passe incorrect.",
  invalid_reset_code: "Code invalide ou expiré.",
  wrong_password: "Mot de passe incorrect.",
  rate_limited: "Trop de tentatives. Patiente un instant.",
  invalid_first_name: "Ton prénom ne doit contenir que des lettres (40 max).",
  invalid_birthdate: "Date de naissance invalide.",
  underage: "Hearth est réservé aux personnes majeures (18 ans et plus).",
  invalid_gender: "Choisis une option.",
  bio_too_long: "Ta bio est limitée à 500 caractères.",
  too_many_interests: "10 centres d'intérêt maximum.",
  invalid_interested_in: "Choisis au moins une option.",
  invalid_age_range: "Tranche d'âge invalide (18 à 99 ans).",
  invalid_distance: "Distance invalide.",
  unsupported_type: "Format non supporté. Utilise une photo JPEG, PNG ou WebP.",
  image_too_large: "Photo trop lourde (10 Mo maximum).",
  image_too_small: "Photo trop petite (200 × 200 pixels minimum).",
  invalid_image: "Ce fichier n'est pas une image valide.",
  too_many_photos: "6 photos maximum.",
  already_swiped: "Tu as déjà répondu à ce profil.",
  not_available: "Ce profil n'est plus disponible.",
  invalid_message: "Ton message est vide ou trop long (2000 caractères max).",
  not_found: "Introuvable ou plus disponible.",
  network: "Connexion impossible. Vérifie ton réseau.",
  timeout: "Le serveur met trop de temps à répondre."
};

export function errorMessage(error: unknown, fallback = "Une erreur est survenue. Réessaie."): string {
  if (isApiError(error)) {
    return messages[error.code] ?? fallback;
  }
  return fallback;
}
