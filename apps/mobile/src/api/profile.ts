import { api } from "./http";
import { endpoints } from "./endpoints";
import type { Interest, LocationInput, Photo, Preferences, Profile, ProfileInput } from "./types";

export function saveProfile(input: ProfileInput): Promise<Profile> {
  return api.request<Profile>(endpoints.profile, { method: "PUT", body: input });
}

export function saveLocation(input: LocationInput): Promise<Profile> {
  return api.request<Profile>(endpoints.location, { method: "PUT", body: input });
}

export function savePreferences(input: Preferences): Promise<Preferences> {
  return api.request<Preferences>(endpoints.preferences, { method: "PUT", body: input });
}

export async function getInterests(): Promise<Interest[]> {
  const response = await api.request<{ items: Interest[] }>(endpoints.interests);
  return response.items;
}

export type UploadFile = { uri: string; name: string; type: string };

function buildForm(file: UploadFile): FormData {
  const form = new FormData();
  // React Native's FormData accepts a {uri,name,type} descriptor in place of a Blob.
  form.append("file", file as unknown as Blob);
  return form;
}

export function uploadPhoto(file: UploadFile): Promise<Photo> {
  return api.request<Photo>(endpoints.photos, { method: "POST", form: buildForm(file) });
}

export async function reorderPhotos(photoIds: string[]): Promise<Photo[]> {
  const response = await api.request<{ items: Photo[] }>(endpoints.photosOrder, {
    method: "PUT",
    body: { photoIds }
  });
  return response.items;
}

export function deletePhoto(photoId: string): Promise<void> {
  return api.request(endpoints.photo(photoId), { method: "DELETE" });
}
