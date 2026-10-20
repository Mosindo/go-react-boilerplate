import { apiRequest } from "./client";
import type { Gender, Interest, Photo, Preferences, Profile } from "./types";

export type ProfileInput = {
  firstName: string;
  birthDate: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
};

export type MyProfile = { profile: Profile | null; preferences: Preferences; complete: boolean };

export const fetchInterests = () => apiRequest<{ interests: Interest[] }>("/interests");
export const fetchMyProfile = () => apiRequest<MyProfile>("/profile/me");
export const saveProfile = (input: ProfileInput) =>
  apiRequest<Profile>("/profile/me", { method: "PUT", body: input });
export const savePreferences = (input: Preferences) =>
  apiRequest<Preferences>("/profile/preferences", { method: "PUT", body: input });
export const saveLocation = (latitude: number, longitude: number) =>
  apiRequest("/profile/location", { method: "PUT", body: { latitude, longitude } });
export const fetchProfile = (userId: string) => apiRequest<Profile>(`/profiles/${userId}`);

export function uploadPhoto(uri: string, mimeType: string | undefined): Promise<Photo> {
  const form = new FormData();
  const type = mimeType === "image/png" ? "image/png" : "image/jpeg";
  const name = type === "image/png" ? "photo.png" : "photo.jpg";
  // React Native's FormData accepts a { uri, name, type } descriptor in place of a Blob.
  form.append("file", { uri, name, type } as unknown as Blob);
  return apiRequest<Photo>("/photos", { method: "POST", form });
}
export const reorderPhotos = (photoIds: string[]) =>
  apiRequest<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { photoIds } });
export const deletePhoto = (id: string) => apiRequest(`/photos/${id}`, { method: "DELETE" });
