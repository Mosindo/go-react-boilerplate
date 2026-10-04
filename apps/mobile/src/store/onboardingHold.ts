import { useSyncExternalStore } from "react";

/**
 * While the onboarding wizard is mounted the app keeps showing it, even if the server already
 * reports the profile as completed (first photo uploaded) — the user may still be adding photos.
 */
let held = false;
const listeners = new Set<() => void>();

export function setOnboardingHold(value: boolean): void {
  if (held === value) return;
  held = value;
  listeners.forEach((l) => l());
}

export function useOnboardingHold(): boolean {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    () => held,
    () => held
  );
}
