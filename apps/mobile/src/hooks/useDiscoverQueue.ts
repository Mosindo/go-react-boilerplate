import { useCallback, useEffect, useRef, useState } from "react";
import { errorMessage } from "../api/client";
import { fetchDiscover } from "../api/discovery";
import type { PublicProfile } from "../api/types";

type Status = "loading" | "ready" | "empty" | "error";

const BATCH_SIZE = 10;
const REFILL_AT = 3;

/**
 * Keeps a small queue of profiles to swipe. The server never returns a profile
 * that was already swiped, so the queue is refilled by simply asking again;
 * `seen` only guards the short window while a swipe request is still in flight.
 */
export function useDiscoverQueue() {
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<Status>("loading");
  const [error, setError] = useState<string | null>(null);
  const seen = useRef(new Set<string>());
  const loading = useRef(false);
  const queueRef = useRef<PublicProfile[]>([]);
  queueRef.current = queue;

  const load = useCallback(async () => {
    if (loading.current) {
      return;
    }
    loading.current = true;
    try {
      const batch = await fetchDiscover(BATCH_SIZE);
      const known = new Set(queueRef.current.map((profile) => profile.userId));
      const fresh = batch.filter((profile) => !seen.current.has(profile.userId) && !known.has(profile.userId));
      setQueue((current) => [...current, ...fresh]);
      setError(null);
      setStatus(queueRef.current.length + fresh.length === 0 ? "empty" : "ready");
    } catch (err) {
      setError(errorMessage(err));
      setStatus((current) => (queueRef.current.length > 0 ? current : "error"));
    } finally {
      loading.current = false;
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const dequeue = useCallback(
    (userId: string) => {
      seen.current.add(userId);
      const remaining = queueRef.current.filter((profile) => profile.userId !== userId);
      queueRef.current = remaining;
      setQueue(remaining);
      if (remaining.length === 0) {
        setStatus("loading");
        void load();
      } else if (remaining.length <= REFILL_AT) {
        void load();
      }
    },
    [load]
  );

  /** Puts a profile back after a failed swipe request. */
  const requeue = useCallback((profile: PublicProfile) => {
    seen.current.delete(profile.userId);
    queueRef.current = [profile, ...queueRef.current];
    setQueue(queueRef.current);
    setStatus("ready");
  }, []);

  const refresh = useCallback(() => {
    seen.current.clear();
    queueRef.current = [];
    setQueue([]);
    setStatus("loading");
    void load();
  }, [load]);

  return { queue, status, error, dequeue, requeue, refresh };
}
