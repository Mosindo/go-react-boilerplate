import type { SwipeResult } from "../../lib/api/types";

type Listener = (userId: string, result: SwipeResult | null) => void;
const listeners = new Set<Listener>();

/** Notifies the deck when a profile was handled elsewhere (detail screen, block). */
export const swipeEvents = {
  emit(userId: string, result: SwipeResult | null) {
    listeners.forEach((listener) => listener(userId, result));
  },
  subscribe(listener: Listener) {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }
};
