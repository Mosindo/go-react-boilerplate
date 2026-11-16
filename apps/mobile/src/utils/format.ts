import type { Gender, LookingFor, ReportReason } from "../api/types";

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

export const lookingForLabels: Record<LookingFor, string> = {
  relationship: "Une relation sérieuse",
  casual: "Quelque chose de léger",
  friendship: "De nouvelles amitiés",
  unsure: "Je verrai bien"
};

export const lookingForShort: Record<LookingFor, string> = {
  relationship: "Relation sérieuse",
  casual: "Léger",
  friendship: "Amitié",
  unsure: "Ouvert·e"
};

export const reportReasonLabels: Record<ReportReason, string> = {
  fake_profile: "Faux profil ou usurpation",
  inappropriate_content: "Photos ou contenu inapproprié",
  harassment: "Harcèlement ou propos offensants",
  spam: "Spam ou arnaque",
  underage: "Personne mineure",
  other: "Autre raison"
};

export function distanceLabel(km: number | null | undefined): string | null {
  if (km === null || km === undefined) {
    return null;
  }
  return km <= 1 ? "À moins de 2 km" : `À ${km} km`;
}

const timeFormatter = new Intl.DateTimeFormat("fr-FR", { hour: "2-digit", minute: "2-digit" });
const dayFormatter = new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "short" });

export function messageTime(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

export function relativeTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diffMin = Math.round((now.getTime() - date.getTime()) / 60_000);
  if (diffMin < 1) {
    return "À l'instant";
  }
  if (diffMin < 60) {
    return `${diffMin} min`;
  }
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) {
    return timeFormatter.format(date);
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) {
    return "Hier";
  }
  return dayFormatter.format(date);
}

/** Parses a French-style JJ/MM/AAAA date into ISO YYYY-MM-DD, or null. */
export function parseBirthdate(input: string): string | null {
  const m = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!m) {
    return null;
  }
  const [, dd, mm, yyyy] = m;
  const date = new Date(Date.UTC(Number(yyyy), Number(mm) - 1, Number(dd)));
  if (date.getUTCFullYear() !== Number(yyyy) || date.getUTCMonth() !== Number(mm) - 1 || date.getUTCDate() !== Number(dd)) {
    return null;
  }
  return `${yyyy}-${mm}-${dd}`;
}

export function formatBirthdate(iso: string): string {
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : "";
}

/** Auto-inserts slashes while typing a JJ/MM/AAAA date. */
export function maskBirthdate(raw: string): string {
  const digits = raw.replace(/\D/g, "").slice(0, 8);
  if (digits.length <= 2) {
    return digits;
  }
  if (digits.length <= 4) {
    return `${digits.slice(0, 2)}/${digits.slice(2)}`;
  }
  return `${digits.slice(0, 2)}/${digits.slice(2, 4)}/${digits.slice(4)}`;
}

export function ageFrom(isoDate: string, now: Date = new Date()): number {
  const [y, m, d] = isoDate.split("-").map(Number);
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) {
    age -= 1;
  }
  return age;
}
