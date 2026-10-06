const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

export function formatClock(iso: string): string {
  const date = new Date(iso);
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${hours}:${minutes}`;
}

/** Short relative time for lists: "à l'instant", "5 min", "3 h", "hier", "12/03". */
export function formatRelative(iso: string, now: number = Date.now()): string {
  const date = new Date(iso);
  const diff = now - date.getTime();
  if (diff < MINUTE) {
    return "à l'instant";
  }
  if (diff < HOUR) {
    return `${Math.floor(diff / MINUTE)} min`;
  }
  if (diff < DAY) {
    return `${Math.floor(diff / HOUR)} h`;
  }
  if (diff < 2 * DAY) {
    return "hier";
  }
  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  return `${day}/${month}`;
}

export function formatDistance(km: number | undefined): string | null {
  if (km === undefined) {
    return null;
  }
  return km <= 5 ? "À moins de 5 km" : `À environ ${km} km`;
}

const GENDER_LABELS: Record<string, string> = {
  man: "Homme",
  woman: "Femme",
  non_binary: "Non-binaire"
};

export function genderLabel(gender: string): string {
  return GENDER_LABELS[gender] ?? gender;
}

/** "Léa, 28" or just "Léa" when the age is hidden. */
export function nameWithAge(name: string, age: number | undefined): string {
  return age === undefined ? name : `${name}, ${age}`;
}

export function previewText(text: string, max = 60): string {
  const flat = text.replace(/\s+/g, " ").trim();
  return flat.length > max ? `${flat.slice(0, max - 1)}…` : flat;
}
