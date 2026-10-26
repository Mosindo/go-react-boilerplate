/** Distance label shown on cards: the API only returns 5 km steps. */
export function distanceLabel(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  return km <= 5 ? "À moins de 5 km" : `À moins de ${km} km`;
}

export function genderLabel(g: string): string {
  return g === "man" ? "Homme" : g === "woman" ? "Femme" : "Non-binaire";
}

export function interestedLabel(g: string): string {
  return g === "man" ? "Des hommes" : g === "woman" ? "Des femmes" : "Des personnes non-binaires";
}

export function timeLabel(iso: string, now = new Date()): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const sameDay = d.toDateString() === now.toDateString();
  if (sameDay) return d.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  const diffDays = Math.floor((now.getTime() - d.getTime()) / 86_400_000);
  if (diffDays < 7) return d.toLocaleDateString("fr-FR", { weekday: "short" });
  return d.toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" });
}

/** Parses "JJ/MM/AAAA" into ISO "AAAA-MM-JJ"; null when impossible. */
export function parseBirthDate(input: string): string | null {
  const m = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(input.trim());
  if (!m) return null;
  const [, dd, mm, yyyy] = m;
  const day = Number(dd);
  const month = Number(mm);
  const year = Number(yyyy);
  const date = new Date(Date.UTC(year, month - 1, day));
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day)
    return null;
  return `${yyyy}-${mm}-${dd}`;
}

export function toDisplayDate(iso: string): string {
  const [y, m, d] = iso.split("-");
  return y && m && d ? `${d}/${m}/${y}` : iso;
}

export function ageFromIso(iso: string, now = new Date()): number {
  const [y, m, d] = iso.split("-").map(Number);
  if (!y || !m || !d) return 0;
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) age -= 1;
  return age;
}

/** Inserts slashes while typing a date: 12031990 -> 12/03/1990. */
export function maskDate(input: string): string {
  const digits = input.replace(/\D/g, "").slice(0, 8);
  const parts = [digits.slice(0, 2), digits.slice(2, 4), digits.slice(4, 8)].filter(Boolean);
  return parts.join("/");
}

export function validEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
}

export function validPassword(password: string): string | null {
  if (password.length < 8) return "8 caractères minimum.";
  if (password.length > 72) return "72 caractères maximum.";
  if (!/\p{L}/u.test(password) || !/\d/.test(password)) return "Utilisez au moins une lettre et un chiffre.";
  return null;
}
