import { useCallback, useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { discoveryApi, notificationsApi, profileApi } from "./api/services";
import { ApiError, errorMessage } from "./api/http";
import type { Card, MyProfile } from "./api/types";

/** The signed-in user's profile; `null` means "not created yet" (onboarding needed). */
export function useMyProfile() {
  return useQuery<MyProfile | null>({
    queryKey: ["profile"],
    queryFn: async () => {
      try {
        return await profileApi.mine();
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null;
        throw e;
      }
    },
    retry: 1,
  });
}

export function useUnreadNotifications(): number {
  const q = useQuery({
    queryKey: ["notifications"],
    queryFn: notificationsApi.list,
    refetchInterval: 60_000,
  });
  return q.data?.unreadCount ?? 0;
}

/**
 * Discovery deck: keeps a small buffer of profiles, de-duplicates by id (so a profile never
 * shows twice even if the server re-sends it before a swipe is persisted) and prefetches
 * the next batch when the buffer runs low.
 */
export function useDeck() {
  const [queue, setQueue] = useState<Card[]>([]);
  const [loading, setLoading] = useState(true);
  const [exhausted, setExhausted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);

  const fetchMore = useCallback(async () => {
    if (fetching.current) return;
    fetching.current = true;
    try {
      const batch = await discoveryApi.feed(10);
      const fresh = batch.filter((c) => !seen.current.has(c.userId));
      fresh.forEach((c) => seen.current.add(c.userId));
      setQueue((q) => [...q, ...fresh]);
      if (fresh.length === 0) setExhausted(true);
      setError(null);
    } catch (e) {
      setError(errorMessage(e));
      setExhausted(true);
    } finally {
      fetching.current = false;
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!exhausted && queue.length <= 3) void fetchMore();
  }, [queue.length, exhausted, fetchMore]);

  const shift = useCallback(() => setQueue((q) => q.slice(1)), []);
  const restore = useCallback((card: Card) => setQueue((q) => [card, ...q]), []);
  const reset = useCallback(() => {
    seen.current = new Set();
    setQueue([]);
    setExhausted(false);
    setLoading(true);
    setError(null);
  }, []);

  return { queue, loading, exhausted, error, shift, restore, reset };
}
