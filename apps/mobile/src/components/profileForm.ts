import type { Gender, LookingFor, OwnProfile, Preferences, ProfileInput } from "../api/types";
import { isoToFrench, parseBirthdate } from "../utils/validation";

export type ProfileFormState = {
  firstName: string;
  birthdate: string;
  gender: Gender | null;
  bio: string;
  jobTitle: string;
  city: string;
  heightCm: string;
  lookingFor: LookingFor | null;
  interestIds: number[];
};

export type ProfileFormErrors = Partial<Record<keyof ProfileFormState, string>>;

export function emptyProfileForm(): ProfileFormState {
  return { firstName: "", birthdate: "", gender: null, bio: "", jobTitle: "", city: "", heightCm: "", lookingFor: null, interestIds: [] };
}

export function profileToForm(p: OwnProfile | null | undefined): ProfileFormState {
  if (!p) {
    return emptyProfileForm();
  }
  return {
    firstName: p.firstName,
    birthdate: isoToFrench(p.birthdate),
    gender: p.gender,
    bio: p.bio,
    jobTitle: p.jobTitle ?? "",
    city: p.city ?? "",
    heightCm: p.heightCm ? String(p.heightCm) : "",
    lookingFor: p.lookingFor,
    interestIds: p.interests.map((i) => i.id)
  };
}

/** Validates the form client-side (the server validates again). */
export function formToInput(form: ProfileFormState): { input?: ProfileInput; errors: ProfileFormErrors } {
  const errors: ProfileFormErrors = {};
  const firstName = form.firstName.trim().replace(/\s+/g, " ");
  if (!firstName || firstName.length > 40 || !/^[\p{L}\p{M}' ’-]+$/u.test(firstName)) {
    errors.firstName = "Votre prénom (lettres uniquement, 40 max).";
  }
  const birth = parseBirthdate(form.birthdate);
  if (typeof birth === "string") {
    errors.birthdate = birth;
  }
  if (!form.gender) {
    errors.gender = "Choisissez une option.";
  }
  if (form.bio.length > 500) {
    errors.bio = "500 caractères maximum.";
  }
  let heightCm: number | null = null;
  if (form.heightCm.trim()) {
    heightCm = Number(form.heightCm);
    if (!Number.isInteger(heightCm) || heightCm < 120 || heightCm > 230) {
      errors.heightCm = "Entre 120 et 230 cm.";
    }
  }
  if (Object.keys(errors).length > 0 || typeof birth === "string" || !form.gender) {
    return { errors };
  }
  return {
    errors,
    input: {
      firstName,
      birthdate: birth.iso,
      gender: form.gender,
      bio: form.bio.trim(),
      jobTitle: form.jobTitle.trim() || null,
      city: form.city.trim() || null,
      heightCm,
      lookingFor: form.lookingFor,
      interestIds: form.interestIds
    }
  };
}

export const defaultPreferences: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 30 };

export function validatePreferences(p: Preferences): string | null {
  if (p.interestedIn.length === 0) {
    return "Choisissez au moins une option.";
  }
  if (p.minAge < 18 || p.maxAge > 99 || p.minAge > p.maxAge) {
    return "Tranche d'âge invalide.";
  }
  return null;
}
