import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { API_BASE_URL, type MatchView, type RealtimeEvent } from "../api";
import { getFreshAccessToken } from "../auth/session";
import { useAuth } from "../auth/AuthContext";
import { applyReadReceipt, insertMessage, type MessagePage } from "../lib/chatCache";
import { queryKeys } from "../lib/queryKeys";

type RealtimeValue = {
  connected: boolean;
  /** A match received while the app is open (null once dismissed). */
  incomingMatch: MatchView | null;
  dismissMatch: () => void;
  showMatch: (match: MatchView) => void;
};

const RealtimeContext = createContext<RealtimeValue>({
  connected: false,
  incomingMatch: null,
  dismissMatch: () => undefined,
  showMatch: () => undefined
});

const MAX_BACKOFF_MS = 30_000;

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status, user } = useAuth();
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [incomingMatch, setIncomingMatch] = useState<MatchView | null>(null);
  const userId = user?.id ?? null;
  const signedIn = status === "signedIn";
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!signedIn || !userId) return;
    let stopped = false;
    let attempt = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "ready":
          // Anything may have changed while offline: resync the lists.
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        case "message": {
          const msg = event.data;
          queryClient.setQueryData<InfiniteData<MessagePage, number | undefined>>(
            queryKeys.messages(msg.matchId),
            (d) => insertMessage(d, msg)
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        }
        case "read":
          queryClient.setQueryData<InfiniteData<MessagePage, number | undefined>>(
            queryKeys.messages(event.data.matchId),
            (d) => applyReadReceipt(d, event.data.readerId, event.data.readAt)
          );
          break;
        case "match":
          setIncomingMatch(event.data);
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        case "match_ended":
          queryClient.removeQueries({ queryKey: queryKeys.messages(event.data.matchId) });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
      }
    };

    const connect = async () => {
      if (stopped) return;
      const token = await getFreshAccessToken();
      if (!token || stopped) return;
      const ws = new WebSocket(`${API_BASE_URL.replace(/^http/, "ws")}/ws`);
      socketRef.current = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (e) => {
        try {
          const event = JSON.parse(String(e.data)) as RealtimeEvent;
          if (event.type === "ready") {
            attempt = 0;
            setConnected(true);
          }
          handle(event);
        } catch {
          // ignore malformed frames
        }
      };
      ws.onclose = () => {
        setConnected(false);
        if (socketRef.current === ws) socketRef.current = null;
        if (stopped) return;
        const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt++);
        retryTimer = setTimeout(() => void connect(), delay);
      };
      ws.onerror = () => ws.close();
    };

    void connect();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active" && !socketRef.current && !stopped) {
        if (retryTimer) clearTimeout(retryTimer);
        attempt = 0;
        void connect();
      }
    });

    return () => {
      stopped = true;
      if (retryTimer) clearTimeout(retryTimer);
      appState.remove();
      socketRef.current?.close();
      socketRef.current = null;
      setConnected(false);
    };
  }, [signedIn, userId, queryClient]);

  const value = useMemo<RealtimeValue>(
    () => ({
      connected,
      incomingMatch,
      dismissMatch: () => setIncomingMatch(null),
      showMatch: (m) => setIncomingMatch(m)
    }),
    [connected, incomingMatch]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeValue {
  return useContext(RealtimeContext);
}
