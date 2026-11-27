import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "../../api/client";
import { discoveryApi } from "../../api/endpoints";
import type { MatchSummary, ProfileCard, SwipeAction } from "../../api/types";
import { keys, queryClient } from "../../hooks/queryClient";
import { markLikeInFlight } from "../../hooks/useRealtime";
import { prefetchImages } from "../../ui";

const REFILL_THRESHOLD = 3;

/**
 * Local queue of profiles to review. Swipes are optimistic: the card leaves
 * immediately and comes back only if the request fails for a network reason.
 * Already-processed ids are remembered so a refill never shows them again.
 */
export function useDeck(onMatch: (match: MatchSummary) => void) {
  const [queue, setQueue] = useState<ProfileCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [exhausted, setExhausted] = useState(false);
  const processed = useRef(new Set<string>());
  const inFlight = useRef(false);

  const load = useCallback(async (reset = false) => {
    if (inFlight.current) {
      return;
    }
    inFlight.current = true;
    if (reset) {
      setLoading(true);
    }
    try {
      const cards = await discoveryApi.feed(10);
      const fresh = cards.filter((c) => !processed.current.has(c.userId));
      setQueue((current) => {
        const base = reset ? [] : current;
        const known = new Set(base.map((c) => c.userId));
        return [...base, ...fresh.filter((c) => !known.has(c.userId))];
      });
      setExhausted(fresh.length === 0);
      setError(null);
      prefetchImages(fresh.slice(0, 3).flatMap((c) => c.photos.slice(0, 1).map((p) => p.url)));
    } catch (e) {
      setError(e);
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    if (!loading && !exhausted && queue.length <= REFILL_THRESHOLD) {
      void load();
    }
    // Warm the cache for the next cards' other photos.
    prefetchImages(queue.slice(1, 3).flatMap((c) => c.photos.slice(0, 2).map((p) => p.url)));
  }, [queue, loading, exhausted, load]);

  const swipe = useCallback(
    async (card: ProfileCard, action: SwipeAction) => {
      processed.current.add(card.userId);
      setQueue((q) => q.filter((c) => c.userId !== card.userId));
      if (action === "like") {
        markLikeInFlight(card.userId, true);
      }
      try {
        const result = await discoveryApi.swipe(card.userId, action);
        if (result.matched && result.match && !result.alreadySwiped) {
          void queryClient.invalidateQueries({ queryKey: keys.matches });
          onMatch(result.match);
        }
      } catch (e) {
        if (e instanceof ApiError && (e.status === 0 || e.status >= 500 || e.status === 429)) {
          processed.current.delete(card.userId);
          setQueue((q) => [card, ...q]);
          throw e;
        }
        // 404 (profile no longer available) and other client errors: drop it.
      } finally {
        if (action === "like") {
          markLikeInFlight(card.userId, false);
        }
      }
    },
    [onMatch]
  );

  const reload = useCallback(() => {
    setExhausted(false);
    void load(true);
  }, [load]);

  return { queue, loading, error, exhausted, swipe, reload };
}
