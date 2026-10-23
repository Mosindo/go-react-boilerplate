import { t } from "../i18n/fr";
import type { Gender } from "../api/types";

export function formatDistance(km: number | null): string | null {
  if (km === null || km === undefined) return null;
  return t.discover.distance(Math.max(1, Math.round(km)));
}

export function genderLabel(gender: Gender): string {
  return t.gender[gender];
}

export function genderPluralLabel(gender: Gender): string {
  return t.gender[`${gender}Plural` as const];
}

export function initialsOf(name: string): string {
  const trimmed = name.trim();
  return trimmed ? trimmed.charAt(0).toUpperCase() : "?";
}

export function messagePreview(
  last: { content: string; senderId: string } | null,
  myUserId: string | null
): string {
  if (!last) return t.matches.newMatch;
  return (last.senderId === myUserId ? t.matches.you : "") + last.content;
}

/** Toggles a value in a list, optionally capped to `max` entries. */
export function toggleInList<T>(list: T[], value: T, max = Infinity): T[] {
  if (list.includes(value)) return list.filter((item) => item !== value);
  return list.length >= max ? list : [...list, value];
}

/** Moves the item at `from` to `to` (clamped). Returns the same array if nothing changes. */
export function moveItem<T>(list: T[], from: number, to: number): T[] {
  if (from < 0 || from >= list.length) return list;
  const target = Math.max(0, Math.min(list.length - 1, to));
  if (target === from) return list;
  const next = [...list];
  const [item] = next.splice(from, 1);
  next.splice(target, 0, item);
  return next;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Rounds a coordinate to 2 decimals (~1 km); the server rounds further to ~2 km. */
export function roundCoordinate(value: number): number {
  return Math.round(value * 100) / 100;
}
