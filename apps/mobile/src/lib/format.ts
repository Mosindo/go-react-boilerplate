import { fr } from '../i18n';

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  const first = Array.from(parts[0])[0] ?? '?';
  const second = parts.length > 1 ? (Array.from(parts[1])[0] ?? '') : '';
  return (first + second).toLocaleUpperCase('fr-FR');
}

export function formatDistance(km: number | null | undefined): string | null {
  if (km === null || km === undefined) return null;
  return km < 1 ? fr.profile.distanceNear : fr.profile.distanceKm(Math.round(km));
}

export function formatUnread(count: number): string {
  return count > 99 ? '99+' : String(count);
}
