import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { fetchDiscover } from "../../api/discover";
import { initialDeck, deckReducer, shouldPrefetch } from "./deck";

/** Loads discovery pages into the deck and prefetches when few cards remain. */
export function useDiscoverDeck(enabled: boolean) {
  const [deck, dispatch] = useReducer(deckReducer, initialDeck);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const inFlight = useRef(false);
  const generation = useRef(0);

  const loadMore = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    const mine = generation.current;
    setLoading(true);
    try {
      const profiles = await fetchDiscover(10);
      if (mine === generation.current) {
        dispatch({ type: "append", profiles });
        setError(null);
      }
    } catch (e) {
      if (mine === generation.current) setError(e);
    } finally {
      inFlight.current = false;
      if (mine === generation.current) setLoading(false);
    }
  }, []);

  const refresh = useCallback(() => {
    generation.current += 1;
    inFlight.current = false;
    dispatch({ type: "reset" });
    setError(null);
    setLoading(false);
    void loadMore();
  }, [loadMore]);

  useEffect(() => {
    if (!enabled || error) return;
    if (shouldPrefetch(deck.cards.length, loading, deck.exhausted)) void loadMore();
  }, [enabled, error, deck.cards.length, deck.exhausted, loading, loadMore]);

  return { deck, dispatch, loading, error, loadMore, refresh };
}
