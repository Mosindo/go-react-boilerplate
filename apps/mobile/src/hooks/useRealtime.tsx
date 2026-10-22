import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { fetchMe } from "../api/auth";
import { websocketUrl } from "../api/client";
import type { Match, RealtimeEvent } from "../api/types";
import { getAccessToken } from "../store/tokenStore";
import { useAuth } from "./useAuth";

type Listener = (event: RealtimeEvent) => void;

type RealtimeContextValue = {
  connected: boolean;
  /** Match waiting to be celebrated in the "It's a match" overlay. */
  pendingMatch: Match | null;
  announceMatch: (match: Match) => void;
  dismissMatch: () => void;
  subscribe: (listener: Listener) => () => void;
};

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

const MAX_BACKOFF_MS = 15_000;

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, user } = useAuth();
  const queryClient = useQueryClient();
  const listeners = useRef(new Set<Listener>());
  const [connected, setConnected] = useState(false);
  const [pendingMatch, setPendingMatch] = useState<Match | null>(null);
  const userId = user?.id ?? null;

  const subscribe = useCallback((listener: Listener) => {
    listeners.current.add(listener);
    return () => {
      listeners.current.delete(listener);
    };
  }, []);

  const handleEvent = useCallback(
    (event: RealtimeEvent) => {
      switch (event.type) {
        case "match.new":
          setPendingMatch(event.data);
          void queryClient.invalidateQueries({ queryKey: ["matches"] });
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          break;
        case "match.removed":
          void queryClient.invalidateQueries({ queryKey: ["matches"] });
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          setPendingMatch((current) => (current?.id === event.data.matchId ? null : current));
          break;
        case "message.new":
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          void queryClient.invalidateQueries({ queryKey: ["matches"] });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          break;
        case "ready":
          // After a reconnect, anything may have changed while we were away.
          void queryClient.invalidateQueries();
          break;
        default:
          break;
      }
      listeners.current.forEach((listener) => listener(event));
    },
    [queryClient]
  );

  useEffect(() => {
    if (!isAuthenticated || !userId) {
      return undefined;
    }
    let stopped = false;
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let attempt = 0;

    const scheduleReconnect = () => {
      if (stopped) {
        return;
      }
      const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt);
      attempt += 1;
      retryTimer = setTimeout(connect, delay);
    };

    const connect = async () => {
      try {
        // A cheap authenticated call refreshes an expired access token before we present it.
        await fetchMe();
        const token = await getAccessToken();
        if (stopped || !token) {
          return;
        }
        socket = new WebSocket(websocketUrl());
        socket.onopen = () => socket?.send(JSON.stringify({ type: "auth", token }));
        socket.onmessage = (message) => {
          try {
            const event = JSON.parse(String(message.data)) as RealtimeEvent;
            if (event.type === "ready") {
              attempt = 0;
              setConnected(true);
            }
            handleEvent(event);
          } catch {
            // Ignore malformed frames.
          }
        };
        socket.onclose = () => {
          setConnected(false);
          scheduleReconnect();
        };
        socket.onerror = () => socket?.close();
      } catch {
        scheduleReconnect();
      }
    };

    void connect();
    return () => {
      stopped = true;
      clearTimeout(retryTimer);
      socket?.close();
      setConnected(false);
    };
  }, [isAuthenticated, userId, handleEvent]);

  const value = useMemo<RealtimeContextValue>(
    () => ({
      connected,
      pendingMatch,
      announceMatch: setPendingMatch,
      dismissMatch: () => setPendingMatch(null),
      subscribe
    }),
    [connected, pendingMatch, subscribe]
  );

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  const context = useContext(RealtimeContext);
  if (!context) {
    throw new Error("useRealtime must be used inside RealtimeProvider");
  }
  return context;
}

/** Subscribe to raw realtime events for the lifetime of a component. */
export function useRealtimeEvents(listener: Listener) {
  const { subscribe } = useRealtime();
  const latest = useRef(listener);
  latest.current = listener;
  useEffect(() => subscribe((event) => latest.current(event)), [subscribe]);
}
