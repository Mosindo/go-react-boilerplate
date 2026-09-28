export const MAX_PHOTO_SIDE = 1280;
export const MAX_PHOTOS = 6;

/** Returns the resize target keeping the longest side <= max, or null when no resize is needed. */
export function computeResize(
  width: number,
  height: number,
  max: number = MAX_PHOTO_SIDE
): { width: number } | { height: number } | null {
  if (width <= max && height <= max) {
    return null;
  }
  return width >= height ? { width: max } : { height: max };
}

/** Moves the item at `index` by `delta` positions, returning a new array (or the same order if out of range). */
export function moveItem<T>(items: T[], index: number, delta: number): T[] {
  const target = index + delta;
  if (index < 0 || index >= items.length || target < 0 || target >= items.length) {
    return items;
  }
  const next = [...items];
  const [moved] = next.splice(index, 1);
  next.splice(target, 0, moved);
  return next;
}
