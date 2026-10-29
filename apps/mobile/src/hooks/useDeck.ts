import { useCallback, useEffect, useRef, useState } from "react";
import { fetchDiscover, swipe } from "../api/discovery";
import { ApiError } from "../api/client";
import type { Match, PublicProfile, SwipeAction } from "../api/types";
import {
  markProfileResolved,
  unmarkProfileResolved,
  useDiscoverStore
} from "../state/discoverStore";
import { errorMessage } from "./useAsyncAction";

const REFILL_THRESHOLD = 3;

type Status = "loading" | "ready" | "error";

/**
 * Local queue of profiles to review. The server never returns a profile that
 * was already liked/passed, so refilling only needs to skip what is already queued.
 */
export function useDeck(onMatch: (match: Match) => void, onProblem: (message: string) => void) {
  const { version, resolved } = useDiscoverStore();
  const [deck, setDeck] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<string | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);
  const generation = useRef(0);

  const refill = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    const current = generation.current;
    try {
      const batch = await fetchDiscover(10);
      if (current !== generation.current) return;
      const fresh = batch.filter((p) => !seen.current.has(p.id));
      fresh.forEach((p) => seen.current.add(p.id));
      setDeck((existing) => [...existing, ...fresh]);
      setStatus("ready");
      setError(null);
    } catch (e) {
      if (current !== generation.current) return;
      setError(errorMessage(e));
      setStatus("error");
    } finally {
      fetching.current = false;
    }
  }, []);

  const reload = useCallback(() => {
    generation.current += 1;
    fetching.current = false;
    seen.current = new Set();
    setDeck([]);
    setStatus("loading");
    void refill();
  }, [refill]);

  // First load, and reload when preferences or the profile changed.
  useEffect(() => {
    reload();
  }, [version, reload]);

  // Profiles handled from their detail page leave the deck.
  useEffect(() => {
    if (resolved.size === 0) return;
    setDeck((existing) => existing.filter((p) => !resolved.has(p.id)));
  }, [resolved]);

  useEffect(() => {
    if (status === "ready" && deck.length <= REFILL_THRESHOLD) {
      void refill();
    }
  }, [deck.length, status, refill]);

  const act = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      setDeck((existing) => existing.filter((p) => p.id !== profile.id));
      markProfileResolved(profile.id);
      try {
        const result = await swipe(profile.id, action);
        if (result.matched && result.match && !result.alreadyProcessed) {
          onMatch(result.match);
        }
      } catch (e) {
        // 404/409: the profile is gone or was already answered, nothing to restore.
        if (e instanceof ApiError && (e.status === 404 || e.status === 409)) return;
        seen.current.delete(profile.id);
        unmarkProfileResolved(profile.id);
        setDeck((existing) => [profile, ...existing]);
        onProblem(errorMessage(e, "Action impossible pour le moment."));
      }
    },
    [onMatch, onProblem]
  );

  return { deck, status, error, act, reload };
}
