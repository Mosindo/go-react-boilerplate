import React, { createContext, useContext, useEffect, useState } from "react";
import { AppState } from "react-native";
import { getValidAccessToken } from "../api/http";

const REFRESH_EVERY_MS = 5 * 60 * 1000;
const AccessTokenContext = createContext<string | null>(null);

/** Keeps a fresh access token in React state so protected images can send it as a header. */
export function AccessTokenProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    const load = () => {
      getValidAccessToken()
        .then((next) => {
          if (active && next) {
            setToken((previous) => (previous === next ? previous : next));
          }
        })
        .catch(() => {
          // Keep the previous token; the API client reports auth problems elsewhere.
        });
    };
    load();
    const timer = setInterval(load, REFRESH_EVERY_MS);
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        load();
      }
    });
    return () => {
      active = false;
      clearInterval(timer);
      subscription.remove();
    };
  }, []);

  return <AccessTokenContext.Provider value={token}>{children}</AccessTokenContext.Provider>;
}

export function useAccessToken(): string | null {
  return useContext(AccessTokenContext);
}
