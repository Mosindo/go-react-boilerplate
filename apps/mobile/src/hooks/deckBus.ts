import type { SwipeResult } from "../api/types";

type SwipedListener = (userId: string, result: SwipeResult) => void;
const listeners = new Set<SwipedListener>();

/** Lets the profile detail screen act on a card that still sits in the discovery deck. */
export function onProfileSwiped(listener: SwipedListener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function emitProfileSwiped(userId: string, result: SwipeResult): void {
  listeners.forEach((listener) => listener(userId, result));
}
