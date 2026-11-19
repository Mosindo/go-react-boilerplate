import React, { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";
import type { Match } from "../api/types";

type Value = {
  current: Match | null;
  celebrate: (match: Match) => void;
  dismiss: () => void;
};

const Ctx = createContext<Value | null>(null);

/** Shows the "It's a match" moment once per match, whether it comes from a swipe or a realtime event. */
export function MatchCelebrationProvider({ children }: { children: React.ReactNode }) {
  const [current, setCurrent] = useState<Match | null>(null);
  const seen = useRef(new Set<string>());

  const celebrate = useCallback((match: Match) => {
    if (seen.current.has(match.id)) return;
    seen.current.add(match.id);
    setCurrent(match);
  }, []);
  const dismiss = useCallback(() => setCurrent(null), []);
  const value = useMemo(() => ({ current, celebrate, dismiss }), [current, celebrate, dismiss]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useMatchCelebration(): Value {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useMatchCelebration must be used within MatchCelebrationProvider");
  return ctx;
}
