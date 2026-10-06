import { useSyncExternalStore } from "react";

// Onboarding becomes "complete" on the server as soon as the last required
// step (first photo) is saved, but the user should still be able to add more
// photos and press "Terminer". This flag keeps the flow on screen until then.

let active = false;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((listener) => listener());
}

export function startOnboardingFlow(): void {
  if (!active) {
    active = true;
    emit();
  }
}

export function finishOnboardingFlow(): void {
  if (active) {
    active = false;
    emit();
  }
}

export function useOnboardingFlowActive(): boolean {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    () => active,
    () => active
  );
}
