/** "il y a 5 min", "hier", "12 mars" — pour listes de conversations et notifications. */
export function formatRelative(iso: string, now: Date = new Date()): string {
  const then = new Date(iso);
  const diffMs = now.getTime() - then.getTime();
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) {
    return "À l'instant";
  }
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours} h`;
  }
  const days = Math.floor(hours / 24);
  if (days === 1) {
    return "Hier";
  }
  if (days < 7) {
    return `${days} j`;
  }
  return then.toLocaleDateString("fr-FR", { day: "numeric", month: "short" });
}

/** Heure d'envoi d'un message ("14:32"). */
export function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
}

export function formatDistance(km: number | null): string | null {
  if (km === null) {
    return null;
  }
  return km <= 5 ? "À moins de 5 km" : `À environ ${km} km`;
}

export function initials(name: string): string {
  return name.trim().charAt(0).toUpperCase();
}
