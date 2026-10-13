import { useCallback, useEffect, useRef, useState } from "react";
import { fetchDiscover, swipe } from "../../api/platform";
import { errorMessage, isApiError } from "../../api/client";
import type { PublicProfile, SwipeAction, SwipeResult } from "../../api/types";

const BATCH_SIZE = 10;
const REFILL_THRESHOLD = 3;

export type DeckStatus = "loading" | "ready" | "empty" | "error" | "incomplete";

type UseDeckOptions = {
  onMatch: (profile: PublicProfile, result: SwipeResult) => void;
};

/**
 * Client-side queue over GET /discover. The server never returns a profile the
 * user has already liked or passed, so the queue only needs to skip cards that
 * are still in flight locally. Swipes are optimistic and restored on failure.
 */
export function useDeck({ onMatch }: UseDeckOptions) {
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<DeckStatus>("loading");
  const [error, setError] = useState<string | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);
  const exhausted = useRef(false);
  const onMatchRef = useRef(onMatch);
  onMatchRef.current = onMatch;

  const refill = useCallback(async () => {
    if (fetching.current || exhausted.current) {
      return;
    }
    fetching.current = true;
    try {
      const batch = await fetchDiscover(BATCH_SIZE);
      const fresh = batch.filter((p) => !seen.current.has(p.id));
      fresh.forEach((p) => seen.current.add(p.id));
      if (fresh.length === 0) {
        exhausted.current = true;
      }
      setQueue((current) => {
        const known = new Set(current.map((p) => p.id));
        const next = [...current, ...fresh.filter((p) => !known.has(p.id))];
        setStatus(next.length === 0 ? "empty" : "ready");
        return next;
      });
      setError(null);
    } catch (e) {
      if (isApiError(e, 409)) {
        setStatus("incomplete");
      } else {
        setError(errorMessage(e));
        setStatus((current) => (current === "loading" ? "error" : current));
      }
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    void refill();
  }, [refill]);

  useEffect(() => {
    if (status === "ready" && queue.length <= REFILL_THRESHOLD) {
      void refill();
    }
  }, [queue.length, status, refill]);

  const act = useCallback(async (profile: PublicProfile, action: SwipeAction) => {
    setQueue((current) => {
      const next = current.filter((p) => p.id !== profile.id);
      if (next.length === 0) {
        setStatus(exhausted.current ? "empty" : "loading");
      }
      return next;
    });
    try {
      const result = await swipe(profile.id, action);
      if (result.matched) {
        onMatchRef.current(profile, result);
      }
    } catch (e) {
      if (isApiError(e, 409) || isApiError(e, 404)) {
        return; // already processed or no longer available: the card stays gone
      }
      seen.current.delete(profile.id);
      setQueue((current) => [profile, ...current]);
      setStatus("ready");
    }
  }, []);

  /** A profile left the deck for another reason (blocked, reported). */
  const drop = useCallback((profileId: string) => {
    setQueue((current) => current.filter((p) => p.id !== profileId));
  }, []);

  const reload = useCallback(() => {
    exhausted.current = false;
    setStatus("loading");
    void refill();
  }, [refill]);

  return { queue, status, error, act, drop, reload };
}
