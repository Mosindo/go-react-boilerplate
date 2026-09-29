export const MAX_PHOTOS = 6;
export const MAX_UPLOAD_BYTES = 5 * 1024 * 1024;

export type OrderedPhoto = { id: string; position: number };

export function sortPhotos<T extends OrderedPhoto>(photos: readonly T[]): T[] {
  return [...photos].sort((a, b) => a.position - b.position);
}

export function photoIds(photos: readonly OrderedPhoto[]): string[] {
  return sortPhotos(photos).map((photo) => photo.id);
}

export function remainingSlots(count: number): number {
  return Math.max(0, MAX_PHOTOS - count);
}

export function canAddPhoto(count: number): boolean {
  return count < MAX_PHOTOS;
}

export function movePhoto(ids: readonly string[], from: number, to: number): string[] {
  if (from === to || from < 0 || to < 0 || from >= ids.length || to >= ids.length) {
    return [...ids];
  }
  const next = [...ids];
  const [moved] = next.splice(from, 1);
  if (moved === undefined) {
    return [...ids];
  }
  next.splice(to, 0, moved);
  return next;
}

export function makeMain(ids: readonly string[], id: string): string[] {
  const index = ids.indexOf(id);
  return index < 0 ? [...ids] : movePhoto(ids, index, 0);
}

export function isPermutation(a: readonly string[], b: readonly string[]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  const sortedA = [...a].sort();
  const sortedB = [...b].sort();
  return sortedA.every((value, index) => value === sortedB[index]);
}

export function sameOrder(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((value, index) => value === b[index]);
}

const MIME_BY_EXTENSION: Record<string, string> = {
  jpg: "image/jpeg",
  jpeg: "image/jpeg",
  png: "image/png",
  webp: "image/webp",
  heic: "image/jpeg",
  heif: "image/jpeg"
};

/** Picks an upload MIME type. The server sniffs content, so this only has to be plausible. */
export function guessImageMime(uri: string, reported?: string | null): string {
  if (reported && /^image\/(jpeg|png|webp)$/.test(reported)) {
    return reported;
  }
  const extension = uri.split("?")[0]?.split(".").pop()?.toLowerCase() ?? "";
  return MIME_BY_EXTENSION[extension] ?? "image/jpeg";
}

export function uploadFileName(uri: string, reported?: string | null, index = 0): string {
  const mime = guessImageMime(uri, null);
  const extension = mime === "image/png" ? "png" : mime === "image/webp" ? "webp" : "jpg";
  const trimmed = reported?.trim();
  if (trimmed && /\.(jpe?g|png|webp)$/i.test(trimmed)) {
    return trimmed;
  }
  return `photo-${Date.now()}-${index}.${extension}`;
}

export function isTooLarge(sizeBytes: number | null | undefined): boolean {
  return typeof sizeBytes === "number" && sizeBytes > MAX_UPLOAD_BYTES;
}
