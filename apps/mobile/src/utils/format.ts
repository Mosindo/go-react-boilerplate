import type { Gender, RelationshipGoal, ReportReason } from "../api/types";

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
  fake_profile: "Faux profil ou arnaque",
  inappropriate_content: "Photos ou contenu inapproprié",
  harassment: "Harcèlement ou propos insultants",
  spam: "Spam ou publicité",
  underage: "Personne mineure",
  other: "Autre raison"
};

export function formatDistance(km: number | null): string | null {
  if (km === null) {
    return null;
  }
  return km <= 1 ? "À moins d'1 km" : `À ${km} km`;
}

const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });

/** Short timestamp for lists: "14:05", "Hier", "12 nov.". */
export function formatShortTime(iso: string, now = new Date()): string {
  const date = new Date(iso);
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  if (date.getTime() >= startOfToday) {
    return timeFormatter.format(date);
  }
  if (date.getTime() >= startOfToday - 86_400_000) {
    return "Hier";
  }
  return dayFormatter.format(date);
}

export function formatMessageTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

/** "Il y a 5 min" style relative time for notifications. */
export function formatRelative(iso: string, now = Date.now()): string {
  const diff = Math.max(0, now - new Date(iso).getTime());
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) {
    return "À l'instant";
  }
  if (minutes < 60) {
    return `Il y a ${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `Il y a ${hours} h`;
  }
  const days = Math.floor(hours / 24);
  if (days < 7) {
    return `Il y a ${days} j`;
  }
  return dayFormatter.format(new Date(iso));
}

/** Validates a DD/MM/YYYY triple and returns an ISO date or null. */
export function toIsoDate(day: string, month: string, year: string): string | null {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return null;
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return null;
  }
  return `${y}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

export function ageFromIso(iso: string, now = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) {
    age -= 1;
  }
  return age;
}
