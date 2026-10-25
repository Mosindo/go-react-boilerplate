import type { Photo, ProfileState } from "../../api";

/** Keeps `position` contiguous from 0 and flags the first photo as primary. */
export function normalizePhotos(photos: Photo[]): Photo[] {
  return photos.map((photo, index) => ({ ...photo, position: index, isPrimary: index === 0 }));
}

function withPhotos(state: ProfileState | undefined, update: (photos: Photo[]) => Photo[]): ProfileState | undefined {
  if (!state?.profile) {
    return state;
  }
  const photos = normalizePhotos(update(state.profile.photos));
  return { ...state, profile: { ...state.profile, photos } };
}

export function addPhoto(state: ProfileState | undefined, photo: Photo): ProfileState | undefined {
  return withPhotos(state, (photos) => [...photos.filter((existing) => existing.id !== photo.id), photo]);
}

/** The API issues a new id when a picture is replaced, so the entry is matched on the old id. */
export function replacePhoto(state: ProfileState | undefined, oldId: string, photo: Photo): ProfileState | undefined {
  return withPhotos(state, (photos) => photos.map((existing) => (existing.id === oldId ? photo : existing)));
}

export function removePhoto(state: ProfileState | undefined, photoId: string): ProfileState | undefined {
  return withPhotos(state, (photos) => photos.filter((photo) => photo.id !== photoId));
}

export function setPhotoOrder(state: ProfileState | undefined, ordered: Photo[]): ProfileState | undefined {
  return withPhotos(state, () => ordered);
}

/** Returns the ids after moving `photoId` by `offset` slots (clamped), or null if it cannot move. */
export function movePhotoId(photos: Photo[], photoId: string, offset: number): string[] | null {
  const ids = photos.map((photo) => photo.id);
  const from = ids.indexOf(photoId);
  if (from === -1) {
    return null;
  }
  const to = Math.min(Math.max(from + offset, 0), ids.length - 1);
  if (to === from) {
    return null;
  }
  ids.splice(from, 1);
  ids.splice(to, 0, photoId);
  return ids;
}
