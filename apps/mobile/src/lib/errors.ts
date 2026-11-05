import { ApiError } from "../api/client";

const FRIENDLY: Record<string, string> = {
  "invalid credentials": "E-mail ou mot de passe incorrect.",
  "email already exists": "Un compte existe déjà avec cet e-mail.",
  "invalid email address": "Adresse e-mail invalide.",
  "invalid or expired reset token": "Ce code est invalide ou a expiré.",
  "current password is incorrect": "Le mot de passe actuel est incorrect.",
  "password is incorrect": "Mot de passe incorrect.",
  "photo limit reached": "Vous avez atteint 6 photos.",
  "only JPEG and PNG images are accepted": "Seules les images JPEG et PNG sont acceptées.",
  "too many requests": "Trop de tentatives. Réessayez dans un instant.",
  "profile not available": "Ce profil n'est plus disponible."
};

/** Turns any thrown value into a message that is safe and useful to display. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 0) return "Connexion impossible. Vérifiez votre réseau.";
    const known = FRIENDLY[err.message];
    if (known) return known;
    if (err.status >= 500) return "Un problème est survenu de notre côté. Réessayez.";
    if (err.status === 429) return FRIENDLY["too many requests"] ?? "Trop de requêtes.";
    return (
      err.message.charAt(0).toUpperCase() +
      err.message.slice(1) +
      (err.message.endsWith(".") ? "" : ".")
    );
  }
  return "Une erreur inattendue est survenue.";
}
