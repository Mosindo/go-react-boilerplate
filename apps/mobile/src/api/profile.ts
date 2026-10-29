import { request } from "./client";
import type {
  Interest,
  OwnProfile,
  Photo,
  Preferences,
  ProfileInput,
  PublicProfile
} from "./types";

export const fetchProfile = () => request<OwnProfile>("/profile");

export const saveProfile = (input: ProfileInput) =>
  request<OwnProfile>("/profile", { method: "PUT", body: input });

export const fetchPreferences = () => request<Preferences>("/profile/preferences");

export const savePreferences = (input: Preferences) =>
  request<Preferences>("/profile/preferences", { method: "PUT", body: input });

export const fetchInterests = async () =>
  (await request<{ interests: Interest[] }>("/interests")).interests;

export const fetchPublicProfile = (userId: string) => request<PublicProfile>(`/profiles/${userId}`);

export const fetchPhotos = async () => (await request<{ photos: Photo[] }>("/photos")).photos;

export function uploadPhoto(file: { uri: string; name: string; type: string }): Promise<Photo> {
  const form = new FormData();
  // React Native's FormData accepts { uri, name, type } descriptors for files.
  form.append("photo", file as unknown as Blob);
  return request<Photo>("/photos", { method: "POST", formData: form });
}

export const deletePhoto = (id: string) => request<void>(`/photos/${id}`, { method: "DELETE" });

export const reorderPhotos = async (photoIds: string[]) =>
  (await request<{ photos: Photo[] }>("/photos/order", { method: "PUT", body: { photoIds } }))
    .photos;
