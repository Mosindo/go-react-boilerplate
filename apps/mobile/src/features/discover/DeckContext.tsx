import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  useRef,
  useState,
} from 'react';
import type { ReactNode } from 'react';
import { discoverApi } from '../../api';
import type { PublicProfile, SwipeAction } from '../../api';
import { useErrorToast } from '../../lib/useErrorToast';
import { BATCH_SIZE, deckReducer, initialDeck, shouldPrefetch } from './deckLogic';
import type { DeckState } from './deckLogic';
import { useSwipe } from './useSwipe';

interface DeckContextValue {
  state: DeckState;
  /** Last fetch failed; cleared by retry(). */
  loadError: unknown;
  /** Optimistically removes the card, then sends the swipe; restores the card on failure. */
  act: (profile: PublicProfile, action: SwipeAction) => void;
  /** Drops a profile everywhere in the deck (blocked / reported). */
  remove: (userId: string) => void;
  /** Re-queries the server (after widening preferences). */
  retry: () => void;
}

const DeckContext = createContext<DeckContextValue | null>(null);

export function DeckProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(deckReducer, initialDeck);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<unknown>(null);
  const fetching = useRef(false);
  const swipe = useSwipe();
  const onError = useErrorToast();

  const loadMore = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    setLoading(true);
    try {
      const profiles = await discoverApi.batch(BATCH_SIZE);
      dispatch({ type: 'batch', profiles });
      setLoadError(null);
    } catch (error) {
      setLoadError(error);
    } finally {
      fetching.current = false;
      setLoading(false);
    }
  }, []);

  // Prefetch the next batch whenever 3 or fewer cards remain.
  useEffect(() => {
    if (loadError === null && shouldPrefetch(state, loading)) void loadMore();
  }, [state, loading, loadError, loadMore]);

  const act = useCallback<DeckContextValue['act']>(
    (profile, action) => {
      dispatch({ type: 'remove', userId: profile.userId });
      swipe(profile, action).catch((error: unknown) => {
        dispatch({ type: 'restore', profile });
        onError(error);
      });
    },
    [swipe, onError],
  );

  const remove = useCallback((userId: string) => dispatch({ type: 'remove', userId }), []);

  const retry = useCallback(() => {
    setLoadError(null);
    dispatch({ type: 'resume' });
  }, []);

  const value = useMemo(
    () => ({ state, loadError, act, remove, retry }),
    [state, loadError, act, remove, retry],
  );
  return <DeckContext.Provider value={value}>{children}</DeckContext.Provider>;
}

export function useDeck(): DeckContextValue {
  const ctx = useContext(DeckContext);
  if (!ctx) throw new Error('useDeck must be used inside <DeckProvider>');
  return ctx;
}
