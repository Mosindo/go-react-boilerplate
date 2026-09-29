import type { Gender } from "../api/models";

export const LIMITS = {
  passwordMin: 8,
  passwordMax: 128,
  firstNameMax: 40,
  bioMax: 500,
  cityMax: 80,
  interestsMax: 10,
  ageMin: 18,
  ageMax: 99,
  distanceMin: 1,
  distanceMax: 500,
  reportDetailsMax: 1000
} as const;

export const GENDERS: readonly Gender[] = ["woman", "man", "non_binary"];

export function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

export function validateEmail(value: string): string | null {
  const email = normalizeEmail(value);
  if (!email) {
    return "Enter your email address.";
  }
  if (email.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email)) {
    return "Enter a valid email address.";
  }
  return null;
}

export function validatePassword(value: string): string | null {
  if (!value) {
    return "Enter a password.";
  }
  if (value.length < LIMITS.passwordMin) {
    return `Use at least ${LIMITS.passwordMin} characters.`;
  }
  if (value.length > LIMITS.passwordMax) {
    return `Use at most ${LIMITS.passwordMax} characters.`;
  }
  return null;
}

/** Login only checks presence: the server owns the real verdict and old passwords must keep working. */
export function validateLoginPassword(value: string): string | null {
  return value ? null : "Enter your password.";
}

export function validateNewPassword(newPassword: string, currentPassword?: string): string | null {
  const base = validatePassword(newPassword);
  if (base) {
    return base;
  }
  if (currentPassword !== undefined && newPassword === currentPassword) {
    return "Choose a password different from your current one.";
  }
  return null;
}

export type ProfileFormValues = {
  firstName: string;
  gender: Gender | null;
  bio: string;
  city: string;
  interests: string[];
};

export type ProfileFormErrors = Partial<Record<keyof ProfileFormValues, string>>;

export function validateProfileForm(values: ProfileFormValues): ProfileFormErrors {
  const errors: ProfileFormErrors = {};
  const firstName = values.firstName.trim();
  if (!firstName) {
    errors.firstName = "Tell us your first name.";
  } else if (firstName.length > LIMITS.firstNameMax) {
    errors.firstName = `Keep it under ${LIMITS.firstNameMax} characters.`;
  }
  if (!values.gender || !GENDERS.includes(values.gender)) {
    errors.gender = "Choose the option that fits you best.";
  }
  if (values.bio.trim().length > LIMITS.bioMax) {
    errors.bio = `Keep your bio under ${LIMITS.bioMax} characters.`;
  }
  if (values.city.trim().length > LIMITS.cityMax) {
    errors.city = `Keep it under ${LIMITS.cityMax} characters.`;
  }
  if (values.interests.length > LIMITS.interestsMax) {
    errors.interests = `Pick at most ${LIMITS.interestsMax} interests.`;
  }
  return errors;
}

export type PreferencesValues = {
  interestedIn: Gender[];
  ageMin: number;
  ageMax: number;
  maxDistanceKm: number;
};

export type PreferencesErrors = Partial<Record<keyof PreferencesValues, string>>;

export function validatePreferences(values: PreferencesValues): PreferencesErrors {
  const errors: PreferencesErrors = {};
  if (values.interestedIn.length === 0) {
    errors.interestedIn = "Pick at least one option.";
  }
  if (values.ageMin < LIMITS.ageMin || values.ageMin > LIMITS.ageMax) {
    errors.ageMin = `Minimum age must be between ${LIMITS.ageMin} and ${LIMITS.ageMax}.`;
  }
  if (values.ageMax > LIMITS.ageMax || values.ageMax < values.ageMin) {
    errors.ageMax = "Maximum age must be at least the minimum age.";
  }
  if (values.maxDistanceKm < LIMITS.distanceMin || values.maxDistanceKm > LIMITS.distanceMax) {
    errors.maxDistanceKm = `Distance must be between ${LIMITS.distanceMin} and ${LIMITS.distanceMax} km.`;
  }
  return errors;
}

export function hasErrors(errors: object): boolean {
  return Object.keys(errors).length > 0;
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Moves the minimum age while keeping the range valid (max is pushed up when needed). */
export function stepAgeMin(values: PreferencesValues, delta: number): PreferencesValues {
  const ageMin = clamp(values.ageMin + delta, LIMITS.ageMin, LIMITS.ageMax);
  return { ...values, ageMin, ageMax: Math.max(values.ageMax, ageMin) };
}

/** Moves the maximum age while keeping the range valid (min is pulled down when needed). */
export function stepAgeMax(values: PreferencesValues, delta: number): PreferencesValues {
  const ageMax = clamp(values.ageMax + delta, LIMITS.ageMin, LIMITS.ageMax);
  return { ...values, ageMax, ageMin: Math.min(values.ageMin, ageMax) };
}

/** Distance steps: 1 km below 10 km, multiples of 5 km above; always inside the allowed range. */
export function stepDistance(current: number, direction: 1 | -1): number {
  let raw: number;
  if (direction === 1) {
    raw = current < 10 ? current + 1 : Math.floor(current / 5) * 5 + 5;
  } else {
    raw = current <= 10 ? current - 1 : Math.ceil(current / 5) * 5 - 5;
  }
  return clamp(raw, LIMITS.distanceMin, LIMITS.distanceMax);
}

export function toggleInList<T>(list: readonly T[], item: T, max?: number): T[] {
  if (list.includes(item)) {
    return list.filter((entry) => entry !== item);
  }
  if (max !== undefined && list.length >= max) {
    return [...list];
  }
  return [...list, item];
}
