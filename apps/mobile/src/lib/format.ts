export function distanceLabel(km?: number): string | null {
  if (km === undefined) {
    return null;
  }
  return km <= 1 ? "À moins de 1 km" : `À environ ${km} km`;
}

export function genderLabel(gender: string): string {
  switch (gender) {
    case "woman":
      return "Femme";
    case "man":
      return "Homme";
    default:
      return "Non-binaire";
  }
}

export function timeLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const sameDay = date.toDateString() === now.toDateString();
  if (sameDay) {
    return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  }
  const diffDays = Math.floor((now.getTime() - date.getTime()) / 86_400_000);
  if (diffDays < 7) {
    return date.toLocaleDateString("fr-FR", { weekday: "short" });
  }
  return date.toLocaleDateString("fr-FR", { day: "2-digit", month: "2-digit" });
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}
