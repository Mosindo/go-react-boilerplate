import { api } from "./client";
import type { Interest, MyProfile, Preferences, PublicProfile, UpdateProfileInput } from "./types";

export function getMyProfile() {
  return api<MyProfile>("/profile");
}

export function updateMyProfile(input: UpdateProfileInput) {
  return api<MyProfile>("/profile", { method: "PUT", body: input });
}

export function updatePreferences(input: Preferences) {
  return api<MyProfile>("/profile/preferences", { method: "PUT", body: input });
}

export function updateLocation(latitude: number, longitude: number) {
  return api<void>("/profile/location", { method: "PUT", body: { latitude, longitude } });
}

export function clearLocation() {
  return api<void>("/profile/location", { method: "DELETE" });
}

export function updatePrivacy(input: { discoverable?: boolean; showDistance?: boolean }) {
  return api<MyProfile>("/profile/privacy", { method: "PATCH", body: input });
}

export async function listInterests() {
  const res = await api<{ interests: Interest[] }>("/interests", { anonymous: true });
  return res.interests;
}

export function getPublicProfile(userId: string) {
  return api<PublicProfile>(`/profiles/${userId}`);
}
