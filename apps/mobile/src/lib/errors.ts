import { ApiError, NetworkError } from "../api/client";

const MESSAGES: Record<string, string> = {
  email_taken: "Cette adresse e-mail est déjà utilisée.",
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  invalid_code: "Code invalide ou expiré.",
  underage: "Vous devez avoir au moins 18 ans pour utiliser Ember.",
  photo_limit: "Vous avez atteint le maximum de 6 photos.",
  image_too_large: "Cette image est trop lourde (8 Mo maximum).",
  invalid_image: "Image non supportée. Utilisez une photo JPEG ou PNG d'au moins 200 px.",
  invalid_message: "Le message doit contenir entre 1 et 2000 caractères.",
  rate_limited: "Trop de tentatives. Patientez un instant avant de réessayer.",
  profile_incomplete: "Complétez votre profil et ajoutez une photo pour continuer.",
  profile_unavailable: "Ce profil n'est plus disponible.",
  conversation_not_found: "Cette conversation n'existe plus.",
  match_not_found: "Ce match n'existe plus."
};

/** Turns any thrown value into a short, user-facing French sentence. */
export function errorMessage(error: unknown): string {
  if (error instanceof NetworkError) {
    return "Connexion impossible. Vérifiez votre réseau et réessayez.";
  }
  if (error instanceof ApiError) {
    return (
      (error.code && MESSAGES[error.code]) ||
      (error.status >= 500 ? "Un problème est survenu de notre côté. Réessayez." : error.message)
    );
  }
  return "Une erreur inattendue est survenue.";
}
