const timeFmt = new Intl.DateTimeFormat('fr-FR', { hour: '2-digit', minute: '2-digit' });
const dayFmt = new Intl.DateTimeFormat('fr-FR', { day: 'numeric', month: 'short' });
const weekdayFmt = new Intl.DateTimeFormat('fr-FR', { weekday: 'short' });
const longDayFmt = new Intl.DateTimeFormat('fr-FR', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
});

function startOfDay(d: Date): number {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

export function formatClock(iso: string): string {
  return timeFmt.format(new Date(iso));
}

/** Compact timestamp for conversation lists: 14:05 / Hier / lun. / 3 mars */
export function formatListTime(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diffDays = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (diffDays <= 0) return timeFmt.format(date);
  if (diffDays === 1) return 'Hier';
  if (diffDays < 7) return weekdayFmt.format(date);
  return dayFmt.format(date);
}

export function formatDayHeader(iso: string, now: Date = new Date()): string {
  const date = new Date(iso);
  const diffDays = Math.round((startOfDay(now) - startOfDay(date)) / 86_400_000);
  if (diffDays <= 0) return 'Aujourd’hui';
  if (diffDays === 1) return 'Hier';
  return longDayFmt.format(date);
}

export function isSameDay(a: string, b: string): boolean {
  return startOfDay(new Date(a)) === startOfDay(new Date(b));
}

let counter = 0;
export function localId(prefix = 'tmp'): string {
  counter += 1;
  return `${prefix}-${Date.now().toString(36)}-${counter}`;
}
