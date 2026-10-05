import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "../api/client";
import { getDiscover, swipe } from "../api/platform";
import type { MatchPreview, ProfileCard, SwipeAction } from "../api/types";
import { errorMessage } from "../lib/errors";
import { queryKeys } from "../realtime/cache";
import { showError } from "../shared/feedback";

const BATCH = 10;
const REFILL_AT = 3;

export type DeckStatus = "loading" | "ready" | "empty" | "error" | "incomplete";

/**
 * Discovery queue. The server never proposes someone already liked or passed, so the deck only refills when it
 * runs low; the `seen` set removes profiles that are still waiting in the local queue from the next batch.
 */
export function useDeck(onMatch: (match: MatchPreview) => void) {
  const queryClient = useQueryClient();
  const [cards, setCards] = useState<ProfileCard[]>([]);
  const [status, setStatus] = useState<DeckStatus>("loading");
  const [error, setError] = useState<unknown>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);
  const exhausted = useRef(false);
  const pendingReset = useRef(false);
  const [refillFailed, setRefillFailed] = useState(false);
  const onMatchRef = useRef(onMatch);
  useEffect(() => {
    onMatchRef.current = onMatch;
  });

  const load = useCallback(async (reset = false): Promise<void> => {
    if (fetching.current) {
      // A reset requested during a fetch (settings changed, retry pressed) must not be lost: run it afterwards.
      if (reset) pendingReset.current = true;
      return;
    }
    fetching.current = true;
    if (reset) {
      seen.current.clear();
      exhausted.current = false;
      setStatus("loading");
    }
    try {
      const batch = await getDiscover(BATCH);
      const fresh = batch.filter((card) => !seen.current.has(card.id));
      fresh.forEach((card) => seen.current.add(card.id));
      if (batch.length === 0 || fresh.length === 0) exhausted.current = true;
      setCards((current) => (reset ? fresh : [...current, ...fresh]));
      setStatus("ready");
      setRefillFailed(false);
      setError(null);
    } catch (e) {
      if (e instanceof ApiError && e.code === "profile_incomplete") {
        setStatus("incomplete");
      } else {
        setError(e);
        setRefillFailed(true);
        setStatus((s) => (s === "loading" ? "error" : s));
      }
    } finally {
      fetching.current = false;
      if (pendingReset.current) {
        pendingReset.current = false;
        void load(true);
      }
    }
  }, []);

  useEffect(() => {
    void load(true);
  }, [load]);

  // Refill in the background before the queue runs dry.
  useEffect(() => {
    if (status === "ready" && cards.length <= REFILL_AT && !exhausted.current) void load();
  }, [cards.length, status, load]);

  const act = useCallback(
    async (card: ProfileCard, action: SwipeAction) => {
      setCards((current) => current.filter((c) => c.id !== card.id)); // optimistic: the next card shows at once
      try {
        const result = await swipe(card.id, action);
        if (result.matched && result.match) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
          onMatchRef.current(result.match);
        }
      } catch (e) {
        if (e instanceof ApiError && (e.code === "already_swiped" || e.code === "not_available")) return; // nothing to undo
        setCards((current) => [card, ...current]);
        showError(errorMessage(e));
      }
    },
    [queryClient]
  );

  const reload = useCallback(() => load(true), [load]);

  /** Quietly checks whether new people showed up (no loading state), keeping the profiles already queued. */
  const recheck = useCallback(() => {
    exhausted.current = false;
    return load();
  }, [load]);

  // A drained deck is never blank: empty when the server has nothing more, an error when the refill failed
  // (so the retry button shows), otherwise still loading.
  let derived: DeckStatus = status;
  if (status === "ready" && cards.length === 0) derived = exhausted.current ? "empty" : refillFailed ? "error" : "loading";

  return {
    cards,
    status: derived,
    error,
    act,
    reload,
    recheck
  };
}
