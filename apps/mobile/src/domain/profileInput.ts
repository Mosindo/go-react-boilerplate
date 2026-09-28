import type { Profile, ProfileInput } from "../api/types";

/** Builds a full ProfileInput from the stored profile so partial edits never drop other fields. */
export function profileToInput(profile: Profile, overrides: Partial<ProfileInput> = {}): ProfileInput {
  return {
    firstName: profile.firstName,
    birthDate: profile.birthDate,
    gender: profile.gender,
    bio: profile.bio,
    interestIds: profile.interests.map((interest) => interest.id),
    showDistance: profile.showDistance,
    isDiscoverable: profile.isDiscoverable,
    ...overrides
  };
}
