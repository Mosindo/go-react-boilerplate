import React, { createContext, useContext, useMemo, useState, type ReactNode } from "react";

type Value = { hold: boolean; setHold: (hold: boolean) => void };

const Context = createContext<Value>({ hold: false, setHold: () => undefined });

/** Lets the onboarding keep focus until the member finishes the last step. */
export function OnboardingHoldProvider({ children }: { children: ReactNode }) {
  const [hold, setHold] = useState(false);
  const value = useMemo(() => ({ hold, setHold }), [hold]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useOnboardingHold(): Value {
  return useContext(Context);
}
