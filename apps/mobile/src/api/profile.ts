import { api, isApiError } from "./client";
import type { Interest, OwnProfile, Photo, Preferences, ProfileInput, PublicProfile } from "./types";

export const MAX_PHOTOS = 6;

/** Returns null when the profile has never been created (404). */
export async function getOwnProfile(): Promise<OwnProfile | null> {
  try {
    return await api.get<OwnProfile>("/profile");
  } catch (error) {
    if (isApiError(error) && error.status === 404) return null;
    throw error;
  }
}

export function saveProfile(input: ProfileInput): Promise<OwnProfile> {
  return api.put<OwnProfile>("/profile", input);
}

export function setLocation(latitude: number, longitude: number): Promise<void> {
  return api.put<void>("/profile/location", { latitude, longitude });
}

export function clearLocation(): Promise<void> {
  return api.delete<void>("/profile/location");
}

export function getPreferences(): Promise<Preferences> {
  return api.get<Preferences>("/preferences");
}

export function savePreferences(preferences: Preferences): Promise<Preferences> {
  return api.put<Preferences>("/preferences", preferences);
}

export async function listInterests(): Promise<Interest[]> {
  const data = await api.get<{ interests: Interest[] }>("/interests");
  return data.interests;
}

export function getPublicProfile(userId: string): Promise<PublicProfile> {
  return api.get<PublicProfile>(`/profiles/${encodeURIComponent(userId)}`);
}

export type PhotoFile = { uri: string; mimeType?: string | null; fileName?: string | null };

function extensionFor(mime: string): string {
  if (mime.includes("png")) return "png";
  if (mime.includes("webp")) return "webp";
  return "jpg";
}

/** Uploads one image as multipart (field `file`). Works on native (uri parts) and web (Blob). */
export async function uploadPhoto(file: PhotoFile, platformOS: string): Promise<Photo> {
  const mime = file.mimeType && file.mimeType.startsWith("image/") ? file.mimeType : "image/jpeg";
  const name = file.fileName || `photo.${extensionFor(mime)}`;
  const form = new FormData();
  if (platformOS === "web") {
    const blob = await (await fetch(file.uri)).blob();
    form.append("file", blob, name);
  } else {
    // React Native's FormData accepts {uri,name,type} descriptors.
    form.append("file", { uri: file.uri, name, type: mime } as unknown as Blob);
  }
  return api.request<Photo>("POST", "/profile/photos", { formData: form, timeoutMs: 60_000 });
}

export function deletePhoto(id: string): Promise<void> {
  return api.delete<void>(`/profile/photos/${encodeURIComponent(id)}`);
}

export async function reorderPhotos(ids: string[]): Promise<Photo[]> {
  const data = await api.put<{ photos: Photo[] }>("/profile/photos/order", { ids });
  return data.photos;
}
