import { apiRequest } from "./client";
import type { Interest, OwnProfile, Photo, Preferences, ProfileInput, PublicProfile } from "./types";

export const fetchInterests = () => apiRequest<{ interests: Interest[] }>("/interests").then((r) => r.interests);

export const fetchProfile = () => apiRequest<OwnProfile>("/profile");
export const saveProfile = (input: ProfileInput) => apiRequest<OwnProfile>("/profile", { method: "PUT", body: input });

export const saveLocation = (latitude: number, longitude: number, city?: string) =>
  apiRequest<OwnProfile>("/location", { method: "PUT", body: { latitude, longitude, city } });

export const fetchPreferences = () => apiRequest<Preferences>("/preferences");
export const savePreferences = (prefs: Preferences) =>
  apiRequest<Preferences>("/preferences", { method: "PUT", body: prefs });

export const fetchPublicProfile = (userId: string) => apiRequest<PublicProfile>(`/profiles/${userId}`);

export type UploadFile = { uri: string; name: string; type: string };

export function uploadPhoto(file: UploadFile): Promise<Photo> {
  const form = new FormData();
  // React Native accepts { uri, name, type } objects as file parts.
  form.append("photo", file as unknown as Blob);
  return apiRequest<Photo>("/photos", { method: "POST", form });
}

export const reorderPhotos = (photoIds: string[]) =>
  apiRequest<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { photoIds } }).then((r) => r.photos);

export const deletePhoto = (id: string) => apiRequest(`/photos/${id}`, { method: "DELETE" });
