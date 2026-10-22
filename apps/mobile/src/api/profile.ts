import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import { photoFormData, type LocalImage } from "./upload";
import type { Interest, PhotoRef, Preferences, ProfileInput, PublicProfile, SelfProfile } from "./types";

export function fetchProfile(): Promise<SelfProfile> {
  return apiRequest<SelfProfile>(endpoints.profile);
}

export function saveProfile(input: ProfileInput): Promise<SelfProfile> {
  return apiRequest<SelfProfile>(endpoints.profile, { method: "PUT", body: input, quiet: true });
}

export function fetchPreferences(): Promise<Preferences> {
  return apiRequest<Preferences>(endpoints.preferences);
}

export function savePreferences(input: Preferences): Promise<Preferences> {
  return apiRequest<Preferences>(endpoints.preferences, { method: "PUT", body: input, quiet: true });
}

export async function fetchInterests(): Promise<Interest[]> {
  return (await apiRequest<{ interests: Interest[] }>(endpoints.interests)).interests;
}

export function fetchPublicProfile(userId: string): Promise<PublicProfile> {
  return apiRequest<PublicProfile>(endpoints.publicProfile(userId));
}

export async function uploadPhoto(image: LocalImage): Promise<PhotoRef> {
  return apiRequest<PhotoRef>(endpoints.photos, { method: "POST", form: await photoFormData(image), quiet: true });
}

export async function replacePhoto(id: string, image: LocalImage): Promise<PhotoRef> {
  return apiRequest<PhotoRef>(endpoints.photo(id), { method: "PUT", form: await photoFormData(image), quiet: true });
}

export function deletePhoto(id: string): Promise<void> {
  return apiRequest(endpoints.photo(id), { method: "DELETE", quiet: true });
}

export async function reorderPhotos(photoIds: string[]): Promise<PhotoRef[]> {
  return (await apiRequest<{ photos: PhotoRef[] }>(endpoints.photoOrder, { method: "PUT", body: { photoIds }, quiet: true })).photos;
}

export async function setPrimaryPhoto(id: string): Promise<PhotoRef[]> {
  return (await apiRequest<{ photos: PhotoRef[] }>(endpoints.photoPrimary(id), { method: "PUT", quiet: true })).photos;
}

export async function fetchPhotos(): Promise<PhotoRef[]> {
  return (await apiRequest<{ photos: PhotoRef[] }>(endpoints.photos)).photos;
}
