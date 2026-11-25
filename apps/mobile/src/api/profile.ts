import { Platform } from "react-native";
import { api } from "./client";
import type { Interest, MeProfile, Photo, Preferences, Privacy, ProfileInput, PublicProfile } from "./types";

export type LocalImage = {
  uri: string;
  mimeType: string;
};

async function photoForm(image: LocalImage): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(image.uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts file descriptors with a uri.
    form.append("photo", { uri: image.uri, name: "photo.jpg", type: image.mimeType } as unknown as Blob);
  }
  return form;
}

export const profileApi = {
  me: () => api<MeProfile>("/me/profile"),
  update: (input: ProfileInput) => api<MeProfile>("/me/profile", { method: "PUT", body: input }),
  updatePreferences: (prefs: Preferences) => api<MeProfile>("/me/preferences", { method: "PUT", body: prefs }),
  updatePrivacy: (privacy: Privacy) => api<MeProfile>("/me/privacy", { method: "PUT", body: privacy }),
  updateLocation: (latitude: number, longitude: number) => api<MeProfile>("/me/location", { method: "PUT", body: { latitude, longitude } }),
  interests: async () => (await api<{ interests: Interest[] }>("/interests")).interests,
  publicProfile: (userId: string) => api<PublicProfile>(`/profiles/${userId}`),

  uploadPhoto: async (image: LocalImage) => api<Photo>("/me/photos", { method: "POST", form: await photoForm(image), timeoutMs: 60_000 }),
  replacePhoto: async (photoId: string, image: LocalImage) =>
    api<Photo>(`/me/photos/${photoId}`, { method: "PUT", form: await photoForm(image), timeoutMs: 60_000 }),
  deletePhoto: (photoId: string) => api<void>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorderPhotos: async (photoIds: string[]) =>
    (await api<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })).photos
};
