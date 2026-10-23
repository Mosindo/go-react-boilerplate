import type { Gender, Profile, ProfileInput } from "../api/types";
import { parseBirthDate, splitBirthDate } from "./dates";
import {
  validateBio,
  validateBirthDate,
  validateCity,
  validateFirstName,
  type FieldError
} from "./validation";

export type ProfileDraft = {
  firstName: string;
  day: string;
  month: string;
  year: string;
  gender: Gender | null;
  city: string;
  location: { latitude: number; longitude: number } | null;
  bio: string;
  interests: string[];
};

export const emptyDraft: ProfileDraft = {
  firstName: "",
  day: "",
  month: "",
  year: "",
  gender: null,
  city: "",
  location: null,
  bio: "",
  interests: []
};

export function draftFromProfile(profile: Profile): ProfileDraft {
  return {
    firstName: profile.firstName,
    ...splitBirthDate(profile.birthDate),
    gender: profile.gender,
    city: profile.city,
    location: null,
    bio: profile.bio,
    interests: profile.interests.map((interest) => interest.slug)
  };
}

export type DraftErrors = Partial<
  Record<"firstName" | "birth" | "gender" | "city" | "bio", FieldError>
>;

export function validateDraft(draft: ProfileDraft, now: Date = new Date()): DraftErrors {
  return {
    firstName: validateFirstName(draft.firstName),
    birth: validateBirthDate(draft.day, draft.month, draft.year, now),
    gender: draft.gender ? null : "required",
    city: validateCity(draft.city),
    bio: validateBio(draft.bio)
  };
}

export function hasErrors(errors: DraftErrors): boolean {
  return Object.values(errors).some(Boolean);
}

/** Builds the PUT /me/profile body, or null when the draft is invalid. */
export function draftToInput(draft: ProfileDraft, now: Date = new Date()): ProfileInput | null {
  if (hasErrors(validateDraft(draft, now)) || !draft.gender) return null;
  const birthDate = parseBirthDate(draft.day, draft.month, draft.year, now);
  if (!birthDate) return null;
  return {
    firstName: draft.firstName.trim(),
    birthDate,
    gender: draft.gender,
    bio: draft.bio.trim(),
    city: draft.city.trim(),
    latitude: draft.location?.latitude ?? null,
    longitude: draft.location?.longitude ?? null,
    interests: draft.interests.slice(0, 10)
  };
}
