import type { OwnProfile, ProfileInput } from "../api/types";

export function profileToInput(profile: OwnProfile, patch: Partial<ProfileInput> = {}): ProfileInput {
  return {
    firstName: profile.firstName,
    birthDate: profile.birthDate,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city || undefined,
    interestIds: profile.interests.map((i) => i.id),
    showDistance: profile.showDistance,
    discoverable: profile.discoverable,
    ...patch
  };
}
