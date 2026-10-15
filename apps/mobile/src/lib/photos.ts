import type { PhotoView } from "../api/types";

export const MAX_PHOTOS = 6;

export function sortedPhotos(photos: PhotoView[]): PhotoView[] {
  return [...photos].sort((a, b) => a.position - b.position);
}

/** Moves one photo to a new index and returns the ids in their new order. */
export function moveId(ids: string[], from: number, to: number): string[] {
  if (from === to || from < 0 || to < 0 || from >= ids.length || to >= ids.length) {
    return ids;
  }
  const next = [...ids];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
}
