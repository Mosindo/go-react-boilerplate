import { Platform } from "react-native";
import { apiRequest } from "./client";
import type { Interest, MeProfile, Photo, Preferences, ProfileCard, ProfileInput } from "./types";

export type UploadablePhoto = { uri: string; mimeType?: string };

async function photoForm(photo: UploadablePhoto): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(photo.uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts a { uri, name, type } descriptor.
    form.append("photo", { uri: photo.uri, name: "photo.jpg", type: photo.mimeType ?? "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const profileApi = {
  interests: () => apiRequest<{ interests: Interest[] }>("/interests", { auth: false }),
  me: () => apiRequest<MeProfile>("/me/profile"),
  save: (input: ProfileInput) => apiRequest<MeProfile>("/me/profile", { method: "PUT", body: input }),
  savePreferences: (prefs: Preferences) => apiRequest<MeProfile>("/me/preferences", { method: "PUT", body: prefs }),
  updateLocation: (latitude: number, longitude: number) => apiRequest<void>("/me/location", { method: "PUT", body: { latitude, longitude } }),
  setVisibility: (isDiscoverable: boolean) => apiRequest<void>("/me/visibility", { method: "PUT", body: { isDiscoverable } }),
  view: (userId: string) => apiRequest<ProfileCard>(`/profiles/${userId}`),

  uploadPhoto: async (photo: UploadablePhoto) => apiRequest<Photo>("/me/photos", { method: "POST", form: await photoForm(photo), timeoutMs: 60_000 }),
  replacePhoto: async (photoId: string, photo: UploadablePhoto) =>
    apiRequest<Photo>(`/me/photos/${photoId}`, { method: "PUT", form: await photoForm(photo), timeoutMs: 60_000 }),
  deletePhoto: (photoId: string) => apiRequest<void>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorderPhotos: (photoIds: string[]) => apiRequest<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })
};
