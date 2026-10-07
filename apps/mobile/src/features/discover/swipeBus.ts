import type { Match, SwipeAction } from "../types";

export type SwipeOutcome = { userId: string; action: SwipeAction; matched: boolean; match: Match | null };
type Listener = (outcome: SwipeOutcome) => void;

const listeners = new Set<Listener>();

/** Lets UserProfileScreen tell the deck which candidate was acted on. */
export function onSwipeOutcome(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function emitSwipeOutcome(outcome: SwipeOutcome): void {
  for (const listener of Array.from(listeners)) {
    listener(outcome);
  }
}
