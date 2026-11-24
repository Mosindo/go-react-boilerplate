import { useSyncExternalStore } from "react";
import type { Match } from "../../api/types";

let current: Match | null = null;
const listeners = new Set<() => void>();

export function celebrateMatch(match: Match) {
  current = match;
  listeners.forEach((l) => l());
}

export function dismissMatch() {
  current = null;
  listeners.forEach((l) => l());
}

export function useCelebratedMatch(): Match | null {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => current
  );
}
