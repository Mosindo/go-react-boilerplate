import type { Gender } from "../api/models";
import type { MyProfile, ProfileInput } from "../api/profile";
import type { ProfileFormValues } from "./validation";

export type PrivacyValues = { showDistance: boolean; showAge: boolean; discoverable: boolean };

export const DEFAULT_PRIVACY_VALUES: PrivacyValues = {
  showDistance: true,
  showAge: true,
  discoverable: true
};

export const EMPTY_PROFILE_FORM: ProfileFormValues = {
  firstName: "",
  gender: null,
  bio: "",
  city: "",
  interests: []
};

export function profileToFormValues(profile: MyProfile | null | undefined): ProfileFormValues {
  if (!profile) {
    return { ...EMPTY_PROFILE_FORM, interests: [] };
  }
  return {
    firstName: profile.firstName,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city,
    interests: [...profile.interests]
  };
}

export function profileToPrivacy(profile: MyProfile | null | undefined): PrivacyValues {
  return profile
    ? {
        showDistance: profile.showDistance,
        showAge: profile.showAge,
        discoverable: profile.discoverable
      }
    : { ...DEFAULT_PRIVACY_VALUES };
}

/** Builds the PUT /me/profile body. The caller must have validated the form (gender is required). */
export function toProfileInput(values: ProfileFormValues, privacy: PrivacyValues): ProfileInput {
  const gender: Gender | null = values.gender;
  if (!gender) {
    throw new Error("gender is required");
  }
  return {
    firstName: values.firstName.trim(),
    gender,
    bio: values.bio.trim(),
    city: values.city.trim(),
    interests: [...values.interests],
    showDistance: privacy.showDistance,
    showAge: privacy.showAge,
    discoverable: privacy.discoverable
  };
}

export function isProfileDirty(a: ProfileFormValues, b: ProfileFormValues): boolean {
  return (
    a.firstName.trim() !== b.firstName.trim() ||
    a.gender !== b.gender ||
    a.bio.trim() !== b.bio.trim() ||
    a.city.trim() !== b.city.trim() ||
    a.interests.length !== b.interests.length ||
    a.interests.some((slug) => !b.interests.includes(slug))
  );
}
