export function formatDistance(km: number | undefined): string | null {
  if (km === undefined) return null;
  return km <= 1 ? "à moins de 1 km" : `à ${km} km`;
}

export function formatTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) {
    return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) return "Hier";
  return date.toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" });
}

export const GENDER_LABELS = { woman: "Femme", man: "Homme", nonbinary: "Non-binaire" } as const;
