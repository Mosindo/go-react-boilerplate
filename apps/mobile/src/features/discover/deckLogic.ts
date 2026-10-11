import type { PublicProfile } from '../../api/types';

export const PREFETCH_THRESHOLD = 3;
export const BATCH_SIZE = 10;

export interface DeckState {
  /** queue[0] is the card on top. */
  queue: PublicProfile[];
  /** Every id ever shown, so a late batch can't re-add someone already swiped. */
  seen: Record<string, true>;
  /** The server returned nothing new: no one left for the current preferences. */
  exhausted: boolean;
}

export type DeckAction =
  | { type: 'batch'; profiles: PublicProfile[] }
  | { type: 'remove'; userId: string }
  | { type: 'restore'; profile: PublicProfile }
  | { type: 'resume' };

export const initialDeck: DeckState = { queue: [], seen: {}, exhausted: false };

export function deckReducer(state: DeckState, action: DeckAction): DeckState {
  switch (action.type) {
    case 'batch': {
      const seen = { ...state.seen };
      const added: PublicProfile[] = [];
      for (const profile of action.profiles) {
        if (seen[profile.userId]) continue;
        seen[profile.userId] = true;
        added.push(profile);
      }
      return { queue: [...state.queue, ...added], seen, exhausted: added.length === 0 };
    }
    case 'remove':
      return { ...state, queue: state.queue.filter((p) => p.userId !== action.userId) };
    case 'restore':
      return state.queue.some((p) => p.userId === action.profile.userId)
        ? state
        : { ...state, queue: [action.profile, ...state.queue], exhausted: false };
    case 'resume':
      return { ...state, exhausted: false };
  }
}

export function shouldPrefetch(state: DeckState, busy: boolean): boolean {
  return !busy && !state.exhausted && state.queue.length <= PREFETCH_THRESHOLD;
}
