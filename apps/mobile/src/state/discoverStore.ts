import { useSyncExternalStore } from "react";
import { ApiError } from "../api/client";
import { matchingApi } from "../api/endpoints";
import type { PublicProfile, SwipeAction, SwipeResult } from "../api/types";
import { mergeBatch } from "../lib/queue";

export type DiscoverState = {
  queue: PublicProfile[];
  loading: boolean;
  error: string | null;
  exhausted: boolean;
  incomplete: boolean;
  match: SwipeResult["match"] | null;
};

const initial: DiscoverState = {
  queue: [],
  loading: false,
  error: null,
  exhausted: false,
  incomplete: false,
  match: null
};
const REFILL_THRESHOLD = 3;

let state: DiscoverState = initial;
let seen = new Set<string>();
let generation = 0;
const listeners = new Set<() => void>();

function set(patch: Partial<DiscoverState>): void {
  state = { ...state, ...patch };
  listeners.forEach((listener) => listener());
}

async function load(): Promise<void> {
  if (state.loading) return;
  const current = generation;
  set({ loading: true, error: null });
  try {
    const batch = await matchingApi.discover(10);
    if (current !== generation) return;
    const queue = mergeBatch(state.queue, batch, seen);
    set({ queue, loading: false, exhausted: queue.length === state.queue.length, incomplete: false });
  } catch (error) {
    if (current !== generation) return;
    if (error instanceof ApiError && error.status === 409) {
      set({ loading: false, incomplete: true });
      return;
    }
    set({ loading: false, error: error instanceof ApiError ? error.message : "Chargement impossible." });
  }
}

async function swipe(profile: PublicProfile, action: SwipeAction): Promise<void> {
  seen.add(profile.userId);
  set({ queue: state.queue.filter((p) => p.userId !== profile.userId), error: null });
  if (state.queue.length <= REFILL_THRESHOLD && !state.exhausted) {
    void load();
  }
  try {
    const result = await matchingApi.swipe(profile.userId, action);
    if (result.matched && !result.duplicate && result.match) {
      set({ match: result.match });
    }
  } catch (error) {
    // Give the card back so the decision is not silently lost.
    seen.delete(profile.userId);
    set({
      queue: [profile, ...state.queue],
      error: error instanceof ApiError ? error.message : "Action impossible pour le moment."
    });
  }
}

export const discoverStore = {
  load,
  swipe,
  dismissMatch: () => set({ match: null }),
  clearError: () => set({ error: null }),
  /** Forces a fresh fetch, e.g. after preferences changed. */
  refresh: () => {
    generation += 1;
    seen = new Set();
    state = { ...initial, match: state.match };
    listeners.forEach((listener) => listener());
    void load();
  },
  reset: () => {
    generation += 1;
    seen = new Set();
    set({ ...initial });
  }
};

export function useDiscover(): DiscoverState {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    () => state,
    () => state
  );
}
