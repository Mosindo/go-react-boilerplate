import type { MyProfile, ProfileInput } from "../../api";

/** PUT /profile replaces the whole profile: start from the current one and apply a patch. */
export function profileToInput(profile: MyProfile, patch: Partial<ProfileInput> = {}): ProfileInput {
  return {
    firstName: profile.firstName,
    birthDate: profile.birthDate,
    gender: profile.gender,
    bio: profile.bio,
    interests: profile.interests.map((interest) => interest.slug),
    discoverable: profile.discoverable,
    showDistance: profile.showDistance,
    ...patch
  };
}
