import { ApiError } from "./client";

// The API answers in English with stable messages; the app shows French copy.
const KNOWN_MESSAGES: readonly [needle: string, french: string][] = [
  ["email already exists", "Cette adresse e-mail est déjà utilisée."],
  ["invalid credentials", "E-mail ou mot de passe incorrect."],
  ["invalid email address", "Cette adresse e-mail n'est pas valide."],
  ["password must be between 8 and 72", "Le mot de passe doit contenir entre 8 et 72 caractères."],
  ["invalid or expired code", "Code invalide ou expiré."],
  ["too many requests", "Trop de tentatives. Réessayez dans un instant."],
  ["at least 18", "Vous devez avoir au moins 18 ans."],
  ["birth date cannot be changed", "La date de naissance ne peut pas être modifiée."],
  ["at most 6 photos", "Vous pouvez ajouter 6 photos au maximum."],
  ["8 mb or less", "L'image doit peser 8 Mo au maximum."],
  ["only jpeg and png", "Seules les images JPEG et PNG sont acceptées."],
  ["image dimensions", "Les dimensions de l'image ne conviennent pas (200 px minimum)."],
  ["image could not be read", "Cette image n'a pas pu être lue."],
  ["password is incorrect", "Mot de passe incorrect."],
  ["message must contain", "Le message doit contenir entre 1 et 2000 caractères."],
  ["create your profile first", "Créez d'abord votre profil."],
  ["complete your profile", "Terminez d'abord votre profil."],
  ["not found", "Cet élément n'existe plus."],
  ["profile not available", "Ce profil n'est plus disponible."]
];

export function friendlyError(error: unknown, fallback = "Une erreur est survenue. Réessayez."): string {
  if (error instanceof ApiError) {
    const raw = typeof error.data === "object" && error.data !== null ? (error.data as { error?: unknown }).error : undefined;
    const message = typeof raw === "string" ? raw.toLowerCase() : "";
    const match = KNOWN_MESSAGES.find(([needle]) => message.includes(needle));
    if (match) {
      return match[1];
    }
    if (error.status === 429) {
      return "Trop de requêtes. Patientez un instant.";
    }
    if (error.status === 401) {
      return "Votre session a expiré. Reconnectez-vous.";
    }
    if (error.status >= 500) {
      return "Le service est momentanément indisponible. Réessayez plus tard.";
    }
    if (error.status === 400 || error.status === 422) {
      return typeof raw === "string" && raw ? raw : fallback;
    }
    return fallback;
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
