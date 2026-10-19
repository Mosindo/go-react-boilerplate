import { request } from "./client";
import type { Interest, OwnProfile, PhotoRef, Preferences, ProfileInput, PublicProfile } from "./types";

export const getOwnProfile = () => request<OwnProfile>("/profile");

export const saveProfile = (input: ProfileInput) =>
  request<OwnProfile>("/profile", { method: "PUT", body: input });

export const savePreferences = (prefs: Preferences) =>
  request<Preferences>("/preferences", { method: "PUT", body: prefs });

export const listInterests = async () => (await request<{ interests: Interest[] }>("/interests")).interests;

export const getProfile = (userId: string) => request<PublicProfile>(`/profiles/${userId}`);

export async function uploadPhoto(uri: string, position?: number): Promise<PhotoRef[]> {
  const form = new FormData();
  // React Native's FormData accepts a { uri, name, type } descriptor for files.
  form.append("photo", { uri, name: "photo.jpg", type: "image/jpeg" } as unknown as Blob);
  if (position !== undefined) form.append("position", String(position));
  return (await request<{ photos: PhotoRef[] }>("/profile/photos", { method: "POST", form })).photos;
}

export const deletePhoto = async (id: string) =>
  (await request<{ photos: PhotoRef[] }>(`/profile/photos/${id}`, { method: "DELETE" })).photos;

export const reorderPhotos = async (photoIds: string[]) =>
  (await request<{ photos: PhotoRef[] }>("/profile/photos/order", { method: "PUT", body: { photoIds } }))
    .photos;
