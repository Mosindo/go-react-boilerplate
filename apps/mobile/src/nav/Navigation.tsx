import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { BackHandler } from "react-native";
import type { Peer } from "../api/types";

export type Tab = "discover" | "messages" | "me";

export type Route =
  | { name: "tabs"; tab: Tab }
  | { name: "profile"; userId: string }
  | { name: "chat"; conversationId: string; peer: Peer; matchId?: string }
  | { name: "notifications" }
  | { name: "editProfile" }
  | { name: "editPhotos" }
  | { name: "editPreferences" }
  | { name: "privacy" }
  | { name: "blocked" };

type NavValue = {
  route: Route;
  canGoBack: boolean;
  push: (route: Route) => void;
  back: () => void;
  /** Replaces the whole stack with a tab root. */
  goTab: (tab: Tab) => void;
};

const NavContext = createContext<NavValue | null>(null);

export function NavigationProvider({ children }: { children: React.ReactNode }) {
  const [stack, setStack] = useState<Route[]>([{ name: "tabs", tab: "discover" }]);

  const push = useCallback((route: Route) => setStack((s) => [...s, route]), []);
  const back = useCallback(() => setStack((s) => (s.length > 1 ? s.slice(0, -1) : s)), []);
  const goTab = useCallback((tab: Tab) => setStack([{ name: "tabs", tab }]), []);

  useEffect(() => {
    const sub = BackHandler.addEventListener("hardwareBackPress", () => {
      if (stack.length > 1) {
        back();
        return true;
      }
      return false;
    });
    return () => sub.remove();
  }, [stack.length, back]);

  const value = useMemo<NavValue>(
    () => ({ route: stack[stack.length - 1], canGoBack: stack.length > 1, push, back, goTab }),
    [stack, push, back, goTab]
  );
  return <NavContext.Provider value={value}>{children}</NavContext.Provider>;
}

export function useNav(): NavValue {
  const ctx = useContext(NavContext);
  if (!ctx) {
    throw new Error("useNav must be used inside NavigationProvider");
  }
  return ctx;
}
