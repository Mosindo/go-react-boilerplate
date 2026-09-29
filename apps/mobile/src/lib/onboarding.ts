export const ONBOARDING_STEPS = ["basics", "preferences", "about", "location", "photos"] as const;
export type OnboardingStep = (typeof ONBOARDING_STEPS)[number];

export type OnboardingSnapshot = {
  hasProfile: boolean;
  hasLocation: boolean;
  photoCount: number;
};

/**
 * Index of the first step that still blocks `profileComplete`.
 * Preferences and "about" have server defaults / are optional, so they are never a resume target.
 */
export function firstMissingStep(snapshot: OnboardingSnapshot): number {
  if (!snapshot.hasProfile) {
    return 0;
  }
  if (!snapshot.hasLocation) {
    return ONBOARDING_STEPS.indexOf("location");
  }
  if (snapshot.photoCount < 1) {
    return ONBOARDING_STEPS.indexOf("photos");
  }
  return ONBOARDING_STEPS.length - 1;
}

export function progressFraction(stepIndex: number): number {
  return (
    (Math.min(Math.max(stepIndex, 0), ONBOARDING_STEPS.length - 1) + 1) / ONBOARDING_STEPS.length
  );
}
