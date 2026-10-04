import type { Gender, OwnProfile, Preferences, ProfileInput } from "../../api/types";
import { isoToBirthInput, parseBirthDate } from "../../lib/dates";
import { BIO_MAX, validateCity, validateDisplayName } from "../../lib/validation";
import type { Coordinates } from "../../lib/location";

export type WizardState = {
  displayName: string;
  birthInput: string;
  gender: Gender | null;
  preferences: Preferences;
  city: string;
  coordinates: Coordinates | null;
  bio: string;
  interests: string[];
};

export const DEFAULT_PREFERENCES: Preferences = { interestedIn: [], minAge: 18, maxAge: 45, maxDistanceKm: 50 };

export const STEPS = ["about", "preferences", "location", "story", "photos"] as const;
export type Step = (typeof STEPS)[number];

export function initialWizardState(profile: OwnProfile | null | undefined): WizardState {
  if (!profile) {
    return {
      displayName: "",
      birthInput: "",
      gender: null,
      preferences: DEFAULT_PREFERENCES,
      city: "",
      coordinates: null,
      bio: "",
      interests: []
    };
  }
  return {
    displayName: profile.displayName,
    birthInput: isoToBirthInput(profile.birthDate),
    gender: profile.gender,
    preferences: profile.preferences,
    city: profile.city,
    coordinates: null,
    bio: profile.bio,
    interests: profile.interests.map((i) => i.slug)
  };
}

export type StepErrors = Partial<
  Record<"displayName" | "birthInput" | "gender" | "preferences" | "city" | "bio", string>
>;

/** Validates the fields owned by a step. Empty object means the step is valid. */
export function validateStep(step: Step, state: WizardState, now: Date = new Date()): StepErrors {
  const errors: StepErrors = {};
  if (step === "about") {
    const name = validateDisplayName(state.displayName);
    if (name) errors.displayName = name;
    const birth = parseBirthDate(state.birthInput, now);
    if ("error" in birth) errors.birthInput = birth.error;
    if (!state.gender) errors.gender = "Choose the option that fits you best.";
  }
  if (step === "preferences" && state.preferences.interestedIn.length === 0) {
    errors.preferences = "Choose at least one option.";
  }
  if (step === "location") {
    const city = validateCity(state.city);
    if (city) errors.city = city;
  }
  if (step === "story" && state.bio.length > BIO_MAX) errors.bio = `Keep your bio under ${BIO_MAX} characters.`;
  return errors;
}

/** Builds the PUT /profile payload from validated wizard state. Throws when invalid. */
export function toProfileInput(
  state: WizardState,
  existing: Pick<OwnProfile, "showDistance" | "discoverable"> | null | undefined,
  now: Date = new Date()
): ProfileInput {
  const birth = parseBirthDate(state.birthInput, now);
  if ("error" in birth) throw new Error(birth.error);
  if (!state.gender) throw new Error("Gender is required");
  return {
    displayName: state.displayName.trim(),
    birthDate: birth.iso,
    gender: state.gender,
    bio: state.bio.trim(),
    city: state.city.trim(),
    interests: state.interests,
    showDistance: existing?.showDistance ?? true,
    discoverable: existing?.discoverable ?? true
  };
}
