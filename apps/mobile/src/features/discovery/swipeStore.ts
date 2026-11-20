import { useSyncExternalStore } from "react";

// Profiles the member already answered during this session, whichever screen
// the decision came from (deck, profile detail, likes list).
let decided = new Set<string>();
const listeners = new Set<() => void>();

export const swipeStore = {
  add(userId: string) {
    decided = new Set(decided).add(userId);
    listeners.forEach((l) => l());
  },
  remove(userId: string) {
    const next = new Set(decided);
    next.delete(userId);
    decided = next;
    listeners.forEach((l) => l());
  },
  reset() {
    decided = new Set();
    listeners.forEach((l) => l());
  }
};

export function useDecidedProfiles(): Set<string> {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => decided,
    () => decided
  );
}
