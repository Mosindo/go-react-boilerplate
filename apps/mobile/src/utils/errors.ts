import { ApiError, NetworkError } from "../api/client";

// Server messages are stable English strings; translate the ones members can
// trigger so every error is shown in French.
const translations: Record<string, string> = {
  "invalid credentials": "E-mail ou mot de passe incorrect.",
  "an account already exists for this e-mail": "Un compte existe déjà avec cet e-mail.",
  "enter a valid e-mail address": "Saisissez une adresse e-mail valide.",
  "enter a valid e-mail and a password of 8 to 72 characters":
    "Saisissez un e-mail valide et un mot de passe de 8 à 72 caractères.",
  "invalid or expired code": "Code invalide ou expiré.",
  "incorrect password": "Mot de passe incorrect.",
  "first name must be 1 to 40 letters": "Le prénom doit contenir 1 à 40 lettres.",
  "birthdate must be a valid YYYY-MM-DD date": "Date de naissance invalide.",
  "you must be at least 18 years old": "Vous devez avoir au moins 18 ans pour utiliser Aimant.",
  "birthdate is not plausible": "Date de naissance invalide.",
  "bio must be at most 500 characters": "La bio est limitée à 500 caractères.",
  "job title must be at most 60 characters": "Le métier est limité à 60 caractères.",
  "height must be between 120 and 230 cm": "La taille doit être comprise entre 120 et 230 cm.",
  "choose at most 10 interests": "Choisissez au maximum 10 centres d'intérêt.",
  "choose at least one gender you are interested in": "Choisissez au moins un genre.",
  "age range must be within 18-99 and min <= max": "Tranche d'âge invalide.",
  "distance must be between 1 and 500 km": "La distance doit être comprise entre 1 et 500 km.",
  "only JPEG, PNG and WebP images are accepted": "Seules les images JPEG, PNG et WebP sont acceptées.",
  "image must be at least 320x320 pixels": "L'image doit faire au moins 320 × 320 pixels.",
  "image must be at most 10 MB": "L'image ne doit pas dépasser 10 Mo.",
  "image could not be decoded": "Cette image est illisible.",
  "a profile can have at most 6 photos": "Vous pouvez ajouter jusqu'à 6 photos.",
  "conversation not found": "Cette conversation n'est plus disponible.",
  "profile not available": "Ce profil n'est plus disponible.",
  "profile not found": "Ce profil n'est plus disponible.",
  "complete your profile to discover people": "Complétez votre profil pour découvrir des personnes.",
  "too many requests, please slow down": "Trop de tentatives. Patientez quelques instants."
};

export function describeError(error: unknown, fallback = "Une erreur est survenue. Réessayez."): string {
  if (error instanceof NetworkError) {
    return error.message;
  }
  if (error instanceof ApiError) {
    if (error.status === 429) {
      return translations["too many requests, please slow down"];
    }
    if (error.status >= 500) {
      return "Le service rencontre un problème. Réessayez dans un instant.";
    }
    return translations[error.message] ?? fallback;
  }
  return fallback;
}

export function isApiError(error: unknown, status: number): boolean {
  return error instanceof ApiError && error.status === status;
}
