import { Platform } from "react-native";
import { apiRequest } from "./client";
import type { Interest, OwnProfile, Photo, Preferences, ProfileInput, PublicProfile } from "./types";

export const fetchProfile = () => apiRequest<OwnProfile>("/profile");
export const saveProfile = (input: ProfileInput) => apiRequest<OwnProfile>("/profile", { method: "PUT", body: input });
export const fetchInterests = () => apiRequest<{ items: Interest[] }>("/interests").then((r) => r.items);
export const fetchPreferences = () => apiRequest<Preferences>("/preferences");
export const savePreferences = (prefs: Preferences) => apiRequest<Preferences>("/preferences", { method: "PUT", body: prefs });
export const fetchPublicProfile = (userId: string) => apiRequest<PublicProfile>(`/users/${userId}`);

export const fetchPhotos = () => apiRequest<{ items: Photo[] }>("/photos").then((r) => r.items);
export const deletePhoto = (id: string) => apiRequest(`/photos/${id}`, { method: "DELETE" });
export const reorderPhotos = (photoIds: string[]) =>
  apiRequest<{ items: Photo[] }>("/photos/order", { method: "PUT", body: { photoIds } }).then((r) => r.items);

async function photoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts a { uri, name, type } descriptor for local files.
    form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export async function uploadPhoto(uri: string): Promise<Photo> {
  return apiRequest<Photo>("/photos", { method: "POST", form: await photoForm(uri) });
}

export async function replacePhoto(id: string, uri: string): Promise<Photo> {
  return apiRequest<Photo>(`/photos/${id}`, { method: "PUT", form: await photoForm(uri) });
}
