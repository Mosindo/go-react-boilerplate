import { createContext, useContext } from "react";

export const OnboardingGateContext = createContext<{ finish: () => void }>({ finish: () => undefined });

export function useOnboardingGate() {
  return useContext(OnboardingGateContext);
}
