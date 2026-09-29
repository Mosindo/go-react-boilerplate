import type { PublicProfile } from "../../api/models";

/** In-memory discovery queue. `seen` holds every user acted on (or queued) during the session. */
export type DiscoverQueueState = {
  queue: PublicProfile[];
  seen: string[];
};

export type DiscoverQueueAction =
  | { type: "append"; profiles: PublicProfile[] }
  | { type: "act"; userId: string }
  | { type: "restore"; profile: PublicProfile }
  | { type: "drop"; userId: string }
  | { type: "reset" };

export const initialDiscoverQueue: DiscoverQueueState = { queue: [], seen: [] };

/** How many cards may remain before the next batch is prefetched. */
export const PREFETCH_THRESHOLD = 3;

export function shouldPrefetch(remaining: number): boolean {
  return remaining <= PREFETCH_THRESHOLD;
}

export function discoverQueueReducer(
  state: DiscoverQueueState,
  action: DiscoverQueueAction
): DiscoverQueueState {
  switch (action.type) {
    case "append": {
      const known = new Set<string>(state.seen);
      for (const p of state.queue) known.add(p.userId);
      const fresh: PublicProfile[] = [];
      for (const p of action.profiles) {
        if (known.has(p.userId)) continue;
        known.add(p.userId);
        fresh.push(p);
      }
      if (fresh.length === 0) return state;
      return { queue: [...state.queue, ...fresh], seen: state.seen };
    }
    case "act":
    case "drop": {
      const queue = state.queue.filter((p) => p.userId !== action.userId);
      const seen = state.seen.includes(action.userId) ? state.seen : [...state.seen, action.userId];
      return { queue, seen };
    }
    case "restore": {
      const seen = state.seen.filter((id) => id !== action.profile.userId);
      const queue = [
        action.profile,
        ...state.queue.filter((p) => p.userId !== action.profile.userId)
      ];
      return { queue, seen };
    }
    case "reset":
      return initialDiscoverQueue;
    default:
      return state;
  }
}
