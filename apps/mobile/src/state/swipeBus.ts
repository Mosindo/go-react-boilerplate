import type { SwipeAction } from "../api/types";

type Listener = (userId: string, action: SwipeAction) => void;

const listeners = new Set<Listener>();

/** Lets a profile opened from the deck tell the deck that the person was swiped. */
export function emitSwiped(userId: string, action: SwipeAction): void {
  listeners.forEach((listener) => listener(userId, action));
}

export function onSwiped(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}
