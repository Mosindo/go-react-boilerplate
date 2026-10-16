export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) {
    return null;
  }
  return km <= 1 ? "À moins de 1 km" : `À ${km} km`;
}

export function formatRelativeTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diffSeconds = Math.round((now.getTime() - date.getTime()) / 1000);
  if (Number.isNaN(diffSeconds)) {
    return "";
  }
  if (diffSeconds < 60) {
    return "À l'instant";
  }
  const minutes = Math.floor(diffSeconds / 60);
  if (minutes < 60) {
    return `Il y a ${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `Il y a ${hours} h`;
  }
  const days = Math.floor(hours / 24);
  if (days < 7) {
    return `Il y a ${days} j`;
  }
  return date.toLocaleDateString("fr-FR", { day: "numeric", month: "short" });
}

export function formatClock(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return date.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
}

export function formatDayLabel(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const sameDay = (a: Date, b: Date) => a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
  if (sameDay(date, now)) {
    return "Aujourd'hui";
  }
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (sameDay(date, yesterday)) {
    return "Hier";
  }
  return date.toLocaleDateString("fr-FR", { weekday: "long", day: "numeric", month: "long" });
}

export function pluralize(count: number, one: string, many: string): string {
  return count > 1 ? many : one;
}
