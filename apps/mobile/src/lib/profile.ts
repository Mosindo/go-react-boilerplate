import type { OwnProfile, ProfileInput } from "../api/types";

/** Rebuilds a save payload from the stored profile (used for partial changes like visibility). */
export function profileToInput(profile: OwnProfile, overrides: Partial<ProfileInput> = {}): ProfileInput {
  return {
    firstName: profile.firstName,
    birthDate: profile.birthDate,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city ?? "",
    interests: profile.interests.map((i) => i.id),
    isVisible: profile.isVisible,
    ...overrides
  };
}
