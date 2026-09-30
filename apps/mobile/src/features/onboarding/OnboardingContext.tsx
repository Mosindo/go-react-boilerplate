import { createContext, useContext } from "react";

/** Lets the onboarding flow tell the navigator it is finished. */
export const OnboardingContext = createContext<{ finish: () => void }>({ finish: () => undefined });

export function useOnboarding() {
  return useContext(OnboardingContext);
}
