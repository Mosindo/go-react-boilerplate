import type { SwipeAction } from "../../api/types";

type Decision = { userId: string; action: SwipeAction };
type Listener = (d: Decision) => void;

const listeners = new Set<Listener>();
let stale = false;

/** Lets the full-profile screen like/pass the card currently in the deck. */
export const deckEvents = {
  emit(d: Decision) {
    listeners.forEach((l) => l(d));
  },
  /** Preferences or location changed: rebuild the deck on next focus. */
  markStale() {
    stale = true;
  },
  consumeStale() {
    const was = stale;
    stale = false;
    return was;
  },
  subscribe(l: Listener) {
    listeners.add(l);
    return () => {
      listeners.delete(l);
    };
  }
};
