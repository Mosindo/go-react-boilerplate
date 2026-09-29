import { resolvePhotoUrl } from "../api/client";
import type { Photo } from "../api/models";

/** Absolute, ready-to-render URL for a photo, or null when there is none. */
export function photoUri(photo: Photo | null | undefined): string | null {
  return photo ? resolvePhotoUrl(photo.url) : null;
}
