import type { Gender, OwnProfile, Preferences, ProfileInput } from '../../api';
import { fr } from '../../i18n';
import { isValidFirstName, parseBirthDate, splitIsoDate } from '../../lib/validation';

export const ONBOARDING_STEPS = [
  'identity',
  'preferences',
  'location',
  'bio',
  'interests',
  'photos',
] as const;
export type OnboardingStep = (typeof ONBOARDING_STEPS)[number];

export interface OnboardingForm {
  firstName: string;
  birth: { day: string; month: string; year: string };
  gender: Gender | null;
  preferences: Preferences;
  city: string;
  coordinates: { latitude: number; longitude: number } | null;
  hasLocation: boolean;
  bio: string;
  interestIds: number[];
}

export const DEFAULT_PREFERENCES: Preferences = {
  interestedIn: [],
  minAge: 18,
  maxAge: 40,
  maxDistanceKm: 50,
};

export function emptyForm(): OnboardingForm {
  return {
    firstName: '',
    birth: { day: '', month: '', year: '' },
    gender: null,
    preferences: DEFAULT_PREFERENCES,
    city: '',
    coordinates: null,
    hasLocation: false,
    bio: '',
    interestIds: [],
  };
}

export function formFromProfile(profile: OwnProfile): OnboardingForm {
  const [day, month, year] = splitIsoDate(profile.birthDate);
  return {
    firstName: profile.firstName,
    birth: { day, month, year },
    gender: profile.gender,
    preferences: profile.preferences,
    city: profile.city,
    coordinates: null,
    hasLocation: profile.hasLocation,
    bio: profile.bio,
    interestIds: profile.interests.map((i) => i.id),
  };
}

/**
 * Resume point. No profile yet (GET /me/profile 404) -> start over; a saved profile means
 * everything but photos is done, so go straight to the photos step.
 */
export function deriveInitialStep(profile: OwnProfile | null): OnboardingStep {
  return profile ? 'photos' : 'identity';
}

export type FieldErrors = Partial<
  Record<'firstName' | 'birth' | 'gender' | 'interestedIn' | 'city', string>
>;

export function validateStep(
  step: OnboardingStep,
  form: OnboardingForm,
  now: Date = new Date(),
): FieldErrors {
  const errors: FieldErrors = {};
  if (step === 'identity') {
    if (!isValidFirstName(form.firstName)) errors.firstName = fr.validation.firstName;
    const birth = parseBirthDate(form.birth.day, form.birth.month, form.birth.year, now);
    if (!birth.ok) errors.birth = fr.validation.birth[birth.error];
    if (!form.gender) errors.gender = fr.validation.gender;
  }
  if (step === 'preferences' && form.preferences.interestedIn.length === 0) {
    errors.interestedIn = fr.validation.interestedIn;
  }
  if (step === 'location' && form.city.trim().length === 0) {
    errors.city = fr.validation.city;
  }
  return errors;
}

/** Builds the PUT /me/profile body; null when the form is not valid yet. */
export function toProfileInput(form: OnboardingForm, now: Date = new Date()): ProfileInput | null {
  const birth = parseBirthDate(form.birth.day, form.birth.month, form.birth.year, now);
  if (!birth.ok || !form.gender || !isValidFirstName(form.firstName)) return null;
  return {
    firstName: form.firstName.trim(),
    birthDate: birth.iso,
    gender: form.gender,
    bio: form.bio.trim(),
    city: form.city.trim(),
    interestIds: form.interestIds,
  };
}
