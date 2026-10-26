import { useCallback, useEffect, useRef, useState } from "react";
import { discoverApi, isApiError, type MatchView, type PublicProfile } from "../api";

type State = "loading" | "ready" | "empty" | "error" | "incomplete";

/**
 * Feeds the swipe deck: keeps a small queue of profiles, tops it up before it
 * runs out and never shows a profile twice in a session.
 */
export function useDiscoverDeck(
  enabled: boolean,
  onMatch: (m: MatchView) => void,
  onNotice: (msg: string) => void
) {
  const [queue, setQueue] = useState<PublicProfile[]>([]);
  const [state, setState] = useState<State>("loading");
  const [error, setError] = useState<string | null>(null);
  const seen = useRef(new Set<string>());
  const fetching = useRef(false);
  const exhausted = useRef(false);

  const fill = useCallback(async () => {
    if (fetching.current || exhausted.current) return;
    fetching.current = true;
    try {
      const batch = await discoverApi.next(10);
      const fresh = batch.filter((p) => !seen.current.has(p.userId));
      fresh.forEach((p) => seen.current.add(p.userId));
      if (batch.length === 0 || fresh.length === 0) exhausted.current = true;
      setQueue((q) => [...q, ...fresh]);
      setError(null);
      setState("ready");
    } catch (err) {
      if (isApiError(err, 409)) setState("incomplete");
      else {
        setError(err instanceof Error ? err.message : "Erreur inconnue");
        setState("error");
      }
    } finally {
      fetching.current = false;
    }
  }, []);

  useEffect(() => {
    if (enabled) void fill();
  }, [enabled, fill]);

  useEffect(() => {
    if (state === "ready" && queue.length > 0 && queue.length <= 3) void fill();
  }, [queue.length, state, fill]);

  const swipe = useCallback(
    async (profile: PublicProfile, action: "like" | "pass") => {
      setQueue((q) => q.filter((p) => p.userId !== profile.userId));
      try {
        const result = await discoverApi.swipe(profile.userId, action);
        if (result.matched && result.match) onMatch(result.match);
      } catch (err) {
        if (isApiError(err, 404)) return; // the profile is no longer available
        // Put the card back so the decision is not silently lost.
        setQueue((q) => [profile, ...q]);
        onNotice(err instanceof Error ? err.message : "Action impossible pour le moment.");
      }
    },
    [onMatch, onNotice]
  );

  const reload = useCallback(() => {
    seen.current.clear();
    exhausted.current = false;
    setQueue([]);
    setState("loading");
    void fill();
  }, [fill]);

  const isEmpty = state === "ready" && queue.length === 0 && exhausted.current;
  return { queue, state: isEmpty ? ("empty" as State) : state, error, swipe, reload };
}
