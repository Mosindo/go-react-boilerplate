import type { PublicProfile } from "../../api";

export const MIN_QUEUE_SIZE = 3;
export const PAGE_SIZE = 10;

export type DeckState = {
  /** Cards still to swipe, top of the deck first. */
  queue: PublicProfile[];
  /** Every user id that has entered the deck during this session, used to dedupe refills. */
  seen: string[];
  /** True once the server returned nothing new: there is no one left to show. */
  exhausted: boolean;
};

export type DeckAction =
  | { type: "received"; profiles: PublicProfile[] }
  | { type: "swiped"; userId: string }
  | { type: "restore"; profile: PublicProfile }
  | { type: "resume" };

export const initialDeckState: DeckState = { queue: [], seen: [], exhausted: false };

export function deckReducer(state: DeckState, action: DeckAction): DeckState {
  switch (action.type) {
    case "received": {
      const known = new Set(state.seen);
      const fresh: PublicProfile[] = [];
      for (const profile of action.profiles) {
        if (!known.has(profile.userId)) {
          known.add(profile.userId);
          fresh.push(profile);
        }
      }
      return {
        queue: [...state.queue, ...fresh],
        seen: [...state.seen, ...fresh.map((profile) => profile.userId)],
        exhausted: fresh.length === 0
      };
    }
    case "swiped":
      return { ...state, queue: state.queue.filter((profile) => profile.userId !== action.userId) };
    case "restore":
      return state.queue.some((profile) => profile.userId === action.profile.userId)
        ? state
        : { ...state, queue: [action.profile, ...state.queue] };
    case "resume":
      return { ...state, exhausted: false };
  }
}

export function needsRefill(state: DeckState): boolean {
  return state.queue.length < MIN_QUEUE_SIZE && !state.exhausted;
}
