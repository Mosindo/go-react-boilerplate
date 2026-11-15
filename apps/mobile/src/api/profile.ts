import { Platform } from "react-native";
import { api } from "./client";
import type { Gender, Interest, MeProfile, Photo, ProfileCard } from "./types";

export type ProfileUpdate = {
  firstName: string;
  birthdate: string;
  gender: Gender | "";
  bio: string;
  jobTitle: string;
  heightCm: number | null;
  city: string;
  interestIds: number[];
};

export type PreferencesUpdate = {
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
};

async function photoForm(uri: string): Promise<FormData> {
  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    // React Native's FormData accepts a file descriptor object.
    form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  }
  return form;
}

export const profileApi = {
  interests: () => api<{ interests: Interest[] }>("/interests", { auth: false }),
  me: () => api<MeProfile>("/me/profile"),
  update: (update: ProfileUpdate) => api<MeProfile>("/me/profile", { method: "PUT", body: update }),
  updateLocation: (latitude: number, longitude: number, city: string) =>
    api<MeProfile>("/me/location", { method: "PUT", body: { latitude, longitude, city } }),
  clearLocation: () => api<MeProfile>("/me/location", { method: "DELETE" }),
  updatePreferences: (prefs: PreferencesUpdate) => api<MeProfile>("/me/preferences", { method: "PUT", body: prefs }),
  updatePrivacy: (isPaused: boolean, hideDistance: boolean) =>
    api<MeProfile>("/me/privacy", { method: "PUT", body: { isPaused, hideDistance } }),
  view: (userId: string) => api<ProfileCard>(`/profiles/${userId}`)
};

export const photosApi = {
  upload: async (uri: string) => api<{ photos: Photo[] }>("/me/photos", { method: "POST", form: await photoForm(uri), timeoutMs: 60_000 }),
  replace: async (photoId: string, uri: string) =>
    api<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "PUT", form: await photoForm(uri), timeoutMs: 60_000 }),
  remove: (photoId: string) => api<{ photos: Photo[] }>(`/me/photos/${photoId}`, { method: "DELETE" }),
  reorder: (photoIds: string[]) => api<{ photos: Photo[] }>("/me/photos/order", { method: "PUT", body: { photoIds } })
};
