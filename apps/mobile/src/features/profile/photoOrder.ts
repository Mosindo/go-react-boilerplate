import type { Photo } from '../../api/types';

/** Moves a photo and renumbers positions; index 0 becomes the primary photo. */
export function movePhoto(photos: Photo[], from: number, to: number): Photo[] {
  if (to < 0 || to >= photos.length || from === to) return photos;
  const next = [...photos];
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next.map((photo, index) => ({ ...photo, position: index, isPrimary: index === 0 }));
}
