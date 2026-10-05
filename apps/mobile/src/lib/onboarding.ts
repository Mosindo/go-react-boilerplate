import type { Completion } from "../api/types";

/** Onboarding steps: 0 about you, 1 photos, 2 bio & interests, 3 preferences, 4 location. */
export const ONBOARDING_STEPS = ["Vous", "Photos", "À propos", "Préférences", "Localisation"] as const;

/** First step that still needs work, so people who quit midway resume where they stopped. */
export function firstIncompleteStep(completion: Completion): number {
  if (!completion.profile) return 0;
  if (!completion.photos) return 1;
  if (!completion.preferences) return 3;
  if (!completion.location) return 4;
  return 2;
}
