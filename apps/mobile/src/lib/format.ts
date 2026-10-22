const DAY_MS = 24 * 60 * 60 * 1000;

/** "14:05" for today, "hier", weekday within a week, otherwise "12/03". */
export function formatShortTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) {
    return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  }
  const days = Math.floor((startOfDay(now) - startOfDay(date)) / DAY_MS);
  if (days === 1) {
    return "Hier";
  }
  if (days > 1 && days < 7) {
    return date.toLocaleDateString("fr-FR", { weekday: "short" });
  }
  return date.toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" });
}

export function formatClock(iso: string): string {
  return new Date(iso).toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
}

export function formatDayHeading(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  if (date.toDateString() === now.toDateString()) {
    return "Aujourd'hui";
  }
  const days = Math.floor((startOfDay(now) - startOfDay(date)) / DAY_MS);
  if (days === 1) {
    return "Hier";
  }
  return date.toLocaleDateString("fr-FR", { weekday: "long", day: "numeric", month: "long" });
}

export function isSameDay(a: string, b: string): boolean {
  return new Date(a).toDateString() === new Date(b).toDateString();
}

function startOfDay(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime();
}

export function distanceLabel(km?: number): string | null {
  if (km === undefined) {
    return null;
  }
  return km <= 5 ? "À moins de 5 km" : `À environ ${km} km`;
}

export function genderLabel(gender: string): string {
  switch (gender) {
    case "man":
      return "Homme";
    case "woman":
      return "Femme";
    case "non_binary":
      return "Non-binaire";
    default:
      return gender;
  }
}
