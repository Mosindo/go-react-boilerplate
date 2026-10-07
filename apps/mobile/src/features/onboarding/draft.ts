import type { Gender, Preferences, ProfileInput, SelfProfile } from "../../api/types";
import { strings } from "../../i18n/strings";
import type { BirthDateParts } from "../../shared/form";
import {
  buildBirthDate,
  firstError,
  splitBirthDate,
  validateAgeRange,
  validateBio,
  validateBirthDate,
  validateCity,
  validateDistance,
  validateFirstName,
  type ValidationResult
} from "../../utils/validation";

export type OnboardingStep =
  "name" | "birthDate" | "gender" | "preferences" | "location" | "about" | "photos";

export const ONBOARDING_STEPS: OnboardingStep[] = [
  "name",
  "birthDate",
  "gender",
  "preferences",
  "location",
  "about",
  "photos"
];

export type Draft = {
  firstName: string;
  birth: BirthDateParts;
  gender: Gender | null;
  interestedIn: Gender[];
  minAge: number;
  maxAge: number;
  maxDistanceKm: number;
  coords: { latitude: number; longitude: number } | null;
  city: string;
  bio: string;
  interests: string[];
};

export const initialDraft: Draft = {
  firstName: "",
  birth: { day: "", month: "", year: "" },
  gender: null,
  interestedIn: [],
  minAge: 18,
  maxAge: 60,
  maxDistanceKm: 50,
  coords: null,
  city: "",
  bio: "",
  interests: []
};

export function draftFromProfile(profile: SelfProfile): Draft {
  return {
    ...initialDraft,
    firstName: profile.firstName,
    birth: splitBirthDate(profile.birthDate),
    gender: profile.gender,
    city: profile.city,
    bio: profile.bio,
    interests: profile.interests.map((interest) => interest.slug)
  };
}

/** Error message for the given step, or null when the step can be left. Pure: easy to unit test. */
export function validateStep(step: OnboardingStep, draft: Draft, now: Date = new Date()): ValidationResult {
  switch (step) {
    case "name":
      return validateFirstName(draft.firstName);
    case "birthDate":
      return validateBirthDate(buildBirthDate(draft.birth.day, draft.birth.month, draft.birth.year), now);
    case "gender":
      return draft.gender ? null : strings.validation.genderRequired;
    case "preferences":
      return firstError(
        draft.interestedIn.length === 0 ? strings.validation.interestedInRequired : null,
        validateAgeRange(draft.minAge, draft.maxAge),
        validateDistance(draft.maxDistanceKm)
      );
    case "location":
      return !draft.coords && draft.city.trim().length === 0
        ? strings.validation.locationRequired
        : validateCity(draft.city);
    case "about":
      return validateBio(draft.bio);
    default:
      return null;
  }
}

export function buildProfileInput(draft: Draft): ProfileInput | null {
  const birthDate = buildBirthDate(draft.birth.day, draft.birth.month, draft.birth.year);
  if (!birthDate || !draft.gender) {
    return null;
  }
  return {
    firstName: draft.firstName.trim(),
    birthDate,
    gender: draft.gender,
    bio: draft.bio.trim(),
    city: draft.city.trim(),
    interests: draft.interests
  };
}

export function buildPreferences(draft: Draft): Preferences {
  return {
    interestedIn: draft.interestedIn,
    minAge: draft.minAge,
    maxAge: draft.maxAge,
    maxDistanceKm: draft.maxDistanceKm,
    discoverable: true,
    showDistance: true
  };
}
