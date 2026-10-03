import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { Interest, OwnProfile, Preferences, ProfileInput, PublicProfile } from "./types";

const json = (method: string, body: unknown) => ({ method, body: JSON.stringify(body), silent: true });

/** Resolves to null while the user has not created a profile yet. */
export async function getOwnProfile(): Promise<OwnProfile | null> {
  try {
    return await apiRequest<OwnProfile>(endpoints.profile.own, { silent: true });
  } catch (error) {
    if ((error as { status?: number }).status === 404) {
      return null;
    }
    throw error;
  }
}

export function saveProfile(input: ProfileInput): Promise<OwnProfile> {
  return apiRequest<OwnProfile>(endpoints.profile.own, json("PUT", input));
}

export async function saveLocation(latitude: number, longitude: number): Promise<void> {
  await apiRequest(endpoints.profile.location, json("PUT", { latitude, longitude }));
}

export function savePreferences(preferences: Preferences): Promise<Preferences> {
  return apiRequest<Preferences>(endpoints.profile.preferences, json("PUT", preferences));
}

export async function savePrivacy(isVisible: boolean, showDistance: boolean): Promise<void> {
  await apiRequest(endpoints.profile.privacy, json("PUT", { isVisible, showDistance }));
}

export async function listInterests(): Promise<Interest[]> {
  const response = await apiRequest<{ interests: Interest[] }>(endpoints.profile.interests, { silent: true });
  return response.interests;
}

export function getPublicProfile(userId: string): Promise<PublicProfile> {
  return apiRequest<PublicProfile>(endpoints.profile.public(userId), { silent: true });
}
