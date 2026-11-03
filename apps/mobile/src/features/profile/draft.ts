import type { Gender, MyProfile, Preferences, ProfileInput } from "../../api/types";
import { validateBirthDate } from "../../lib/format";

export type Draft = {
  firstName: string;
  birthDate: string;
  gender: Gender | null;
  bio: string;
  city: string;
  interests: string[];
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export const GENDERS: { value: Gender; label: string }[] = [
  { value: "woman", label: "Femme" },
  { value: "man", label: "Homme" },
  { value: "nonbinary", label: "Non-binaire" }
];

export const MAX_INTERESTS = 10;

export function emptyDraft(): Draft {
  return {
    firstName: "",
    birthDate: "",
    gender: null,
    bio: "",
    city: "",
    interests: [],
    interestedIn: [],
    ageMin: 18,
    ageMax: 99,
    maxDistanceKm: 50
  };
}

export function draftFromProfile(profile: MyProfile | undefined): Draft {
  if (!profile) {
    return emptyDraft();
  }
  return {
    firstName: profile.firstName,
    birthDate: profile.birthDate,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city,
    interests: profile.interests.map((interest) => interest.slug),
    interestedIn: profile.preferences?.interestedIn ?? [],
    ageMin: profile.preferences?.ageMin ?? 18,
    ageMax: profile.preferences?.ageMax ?? 99,
    maxDistanceKm: profile.preferences?.maxDistanceKm ?? 50
  };
}

export type Errors = Partial<Record<keyof Draft, string>>;

export function validateBasics(draft: Draft): Errors {
  const errors: Errors = {};
  if (!draft.firstName.trim()) {
    errors.firstName = "Indiquez votre prénom.";
  } else if (draft.firstName.trim().length > 40) {
    errors.firstName = "40 caractères maximum.";
  }
  const birthError = validateBirthDate(draft.birthDate);
  if (birthError) {
    errors.birthDate = birthError;
  }
  if (!draft.gender) {
    errors.gender = "Choisissez une option.";
  }
  return errors;
}

export function validateAbout(draft: Draft): Errors {
  const errors: Errors = {};
  if (draft.bio.length > 500) {
    errors.bio = "500 caractères maximum.";
  }
  if (draft.city.length > 80) {
    errors.city = "80 caractères maximum.";
  }
  if (draft.interests.length > MAX_INTERESTS) {
    errors.interests = `${MAX_INTERESTS} centres d'intérêt maximum.`;
  }
  return errors;
}

export function validatePreferences(draft: Draft): Errors {
  const errors: Errors = {};
  if (draft.interestedIn.length === 0) {
    errors.interestedIn = "Choisissez au moins une option.";
  }
  if (draft.ageMin < 18 || draft.ageMax > 120 || draft.ageMin > draft.ageMax) {
    errors.ageMin = "Tranche d'âge invalide.";
  }
  return errors;
}

export function toProfileInput(draft: Draft): ProfileInput {
  if (!draft.gender) {
    throw new Error("gender is required");
  }
  return {
    firstName: draft.firstName.trim(),
    birthDate: draft.birthDate.trim(),
    gender: draft.gender,
    bio: draft.bio.trim(),
    city: draft.city.trim(),
    interests: draft.interests
  };
}

export function toPreferences(draft: Draft): Preferences {
  return {
    interestedIn: draft.interestedIn,
    ageMin: draft.ageMin,
    ageMax: draft.ageMax,
    maxDistanceKm: draft.maxDistanceKm
  };
}

export function hasErrors(errors: Errors): boolean {
  return Object.keys(errors).length > 0;
}

export function toggle<T>(list: T[], value: T): T[] {
  return list.includes(value) ? list.filter((item) => item !== value) : [...list, value];
}
