import { APP_NAME, MIN_AGE } from "../config";
import { ApiError } from "./http";

const MESSAGES: Record<string, string> = {
  invalid_request: "Certaines informations sont invalides. Vérifiez votre saisie.",
  invalid_credentials: "Email ou mot de passe incorrect.",
  email_taken: "Un compte existe déjà avec cet email.",
  weak_password: "Mot de passe trop faible. Utilisez au moins 8 caractères.",
  invalid_code: "Code invalide ou expiré.",
  underage: `${APP_NAME} est réservé aux personnes majeures (${MIN_AGE} ans et plus).`,
  photo_limit: "Vous avez atteint le maximum de 6 photos.",
  invalid_image: "Cette image n'est pas valide. Utilisez un JPEG ou un PNG.",
  payload_too_large: "Ce fichier est trop volumineux (8 Mo maximum).",
  rate_limited: "Trop de tentatives. Réessayez dans un instant.",
  unauthorized: "Votre session a expiré. Reconnectez-vous.",
  forbidden: "Action non autorisée.",
  profile_incomplete: "Complétez votre profil pour continuer.",
  not_found: "Ce contenu n'est plus disponible.",
  conflict: "Cette action est en conflit avec l'état actuel.",
  network_error: "Connexion au serveur impossible. Vérifiez votre réseau.",
  timeout: "Le serveur met trop de temps à répondre.",
  internal: "Une erreur est survenue de notre côté. Réessayez plus tard."
};

export function errorCode(error: unknown): string | null {
  return error instanceof ApiError ? error.code : null;
}

/** Turns any thrown value into a French, user-facing sentence. */
export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error instanceof ApiError) {
    return MESSAGES[error.code] ?? fallback;
  }
  return fallback;
}
