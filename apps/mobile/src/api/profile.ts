import { Platform } from "react-native";
import { request } from "./client";
import type { Interest, OwnProfile, PhotoView, Preferences, PublicProfile, UpdateProfileInput } from "./types";

async function photoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts a file descriptor object for uploads.
    form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const profileApi = {
  getMine: () => request<OwnProfile>("/me/profile"),
  update: (input: UpdateProfileInput) => request<OwnProfile>("/me/profile", { method: "PATCH", body: input }),
  updateLocation: (latitude: number, longitude: number, city: string) =>
    request<OwnProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city } }),
  updatePreferences: (input: Partial<Preferences>) =>
    request<OwnProfile>("/me/preferences", { method: "PATCH", body: input }),
  interests: () => request<{ interests: Interest[] }>("/interests").then((r) => r.interests),
  getPublic: (userId: string) => request<PublicProfile>(`/profiles/${userId}`),

  uploadPhoto: async (uri: string) =>
    request<PhotoView>("/me/photos", { method: "POST", formData: await photoForm(uri) }),
  replacePhoto: async (photoId: string, uri: string) =>
    request<PhotoView>(`/me/photos/${photoId}`, { method: "PUT", formData: await photoForm(uri) }),
  deletePhoto: (photoId: string) => request<void>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorderPhotos: (photoIds: string[]) =>
    request<{ photos: PhotoView[] }>("/me/photos/order", { method: "PUT", body: { photoIds } }).then((r) => r.photos)
};
