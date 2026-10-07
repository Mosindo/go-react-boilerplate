import { apiRequest, isApiError } from "./client";
import type { Interest, ListResponse, Preferences, ProfileInput, SelfProfile, UserSummary } from "./types";

export async function listInterests(): Promise<Interest[]> {
  return (await apiRequest<ListResponse<Interest>>("/interests")).items;
}

/** Resolves to null while the profile has not been created yet (404 not_found). */
export async function getMyProfile(): Promise<SelfProfile | null> {
  try {
    return await apiRequest<SelfProfile>("/profile/me");
  } catch (error) {
    if (isApiError(error, "not_found")) {
      return null;
    }
    throw error;
  }
}

export function saveMyProfile(input: ProfileInput): Promise<SelfProfile> {
  return apiRequest<SelfProfile>("/profile/me", { method: "PUT", body: input });
}

export function updateMyLocation(latitude: number, longitude: number): Promise<void> {
  return apiRequest<void>("/profile/me/location", { method: "PUT", body: { latitude, longitude } });
}

export function getPreferences(): Promise<Preferences> {
  return apiRequest<Preferences>("/preferences");
}

export function savePreferences(preferences: Preferences): Promise<Preferences> {
  return apiRequest<Preferences>("/preferences", { method: "PUT", body: preferences });
}

export async function listBlocks(): Promise<UserSummary[]> {
  return (await apiRequest<ListResponse<UserSummary>>("/blocks")).items;
}

export function unblockUser(userId: string): Promise<void> {
  return apiRequest<void>(`/blocks/${encodeURIComponent(userId)}`, { method: "DELETE" });
}
