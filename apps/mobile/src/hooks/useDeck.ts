import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError, errorMessage } from "../api/client";
import { getDeck, swipe as swipeRequest } from "../api/matching";
import type { DeckResponse, PublicProfile, SwipeAction, SwipeResult } from "../api/types";
import { queryClient, queryKeys } from "./queryClient";

const REFILL_THRESHOLD = 3;

type DeckState = {
  queue: PublicProfile[];
  loading: boolean;
  error: string | null;
  reason: DeckResponse["reason"] | null;
};

/**
 * Local queue of discovery profiles. Swipes update the UI immediately and are sent in the
 * background; the queue refills before it runs out and never shows the same profile twice.
 */
export function useDeck(onMatch: (result: SwipeResult) => void) {
  const [state, setState] = useState<DeckState>({ queue: [], loading: true, error: null, reason: null });
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);

  const refill = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    setState((s) => ({ ...s, loading: s.queue.length === 0, error: null }));
    try {
      const deck = await getDeck(10);
      const fresh = deck.profiles.filter((p) => !seen.current.has(p.userId));
      fresh.forEach((p) => seen.current.add(p.userId));
      setState((s) => ({ queue: [...s.queue, ...fresh], loading: false, error: null, reason: deck.reason ?? null }));
    } catch (e) {
      setState((s) => ({ ...s, loading: false, error: errorMessage(e, "Impossible de charger des profils.") }));
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    void refill();
  }, [refill]);

  const reset = useCallback(() => {
    seen.current.clear();
    setState({ queue: [], loading: true, error: null, reason: null });
    fetching.current = false;
    void refill();
  }, [refill]);

  const decide = useCallback(
    (profile: PublicProfile, action: SwipeAction) => {
      setState((s) => {
        const queue = s.queue.filter((p) => p.userId !== profile.userId);
        if (queue.length <= REFILL_THRESHOLD) setTimeout(() => void refill(), 0);
        return { ...s, queue };
      });
      swipeRequest(profile.userId, action)
        .then((result) => {
          if (action === "like") void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
          if (result.matched) {
            void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
            onMatch(result);
          }
        })
        .catch((e: unknown) => {
          // Already answered elsewhere (another device): nothing to undo.
          if (e instanceof ApiError && e.code === "already_swiped") return;
          setState((s) => ({ ...s, error: errorMessage(e) }));
        });
    },
    [onMatch, refill]
  );

  return { ...state, decide, refill, reset };
}
