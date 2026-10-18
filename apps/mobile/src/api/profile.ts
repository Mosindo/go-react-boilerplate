import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { MyProfile, Photo, ProfileInput, PublicProfile } from "./types";

export function getMyProfile(): Promise<MyProfile> {
  return apiRequest<MyProfile>(endpoints.profile.me);
}

export function saveMyProfile(input: ProfileInput): Promise<MyProfile> {
  return apiRequest<MyProfile>(endpoints.profile.me, { method: "PUT", body: input });
}

export function updateSettings(settings: { isVisible?: boolean; showDistance?: boolean }): Promise<MyProfile> {
  return apiRequest<MyProfile>(endpoints.profile.settings, { method: "PATCH", body: settings });
}

export function getProfile(userId: string): Promise<PublicProfile> {
  return apiRequest<PublicProfile>(endpoints.profile.other(userId));
}

/** A picked local file: on web a Blob-backed uri, on native a file:// uri. */
export type PickedImage = { uri: string; mimeType?: string | null; fileName?: string | null };

async function toFormData(image: PickedImage): Promise<FormData> {
  const form = new FormData();
  // A Blob works on web and React Native alike. The server never trusts the
  // declared type: it sniffs the bytes and re-encodes the image.
  const blob = await fetch(image.uri).then((r) => r.blob());
  form.append("file", blob, image.fileName ?? "photo.jpg");
  return form;
}

export async function uploadPhoto(image: PickedImage): Promise<Photo> {
  return apiRequest<Photo>(endpoints.profile.photos, { method: "POST", formData: await toFormData(image) });
}

export async function replacePhoto(photoId: string, image: PickedImage): Promise<Photo> {
  return apiRequest<Photo>(endpoints.profile.photo(photoId), { method: "PUT", formData: await toFormData(image) });
}

export async function deletePhoto(photoId: string): Promise<void> {
  await apiRequest(endpoints.profile.photo(photoId), { method: "DELETE" });
}

export async function reorderPhotos(photoIds: string[]): Promise<Photo[]> {
  const response = await apiRequest<{ photos: Photo[] }>(endpoints.profile.photosOrder, { method: "PUT", body: { photoIds } });
  return response.photos;
}
