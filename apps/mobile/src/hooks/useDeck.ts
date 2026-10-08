import { useCallback, useEffect, useRef, useState } from "react";
import { describeError } from "../api/client";
import { getDiscover } from "../api/platform";
import type { PublicProfile } from "../api/types";
import { useSwipedVersion, wasSwiped } from "../state/swiped";

const PREFETCH_THRESHOLD = 3;

type DeckStatus = "loading" | "ready" | "error";

/** Local queue of profiles to decide on, topped up from the server before it runs dry. */
export function useDeck(enabled: boolean) {
  const [cards, setCards] = useState<PublicProfile[]>([]);
  const [status, setStatus] = useState<DeckStatus>("loading");
  const [error, setError] = useState<string | null>(null);
  const [exhausted, setExhausted] = useState(false);
  const inFlight = useRef(false);
  const swipedVersion = useSwipedVersion();

  const cardsRef = useRef<PublicProfile[]>([]);
  cardsRef.current = cards;

  const load = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    try {
      const fresh = await getDiscover(10);
      const known = new Set(cardsRef.current.map((card) => card.id));
      const additions = fresh.filter((card) => !known.has(card.id) && !wasSwiped(card.id));
      if (additions.length > 0) setCards((previous) => [...previous, ...additions.filter((card) => !previous.some((p) => p.id === card.id))]);
      // Nothing new means the server has no more candidates for now: stop prefetching until a reload.
      setExhausted(additions.length === 0);
      setStatus("ready");
      setError(null);
    } catch (e) {
      setError(describeError(e));
      setStatus((current) => (current === "loading" ? "error" : current));
    } finally {
      inFlight.current = false;
    }
  }, []);

  useEffect(() => {
    if (enabled) void load();
  }, [enabled, load]);

  useEffect(() => {
    if (enabled && status === "ready" && cards.length <= PREFETCH_THRESHOLD && !exhausted) void load();
  }, [enabled, status, cards.length, exhausted, load]);

  useEffect(() => {
    setCards((previous) => previous.filter((card) => !wasSwiped(card.id)));
  }, [swipedVersion]);

  const remove = useCallback((id: string) => setCards((previous) => previous.filter((card) => card.id !== id)), []);
  const restore = useCallback((card: PublicProfile) => setCards((previous) => [card, ...previous.filter((c) => c.id !== card.id)]), []);
  const reload = useCallback(async () => {
    setExhausted(false);
    setStatus((current) => (current === "error" ? "loading" : current));
    await load();
  }, [load]);

  return { cards, status, error, exhausted, remove, restore, reload };
}
