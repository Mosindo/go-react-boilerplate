import type { OwnProfile, ProfileInput } from "../api/types";
import type { Gender, LookingFor } from "./labels";

export const MAX_INTERESTS = 10;
export const MAX_BIO = 500;

/** Editable form state: strings for text inputs, nullable gender until chosen. */
export type ProfileDraft = {
  firstName: string;
  gender: Gender | null;
  bio: string;
  city: string;
  latitude?: number;
  longitude?: number;
  clearLocation: boolean;
  occupation: string;
  height: string;
  lookingFor: LookingFor;
  interests: string[];
};

export const emptyDraft: ProfileDraft = {
  firstName: "",
  gender: null,
  bio: "",
  city: "",
  clearLocation: false,
  occupation: "",
  height: "",
  lookingFor: "unsure",
  interests: []
};

export function draftFromProfile(p: OwnProfile): ProfileDraft {
  return {
    firstName: p.firstName,
    gender: p.gender,
    bio: p.bio,
    city: p.city,
    latitude: p.latitude ?? undefined,
    longitude: p.longitude ?? undefined,
    clearLocation: false,
    occupation: p.occupation,
    height: p.heightCm ? String(p.heightCm) : "",
    lookingFor: p.lookingFor,
    interests: p.interests
  };
}

export type DraftErrors = Partial<Record<"firstName" | "gender" | "bio" | "height" | "city" | "occupation" | "interests", string>>;

export function validateDraft(d: ProfileDraft): DraftErrors {
  const errors: DraftErrors = {};
  const name = d.firstName.trim();
  if (!name) errors.firstName = "Indiquez votre prénom.";
  else if ([...name].length > 40) errors.firstName = "40 caractères maximum.";
  if (!d.gender) errors.gender = "Choisissez une option.";
  if ([...d.bio.trim()].length > MAX_BIO) errors.bio = `${MAX_BIO} caractères maximum.`;
  if ([...d.city.trim()].length > 80) errors.city = "80 caractères maximum.";
  if ([...d.occupation.trim()].length > 80) errors.occupation = "80 caractères maximum.";
  if (d.height.trim()) {
    const h = Number(d.height);
    if (!Number.isInteger(h) || h < 100 || h > 250) errors.height = "Taille entre 100 et 250 cm.";
  }
  if (d.interests.length > MAX_INTERESTS) errors.interests = `${MAX_INTERESTS} centres d'intérêt maximum.`;
  return errors;
}

export function draftToInput(d: ProfileDraft): ProfileInput {
  if (!d.gender) throw new Error("gender is required");
  return {
    firstName: d.firstName.trim(),
    gender: d.gender,
    bio: d.bio.trim(),
    city: d.city.trim(),
    latitude: d.clearLocation ? undefined : d.latitude,
    longitude: d.clearLocation ? undefined : d.longitude,
    clearLocation: d.clearLocation || undefined,
    occupation: d.occupation.trim(),
    heightCm: d.height.trim() ? Number(d.height) : null,
    lookingFor: d.lookingFor,
    interests: d.interests
  };
}

export function toggleInterest(list: string[], slug: string): string[] {
  if (list.includes(slug)) return list.filter((s) => s !== slug);
  return list.length >= MAX_INTERESTS ? list : [...list, slug];
}
