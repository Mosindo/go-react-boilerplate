import type { Candidate } from "../api/types";

export const PREFETCH_THRESHOLD = 3;

export type DeckState = {
  cards: Candidate[];
  /** Cards removed optimistically whose swipe request has not settled yet. */
  inFlight: Record<string, Candidate>;
  /** Ids the user already acted on this session; never shown again. */
  seen: Record<string, true>;
  loading: boolean;
  /** The server had nothing new to offer at the last fetch. */
  exhausted: boolean;
  loadError: boolean;
};

export type DeckAction =
  | { type: "fetchStarted" }
  | { type: "fetchSucceeded"; items: Candidate[] }
  | { type: "fetchFailed" }
  | { type: "swiped"; userId: string }
  | { type: "swipeConfirmed"; userId: string }
  | { type: "swipeRolledBack"; userId: string }
  | { type: "swipeDropped"; userId: string }
  | { type: "reset" };

export const initialDeck: DeckState = {
  cards: [],
  inFlight: {},
  seen: {},
  loading: false,
  exhausted: false,
  loadError: false
};

function without<T>(record: Record<string, T>, key: string): Record<string, T> {
  const next = { ...record };
  delete next[key];
  return next;
}

export function deckReducer(state: DeckState, action: DeckAction): DeckState {
  switch (action.type) {
    case "fetchStarted":
      return { ...state, loading: true, loadError: false };
    case "fetchSucceeded": {
      const known = new Set(state.cards.map((card) => card.userId));
      const fresh = action.items.filter((card) => {
        if (known.has(card.userId) || state.seen[card.userId] || state.inFlight[card.userId]) {
          return false;
        }
        known.add(card.userId);
        return true;
      });
      return {
        ...state,
        cards: [...state.cards, ...fresh],
        loading: false,
        exhausted: fresh.length === 0,
        loadError: false
      };
    }
    case "fetchFailed":
      return { ...state, loading: false, loadError: true };
    case "swiped": {
      const top = state.cards[0];
      if (!top || top.userId !== action.userId) {
        return state;
      }
      return {
        ...state,
        cards: state.cards.slice(1),
        inFlight: { ...state.inFlight, [top.userId]: top },
        seen: { ...state.seen, [top.userId]: true }
      };
    }
    case "swipeConfirmed":
    case "swipeDropped":
      return { ...state, inFlight: without(state.inFlight, action.userId) };
    case "swipeRolledBack": {
      const card = state.inFlight[action.userId];
      if (!card) {
        return state;
      }
      return {
        ...state,
        cards: [card, ...state.cards],
        inFlight: without(state.inFlight, action.userId),
        seen: without(state.seen, action.userId)
      };
    }
    case "reset":
      return { ...initialDeck, inFlight: state.inFlight, seen: state.seen };
  }
}

export function shouldPrefetch(state: DeckState): boolean {
  return state.cards.length < PREFETCH_THRESHOLD && !state.loading && !state.exhausted && !state.loadError;
}
