import type { Gender, RelationshipGoal, ReportReason } from "./api/types";

export const genderLabels: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const interestedInLabels: Record<Gender, string> = {
  woman: "Des femmes",
  man: "Des hommes",
  nonbinary: "Des personnes non-binaires"
};

export const goalLabels: Record<RelationshipGoal, string> = {
  long_term: "Une relation sérieuse",
  short_term: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  unsure: "Je verrai bien"
};

export const reportReasonLabels: Record<ReportReason, string> = {
  fake_profile: "Faux profil",
  inappropriate_content: "Contenu inapproprié",
  harassment: "Harcèlement ou menaces",
  spam: "Spam ou arnaque",
  underage: "Personne mineure",
  other: "Autre raison"
};

export function formatDistance(km: number | null | undefined): string | null {
  if (km == null) {
    return null;
  }
  return km <= 2 ? "À moins de 2 km" : `À ${km} km`;
}

/** Parses "JJ/MM/AAAA" into ISO "AAAA-MM-JJ", or null when invalid. */
export function parseFrenchDate(input: string): string | null {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!match) {
    return null;
  }
  const [, dd, mm, yyyy] = match;
  const date = new Date(Date.UTC(Number(yyyy), Number(mm) - 1, Number(dd)));
  if (date.getUTCFullYear() !== Number(yyyy) || date.getUTCMonth() !== Number(mm) - 1 || date.getUTCDate() !== Number(dd)) {
    return null;
  }
  return `${yyyy}-${mm}-${dd}`;
}

/** Formats digits typed by the user as "JJ/MM/AAAA" progressively. */
export function maskDateInput(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

export function ageFromIso(iso: string, now = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) {
    age -= 1;
  }
  return age;
}

export function formatMessageTime(iso: string, now = new Date()): string {
  const date = new Date(iso);
  const sameDay = date.toDateString() === now.toDateString();
  const time = date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  if (sameDay) {
    return time;
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) {
    return `Hier ${time}`;
  }
  return `${date.toLocaleDateString("fr-FR", { day: "numeric", month: "short" })} ${time}`;
}

export function formatRelative(iso: string, now = new Date()): string {
  const seconds = Math.max(0, Math.round((now.getTime() - new Date(iso).getTime()) / 1000));
  if (seconds < 60) {
    return "À l'instant";
  }
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `Il y a ${minutes} min`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 24) {
    return `Il y a ${hours} h`;
  }
  const days = Math.round(hours / 24);
  if (days < 7) {
    return `Il y a ${days} j`;
  }
  return new Date(iso).toLocaleDateString("fr-FR", { day: "numeric", month: "short" });
}

export const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function passwordProblem(password: string): string | null {
  if (password.length < 8) {
    return "8 caractères minimum.";
  }
  if (password.length > 72) {
    return "72 caractères maximum.";
  }
  if (!/\d/.test(password) || !/[^\d\s]/.test(password)) {
    return "Utilisez au moins une lettre et un chiffre.";
  }
  return null;
}
