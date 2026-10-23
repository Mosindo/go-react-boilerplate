export const MAX_UPLOAD_SIDE = 1080;
export const MIN_PHOTO_SIDE = 64;

/** Returns the resize target keeping the aspect ratio, or null when the image already fits. */
export function computeResize(
  width: number,
  height: number,
  max: number = MAX_UPLOAD_SIDE
): { width: number } | { height: number } | null {
  if (width <= max && height <= max) return null;
  return width >= height ? { width: max } : { height: max };
}

export function isTooSmall(width: number, height: number): boolean {
  return width < MIN_PHOTO_SIDE || height < MIN_PHOTO_SIDE;
}
