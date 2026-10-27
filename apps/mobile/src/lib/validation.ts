import { ageFromBirthDate } from "./format";

export type ProfileDraft = {
  firstName: string;
  birthDate: string;
  gender: "man" | "woman" | "non_binary" | null;
  bio: string;
  city: string;
};

export type ProfileErrors = Partial<Record<keyof ProfileDraft, string>>;

const NAME_PATTERN = /^[\p{L}][\p{L}\p{M} '-]*$/u;

export function validateProfileDraft(draft: ProfileDraft, now: Date = new Date()): ProfileErrors {
  const errors: ProfileErrors = {};
  const name = draft.firstName.trim().replace(/\s+/g, " ");
  if (!name) errors.firstName = "Tell us your first name.";
  else if (name.length > 40 || !NAME_PATTERN.test(name))
    errors.firstName = "Use letters only (max 40).";

  const age = ageFromBirthDate(draft.birthDate, now);
  if (age === null) errors.birthDate = "Use the format YYYY-MM-DD.";
  else if (age < 18) errors.birthDate = "You must be at least 18 to use Ember.";
  else if (age > 99) errors.birthDate = "Please check your birth date.";

  if (!draft.gender) errors.gender = "Choose how you identify.";
  if (draft.bio.trim().length > 500) errors.bio = "Keep your bio under 500 characters.";
  if (draft.city.trim().length > 80) errors.city = "City name is too long.";
  return errors;
}

export function validateEmail(email: string): string | null {
  const value = email.trim();
  if (!value) return "Enter your email.";
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) || value.length > 254)
    return "Enter a valid email.";
  return null;
}

export function validatePassword(password: string): string | null {
  if (password.length < 8) return "Use at least 8 characters.";
  if (password.length > 72) return "Use at most 72 characters.";
  return null;
}
