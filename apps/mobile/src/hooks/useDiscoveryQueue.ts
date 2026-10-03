import { useCallback, useRef, useState } from "react";
import { discover, errorMessage, swipe, type PublicProfile, type SwipeAction, type SwipeResult } from "../api/platform";

const BATCH = 10;
const PREFETCH_AT = 3;

type QueueState = { queue: PublicProfile[]; loading: boolean; exhausted: boolean; error: string | null; incomplete: boolean };

/**
 * Keeps a short queue of candidates ahead of the user. Swiped ids are remembered so a late
 * server response can never put a card back on screen.
 */
export function useDiscoveryQueue() {
  const [state, setState] = useState<QueueState>({ queue: [], loading: true, exhausted: false, error: null, incomplete: false });
  const seen = useRef(new Set<string>());
  const inflight = useRef(false);

  const load = useCallback(async (reset = false) => {
    if (inflight.current) {
      return;
    }
    inflight.current = true;
    if (reset) {
      seen.current.clear();
    }
    setState((s) => ({ ...s, loading: true, error: null, ...(reset ? { queue: [], exhausted: false } : {}) }));
    try {
      const batch = await discover(BATCH);
      setState((s) => {
        const known = new Set(s.queue.map((p) => p.id));
        const fresh = batch.filter((p) => !seen.current.has(p.id) && !known.has(p.id));
        return { queue: [...(reset ? [] : s.queue), ...fresh], loading: false, exhausted: fresh.length === 0, error: null, incomplete: false };
      });
    } catch (e) {
      const incomplete = (e as { code?: string }).code === "profile_incomplete";
      setState((s) => ({ ...s, loading: false, error: incomplete ? null : errorMessage(e), incomplete }));
    } finally {
      inflight.current = false;
    }
  }, []);

  const decide = useCallback(
    async (profile: PublicProfile, action: SwipeAction): Promise<SwipeResult | null> => {
      seen.current.add(profile.id);
      let remaining = 0;
      setState((s) => {
        const queue = s.queue.filter((p) => p.id !== profile.id);
        remaining = queue.length;
        return { ...s, queue };
      });
      if (remaining <= PREFETCH_AT) {
        void load();
      }
      try {
        return await swipe(profile.id, action);
      } catch (e) {
        // Put the card back so nothing is silently lost.
        seen.current.delete(profile.id);
        setState((s) => ({ ...s, queue: [profile, ...s.queue], error: errorMessage(e) }));
        return null;
      }
    },
    [load]
  );

  const dismissError = useCallback(() => setState((s) => ({ ...s, error: null })), []);

  return { ...state, load, decide, dismissError };
}
