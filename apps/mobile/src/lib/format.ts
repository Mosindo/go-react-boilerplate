// Pure formatting helpers (French locale). Unit tested with node:test.

export const genderLabels = { woman: "Femme", man: "Homme", nonbinary: "Non-binaire" } as const;
export const genderPluralLabels = { woman: "Des femmes", man: "Des hommes", nonbinary: "Des personnes non-binaires" } as const;
export const goalLabels = {
  long_term: "Une relation sérieuse",
  short_term: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  undecided: "Je verrai bien"
} as const;
export const reportReasonLabels = {
  fake_profile: "Faux profil",
  inappropriate_content: "Contenu inapproprié",
  harassment: "Harcèlement ou menaces",
  spam: "Spam ou arnaque",
  underage: "Personne mineure",
  other: "Autre"
} as const;

export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) {
    return null;
  }
  return km <= 1 ? "À moins de 1 km" : `À ${km} km`;
}

function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

const pad = (n: number) => String(n).padStart(2, "0");

export function formatClock(date: Date): string {
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/** Short timestamp for lists: "14:05", "Hier", "lun.", or "12/03". */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (sameDay(date, now)) {
    return formatClock(date);
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) {
    return "Hier";
  }
  const diffDays = (now.getTime() - date.getTime()) / 86_400_000;
  if (diffDays < 7) {
    return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][date.getDay()] ?? "";
  }
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}

/** Day separator label inside a conversation. */
export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (sameDay(date, now)) {
    return "Aujourd’hui";
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) {
    return "Hier";
  }
  const months = ["janv.", "févr.", "mars", "avr.", "mai", "juin", "juil.", "août", "sept.", "oct.", "nov.", "déc."];
  const label = `${date.getDate()} ${months[date.getMonth()]}`;
  return date.getFullYear() === now.getFullYear() ? label : `${label} ${date.getFullYear()}`;
}

export function formatRelative(iso: string, now: Date = new Date()): string {
  const seconds = Math.max(0, (now.getTime() - new Date(iso).getTime()) / 1000);
  if (seconds < 60) {
    return "À l’instant";
  }
  if (seconds < 3600) {
    return `Il y a ${Math.floor(seconds / 60)} min`;
  }
  if (seconds < 86_400) {
    return `Il y a ${Math.floor(seconds / 3600)} h`;
  }
  return formatListTime(iso, now);
}

const errorMessages: Record<string, string> = {
  network: "Impossible de joindre Lueur. Vérifie ta connexion.",
  timeout: "Le serveur met trop de temps à répondre. Réessaie.",
  invalid_credentials: "E-mail ou mot de passe incorrect.",
  email_taken: "Un compte existe déjà avec cet e-mail.",
  invalid_email: "Cette adresse e-mail n’est pas valide.",
  weak_password: "Choisis un mot de passe d’au moins 8 caractères avec une lettre et un chiffre.",
  invalid_reset_code: "Code invalide ou expiré.",
  rate_limited: "Doucement ! Trop de tentatives, réessaie dans un instant.",
  underage: "Lueur est réservée aux personnes majeures.",
  birthdate_locked: "La date de naissance ne peut plus être modifiée.",
  profile_incomplete: "Termine ton profil (infos, préférences et au moins une photo).",
  already_swiped: "Tu as déjà répondu à ce profil.",
  photo_limit: "Tu peux ajouter jusqu’à 6 photos.",
  last_photo: "Garde au moins une photo : remplace-la plutôt.",
  unsupported_type: "Formats acceptés : JPEG, PNG ou WebP.",
  invalid_image: "Cette image est invalide ou trop petite (200 × 200 px minimum).",
  file_too_large: "La photo doit faire 10 Mo maximum.",
  not_found: "Ce contenu n’est plus disponible.",
  unauthorized: "Ta session a expiré. Reconnecte-toi.",
  internal_error: "Un problème est survenu de notre côté. Réessaie."
};

/** Turns any thrown value into a friendly French message. */
export function errorMessage(error: unknown, fallback = "Une erreur est survenue."): string {
  if (error && typeof error === "object" && "code" in error) {
    const code = String((error as { code: unknown }).code);
    if (errorMessages[code]) {
      return errorMessages[code];
    }
  }
  if (error instanceof Error && error.message && !/^Erreur \d+$/.test(error.message)) {
    return error.message;
  }
  return fallback;
}
