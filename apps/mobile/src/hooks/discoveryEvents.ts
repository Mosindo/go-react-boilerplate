import type { Match } from "../api/types";

type DecisionEvent = { userId: string; match?: Match };
type Listener = (event: DecisionEvent) => void;

const listeners = new Set<Listener>();

/** Lets the profile detail screen tell the deck a decision was made there. */
export const discoveryEvents = {
  emit(event: DecisionEvent) {
    listeners.forEach((l) => l(event));
  },
  subscribe(listener: Listener) {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }
};
