import { useCallback, useEffect, useRef, useState } from "react";
import { errorMessage, isApiError } from "../../api/client";
import { discoveryApi } from "../../api/endpoints";
import type { PublicProfile, SwipeAction, SwipeResponse } from "../../api/types";
import { mergeDeck, shouldPrefetch } from "../../utils/deck";

type DeckState = { deck: PublicProfile[]; loading: boolean; exhausted: boolean; error: string | null };

export type UseDeck = DeckState & {
  act: (action: SwipeAction) => void;
  reload: () => void;
  match: NonNullable<SwipeResponse["match"]> | null;
  dismissMatch: () => void;
  actionError: string | null;
};

/** Discovery deck: dedupes, prefetches when few cards remain, advances optimistically. */
export function useDeck(onMatched?: () => void): UseDeck {
  const [state, setState] = useState<DeckState>({ deck: [], loading: false, exhausted: false, error: null });
  const [match, setMatch] = useState<UseDeck["match"]>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const seen = useRef(new Set<string>());
  const inFlight = useRef(false);
  const onMatchedRef = useRef(onMatched);
  onMatchedRef.current = onMatched;

  const load = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const res = await discoveryApi.deck(10);
      setState((s) => {
        const merged = mergeDeck(s.deck, res.profiles, seen.current);
        return { deck: merged, loading: false, exhausted: merged.length === s.deck.length, error: null };
      });
    } catch (err) {
      setState((s) => ({ ...s, loading: false, error: errorMessage(err) }));
    } finally {
      inFlight.current = false;
    }
  }, []);

  useEffect(() => {
    if (!state.error && shouldPrefetch(state.deck.length, state.loading, state.exhausted)) void load();
  }, [state.deck.length, state.loading, state.exhausted, state.error, load]);

  const latest = useRef(state);
  latest.current = state;

  const act = useCallback((action: SwipeAction) => {
    const top = latest.current.deck[0];
    if (!top || seen.current.has(top.userId)) return;
    seen.current.add(top.userId);
    setState((s) => ({ ...s, deck: s.deck.filter((p) => p.userId !== top.userId) }));
    discoveryApi
      .swipe(top.userId, action)
      .then((res) => {
        if (res.result === "matched" && res.match && !res.alreadyProcessed) {
          setMatch({ ...res.match, profile: res.match.profile ?? top });
          onMatchedRef.current?.();
        }
      })
      .catch((err: unknown) => {
        setActionError(errorMessage(err));
        // 404 = profile no longer available (blocked/hidden): keep it gone. Other errors: put it back.
        if (!isApiError(err) || err.status !== 404) {
          seen.current.delete(top.userId);
          setState((cur) => ({ ...cur, deck: [top, ...cur.deck.filter((p) => p.userId !== top.userId)] }));
        }
      });
  }, []);

  const reload = useCallback(() => {
    seen.current.clear();
    setState({ deck: [], loading: false, exhausted: false, error: null });
  }, []);

  return { ...state, act, reload, match, dismissMatch: () => setMatch(null), actionError };
}
