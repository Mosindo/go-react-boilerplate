import { useSyncExternalStore } from "react";

/**
 * Lets other screens (profile detail, safety actions) ask the Discover deck to act on a card.
 * "like"/"pass" run the normal swipe flow; "remove" just drops the card (blocked/reported).
 */
export type DeckIntent = { userId: string; kind: "like" | "pass" | "remove" };

let pending: DeckIntent | null = null;
const listeners = new Set<() => void>();

export function pushDeckIntent(intent: DeckIntent): void {
  pending = intent;
  listeners.forEach((l) => l());
}

export function takeDeckIntent(): DeckIntent | null {
  const intent = pending;
  pending = null;
  return intent;
}

export function usePendingDeckIntent(): DeckIntent | null {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    () => pending,
    () => pending
  );
}

/** Bumped when something that changes who can be discovered (preferences, location, unblock) is saved. */
let discoverEpoch = 0;
export function bumpDiscoverEpoch(): void {
  discoverEpoch += 1;
}
export function getDiscoverEpoch(): number {
  return discoverEpoch;
}
