import { useCallback, useEffect, useRef, useState } from "react";
import { Image } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { absoluteUrl } from "../api/client";
import { friendlyError } from "../api/errors";
import { getDiscover, swipe } from "../api/platform";
import type { PublicProfile, SwipeAction, SwipeResult } from "../api/types";
import { isDecided, markDecided, subscribeDecisions, undoDecided } from "../lib/decisions";
import { mergeQueue, needsRefill } from "../lib/swipeQueue";
import { queryKeys } from "./queries";

type DeckStatus = "loading" | "ready" | "error";

const PAGE_SIZE = 10;

/**
 * Local queue of profiles to swipe. The server never returns a profile that was
 * already swiped, but a refill can overlap with the current queue, hence the dedupe.
 */
export function useDiscoverDeck(onMatch: (profile: PublicProfile, result: SwipeResult) => void) {
  const queryClient = useQueryClient();
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<DeckStatus>("loading");
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const seen = useRef(new Set<string>());
  const loading = useRef(false);

  const refill = useCallback(async () => {
    if (loading.current) {
      return;
    }
    loading.current = true;
    try {
      const incoming = await getDiscover(PAGE_SIZE);
      setQueue((current) => {
        const merged = mergeQueue(current, incoming, seen.current);
        setExhausted(merged.length === 0);
        return merged;
      });
      incoming.slice(0, 3).forEach((profile) => {
        const photo = profile.photos[0];
        if (photo) {
          void Image.prefetch(absoluteUrl(photo.url));
        }
      });
      setError(null);
      setStatus("ready");
    } catch (err) {
      setError(friendlyError(err));
      setStatus((current) => (current === "loading" ? "error" : current));
    } finally {
      loading.current = false;
    }
  }, []);

  useEffect(() => {
    void refill();
  }, [refill]);

  // Decisions taken elsewhere (profile page, block) must also leave the deck.
  useEffect(
    () =>
      subscribeDecisions(() => {
        setQueue((current) => current.filter((p) => !isDecided(p.id)));
      }),
    []
  );

  useEffect(() => {
    if (status === "ready" && needsRefill(queue.length) && !exhausted) {
      void refill();
    }
  }, [exhausted, queue.length, refill, status]);

  const reload = useCallback(() => {
    setStatus("loading");
    setExhausted(false);
    void refill();
  }, [refill]);

  const decide = useCallback(
    async (profile: PublicProfile, action: SwipeAction) => {
      seen.current.add(profile.id);
      markDecided(profile.id);
      setQueue((current) => current.filter((p) => p.id !== profile.id));
      setActionError(null);
      try {
        const result = await swipe(profile.id, action);
        if (result.matched && action === "like") {
          onMatch(profile, result);
        }
        if (action === "like") {
          void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
        }
        if (result.matched) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        }
      } catch (err) {
        // Put the profile back so the decision is not silently lost.
        seen.current.delete(profile.id);
        undoDecided(profile.id);
        setQueue((current) => (current.some((p) => p.id === profile.id) ? current : [profile, ...current]));
        setActionError(friendlyError(err));
      }
    },
    [onMatch, queryClient]
  );

  return { queue, status, error, actionError, exhausted, reload, decide };
}
