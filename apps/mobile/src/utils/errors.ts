import { ApiError, NetworkError } from "../api/client";

const byCode: Record<string, string> = {
  email_exists: "Un compte existe déjà avec cet email.",
  invalid_email: "Adresse email invalide.",
  weak_password: "Le mot de passe doit contenir au moins 8 caractères, dont une lettre et un chiffre.",
  invalid_credentials: "Email ou mot de passe incorrect.",
  invalid_reset_code: "Code invalide ou expiré.",
  rate_limited: "Trop de tentatives. Patiente un instant avant de réessayer.",
  profile_required: "Complète d'abord ton profil.",
  profile_incomplete: "Ajoute au moins une photo et tes préférences pour découvrir des profils.",
  photo_limit: "Tu peux ajouter jusqu'à 6 photos.",
  invalid_image: "Cette image n'est pas acceptée (JPEG, PNG ou WebP, 200 px minimum).",
  already_swiped: "Tu as déjà répondu à ce profil.",
  not_found: "Ce contenu n'est plus disponible.",
  invalid_message: "Ton message est vide ou trop long.",
  unauthorized: "Ta session a expiré. Reconnecte-toi."
};

export function errorMessage(error: unknown, fallback = "Une erreur est survenue. Réessaie."): string {
  if (error instanceof ApiError) {
    if (error.code === "validation_failed") {
      return translateValidation(error.message);
    }
    return byCode[error.code] ?? fallback;
  }
  if (error instanceof NetworkError) {
    return error.message;
  }
  return fallback;
}

function translateValidation(message: string): string {
  if (message.includes("at least 18")) {
    return "Tu dois avoir au moins 18 ans pour utiliser Lueur.";
  }
  if (message.includes("birthdate")) {
    return "Date de naissance invalide.";
  }
  if (message.includes("first name")) {
    return "Prénom invalide (lettres uniquement, 40 caractères max).";
  }
  if (message.includes("age range")) {
    return "Tranche d'âge invalide.";
  }
  if (message.includes("interested")) {
    return "Choisis au moins un genre qui t'intéresse.";
  }
  if (message.includes("interests")) {
    return "Choisis au maximum 10 centres d'intérêt.";
  }
  return "Certaines informations sont invalides.";
}
