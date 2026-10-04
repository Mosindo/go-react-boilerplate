import type { PublicProfile } from "../../api/types";

/** Pure state machine for the discovery deck (testable without React). */
export type DeckState = {
  cards: PublicProfile[];
  /** Everyone already shown to or swiped by the user in this session (never re-added). */
  seen: ReadonlySet<string>;
  /** True after the server returned an empty page: nothing more to show right now. */
  exhausted: boolean;
};

export type DeckAction =
  | { type: "append"; profiles: PublicProfile[] }
  | { type: "swipe"; userId: string }
  | { type: "rollback"; profile: PublicProfile }
  | { type: "remove"; userId: string }
  | { type: "reset" };

export const initialDeck: DeckState = { cards: [], seen: new Set(), exhausted: false };

/** Number of cards left that triggers fetching the next page. */
export const PREFETCH_THRESHOLD = 3;

export function deckReducer(state: DeckState, action: DeckAction): DeckState {
  switch (action.type) {
    case "append": {
      const seen = new Set(state.seen);
      const fresh: PublicProfile[] = [];
      for (const profile of action.profiles) {
        if (!seen.has(profile.userId)) {
          seen.add(profile.userId);
          fresh.push(profile);
        }
      }
      return { cards: [...state.cards, ...fresh], seen, exhausted: fresh.length === 0 };
    }
    case "swipe":
    case "remove":
      return { ...state, cards: state.cards.filter((card) => card.userId !== action.userId) };
    case "rollback":
      if (state.cards.some((card) => card.userId === action.profile.userId)) return state;
      return { ...state, cards: [action.profile, ...state.cards], exhausted: false };
    case "reset":
      return initialDeck;
  }
}

export function shouldPrefetch(cardsLeft: number, loading: boolean, exhausted: boolean): boolean {
  return cardsLeft <= PREFETCH_THRESHOLD && !loading && !exhausted;
}
