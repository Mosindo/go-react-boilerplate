import type { Gender, RelationshipGoal, ReportReason } from "../api/types";

export const genderLabels: Record<Gender, string> = {
  woman: "Femme",
  man: "Homme",
  nonbinary: "Non-binaire"
};

export const genderPluralLabels: Record<Gender, string> = {
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
  other: "Autre"
};

export function distanceLabel(km: number | null): string | null {
  if (km === null) return null;
  if (km <= 1) return "À moins de 2 km";
  return `À ${km} km`;
}

function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

const pad = (n: number) => String(n).padStart(2, "0");

/** "14:05" today, "Hier" yesterday, "lun." this week, otherwise "12/03". */
export function shortTimestamp(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  if (sameDay(date, now)) return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) return "Hier";
  const diffDays = (now.getTime() - date.getTime()) / 86_400_000;
  if (diffDays < 7) {
    return ["dim.", "lun.", "mar.", "mer.", "jeu.", "ven.", "sam."][date.getDay()];
  }
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}`;
}

export function timeOfDay(iso: string): string {
  const date = new Date(iso);
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function dayHeader(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (sameDay(date, now)) return "Aujourd'hui";
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) return "Hier";
  return date.toLocaleDateString("fr-FR", { weekday: "long", day: "numeric", month: "long" });
}

export function relativeTime(iso: string, now: Date = new Date()): string {
  const seconds = Math.max(0, Math.round((now.getTime() - new Date(iso).getTime()) / 1000));
  if (seconds < 60) return "À l'instant";
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `Il y a ${minutes} min`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `Il y a ${hours} h`;
  const days = Math.round(hours / 24);
  if (days < 7) return `Il y a ${days} j`;
  return shortTimestamp(iso, now);
}
