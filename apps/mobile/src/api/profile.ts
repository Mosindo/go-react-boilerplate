import { isStatus } from "../lib/errors";
import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { Gender, PublicProfile } from "./models";

export type Interest = { slug: string; label: string };

/** The caller's own profile. `age` is always set for self, `distanceKm` is always null. */
export type MyProfile = Omit<PublicProfile, "age" | "distanceKm"> & {
  age: number;
  distanceKm: null;
  hasLocation: boolean;
  showDistance: boolean;
  showAge: boolean;
  discoverable: boolean;
  isComplete: boolean;
};

export type ProfileInput = {
  firstName: string;
  gender: Gender;
  bio: string;
  city: string;
  interests: string[];
  showDistance: boolean;
  showAge: boolean;
  discoverable: boolean;
};

export type Preferences = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export const DEFAULT_PRIVACY = { showDistance: true, showAge: true, discoverable: true } as const;

export const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: ["woman", "man", "non_binary"],
  ageMin: 18,
  ageMax: 99,
  maxDistanceKm: 50
};

export async function listInterests(): Promise<Interest[]> {
  const payload = await apiRequest<{ interests: Interest[] }>(endpoints.profile.interests);
  return payload.interests;
}

/** Resolves to null while onboarding has not started (404). */
export async function getMyProfile(): Promise<MyProfile | null> {
  try {
    return await apiRequest<MyProfile>(endpoints.profile.mine);
  } catch (error) {
    if (isStatus(error, 404)) {
      return null;
    }
    throw error;
  }
}

export async function saveMyProfile(input: ProfileInput): Promise<MyProfile> {
  return apiRequest<MyProfile>(endpoints.profile.mine, {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export async function saveLocation(latitude: number, longitude: number): Promise<void> {
  await apiRequest<void>(endpoints.profile.location, {
    method: "PUT",
    body: JSON.stringify({ latitude, longitude })
  });
}

/** Resolves to null before the profile exists (404). */
export async function getPreferences(): Promise<Preferences | null> {
  try {
    return await apiRequest<Preferences>(endpoints.profile.preferences);
  } catch (error) {
    if (isStatus(error, 404)) {
      return null;
    }
    throw error;
  }
}

export async function savePreferences(input: Preferences): Promise<Preferences> {
  return apiRequest<Preferences>(endpoints.profile.preferences, {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export async function getPublicProfile(userId: string): Promise<PublicProfile> {
  return apiRequest<PublicProfile>(endpoints.profile.detail(userId));
}
