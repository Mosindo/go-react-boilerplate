import type { Photo } from "../api/types";

export const MAX_PHOTO_BYTES = 8 * 1024 * 1024;
export const MAX_PHOTO_DIMENSION = 8000;
const ALLOWED_MIME = new Set(["image/jpeg", "image/png", "image/webp"]);

export type PickedAsset = { width?: number; height?: number; fileSize?: number; mimeType?: string | null };

/** Client-side mirror of the server upload constraints. Returns an error message or null. */
export function validatePickedPhoto(asset: PickedAsset): string | null {
  if (asset.mimeType && !ALLOWED_MIME.has(asset.mimeType.toLowerCase())) {
    return "Please choose a JPEG, PNG or WebP image.";
  }
  if (asset.fileSize && asset.fileSize > MAX_PHOTO_BYTES) return "That photo is larger than 8 MB.";
  if ((asset.width ?? 0) > MAX_PHOTO_DIMENSION || (asset.height ?? 0) > MAX_PHOTO_DIMENSION) {
    return "That photo is too large (max 8000 px per side).";
  }
  return null;
}

/** Returns the ids in a new order after moving the photo at `from` to `to`. */
export function moveId(photos: Photo[], from: number, to: number): string[] {
  const ids = photos.map((photo) => photo.id);
  if (from < 0 || from >= ids.length || to < 0 || to >= ids.length || from === to) return ids;
  const [moved] = ids.splice(from, 1);
  ids.splice(to, 0, moved);
  return ids;
}
