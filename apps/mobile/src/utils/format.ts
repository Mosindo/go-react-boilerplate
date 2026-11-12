import type { Gender, RelationshipGoal, ReportReason } from "../api/types";

export const genderLabels: Record<Gender, string> = { woman: "Femme", man: "Homme", nonbinary: "Non-binaire" };
export const seekingLabels: Record<Gender, string> = { woman: "Des femmes", man: "Des hommes", nonbinary: "Des personnes non-binaires" };

export const goalLabels: Record<RelationshipGoal, string> = {
  long_term: "Une relation durable",
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

export function distanceLabel(km?: number): string | null {
  if (km === undefined || km === null) return null;
  return km <= 1 ? "À moins de 2 km" : `À ${km} km`;
}

export function nameAge(firstName: string, age?: number): string {
  return age ? `${firstName}, ${age}` : firstName;
}

const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });

export function messageTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

export function relativeTime(iso: string, now = Date.now()): string {
  const diff = Math.max(0, now - new Date(iso).getTime());
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) return "À l'instant";
  if (minutes < 60) return `${minutes} min`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} h`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days} j`;
  return dayFormatter.format(new Date(iso));
}

export function dayLabel(iso: string, now = new Date()): string {
  const d = new Date(iso);
  const sameDay = (a: Date, b: Date) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  if (sameDay(d, now)) return "Aujourd'hui";
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(d, yesterday)) return "Hier";
  return new Intl.DateTimeFormat("fr-FR", { weekday: "long", day: "numeric", month: "long" }).format(d);
}

/** Parses a DD/MM/YYYY entry into ISO and checks it is a real date for an adult (18–99). */
export function parseBirthdate(day: string, month: string, year: string, now = new Date()): { iso?: string; error?: string } {
  const d = Number(day);
  const m = Number(month);
  const y = Number(year);
  if (!Number.isInteger(d) || !Number.isInteger(m) || !Number.isInteger(y) || year.length !== 4) {
    return { error: "Date de naissance incomplète." };
  }
  const date = new Date(Date.UTC(y, m - 1, d));
  if (date.getUTCFullYear() !== y || date.getUTCMonth() !== m - 1 || date.getUTCDate() !== d) {
    return { error: "Cette date n'existe pas." };
  }
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) age--;
  if (age < 18) return { error: "Lueur est réservé aux personnes majeures (18 ans et plus)." };
  if (age > 99) return { error: "Vérifiez l'année de naissance." };
  return { iso: `${year}-${month.padStart(2, "0")}-${day.padStart(2, "0")}` };
}

export function validatePassword(password: string): string | null {
  if (password.length < 8) return "8 caractères minimum.";
  if (!/[A-Za-zÀ-ÿ]/.test(password) || !/\d/.test(password)) return "Au moins une lettre et un chiffre.";
  return null;
}

export function validateEmail(email: string): string | null {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()) ? null : "Adresse email invalide.";
}
