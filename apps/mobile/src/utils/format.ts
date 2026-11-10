import type { Gender, RelationshipGoal, ReportReason } from "../api/types";

export const genderLabels: Record<Gender, string> = { woman: "Femme", man: "Homme", nonbinary: "Non-binaire" };
export const interestedInLabels: Record<Gender, string> = { woman: "Des femmes", man: "Des hommes", nonbinary: "Des personnes non-binaires" };

export const goalLabels: Record<RelationshipGoal, string> = {
  long_term: "Une relation sérieuse",
  short_term: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  undecided: "Je verrai bien"
};

export const reportReasonLabels: Record<ReportReason, string> = {
  fake_profile: "Faux profil",
  inappropriate_content: "Contenu inapproprié",
  harassment: "Harcèlement ou insultes",
  scam: "Arnaque ou spam",
  underage: "Personne mineure",
  other: "Autre raison"
};

export function formatDistance(km: number | null): string | null {
  if (km === null) return null;
  return km <= 1 ? "À moins de 2 km" : `À ${km} km`;
}

const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });

export function formatTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

/** Short relative label for lists: "à l'instant", "5 min", "14:32", "hier", "3 mars". */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diffMin = Math.floor((now.getTime() - date.getTime()) / 60_000);
  if (diffMin < 1) return "à l'instant";
  if (diffMin < 60) return `${diffMin} min`;
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) return timeFormatter.format(date);
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) return "hier";
  return dayFormatter.format(date);
}

export function formatDayHeader(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (date.toDateString() === now.toDateString()) return "Aujourd'hui";
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) return "Hier";
  return new Intl.DateTimeFormat("fr-FR", { weekday: "long", day: "numeric", month: "long" }).format(date);
}
