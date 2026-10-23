import { useSyncExternalStore } from "react";
import type { Card, MatchSummary } from "../api/types";

export type MatchCelebration = { match: MatchSummary; card: Card | null };

type State = { swiped: ReadonlySet<string>; celebration: MatchCelebration | null };

let state: State = { swiped: new Set(), celebration: null };
const listeners = new Set<() => void>();

function set(next: State) {
  state = next;
  listeners.forEach((listener) => listener());
}

/** Shared between the deck, the profile detail screen and the match modal. */
export const discoverStore = {
  get: () => state,
  subscribe: (listener: () => void) => {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
  markSwiped: (userId: string) => set({ ...state, swiped: new Set(state.swiped).add(userId) }),
  unmarkSwiped: (userId: string) => {
    const swiped = new Set(state.swiped);
    swiped.delete(userId);
    set({ ...state, swiped });
  },
  celebrate: (celebration: MatchCelebration | null) => set({ ...state, celebration }),
  reset: () => set({ swiped: new Set(), celebration: null })
};

export function useDiscoverState(): State {
  return useSyncExternalStore(discoverStore.subscribe, discoverStore.get, discoverStore.get);
}
