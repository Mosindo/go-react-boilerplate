import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { ensureFreshAccessToken, websocketUrl } from "../api/client";
import type { MatchItem, Message, MessagesPage } from "../api/types";
import { prependMessage } from "../lib/messages";
import { queryKeys } from "../lib/queryKeys";

type RealtimeValue = {
  connected: boolean;
  /** Latest match pushed by the server, to celebrate it in the UI. */
  incomingMatch: MatchItem | null;
  dismissMatch: () => void;
};

const RealtimeContext = createContext<RealtimeValue>({ connected: false, incomingMatch: null, dismissMatch: () => undefined });

type ServerEvent = { type: string; data: unknown };

export function RealtimeProvider({ userId, children }: { userId: string; children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [incomingMatch, setIncomingMatch] = useState<MatchItem | null>(null);
  const attempts = useRef(0);

  useEffect(() => {
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    const resync = () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    };

    const handle = (event: ServerEvent) => {
      switch (event.type) {
        case "message": {
          const message = event.data as Message;
          queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(message.conversationId), (old) =>
            old ? { ...old, pages: prependMessage(old.pages, message) } : old
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        }
        case "read": {
          const { conversationId } = event.data as { conversationId: string };
          void queryClient.invalidateQueries({ queryKey: queryKeys.messages(conversationId) });
          break;
        }
        case "match":
          setIncomingMatch(event.data as MatchItem);
          resync();
          break;
        case "unmatch":
          resync();
          break;
        default:
          break;
      }
    };

    const connect = async () => {
      if (stopped) return;
      const token = await ensureFreshAccessToken();
      if (!token || stopped) return;
      const ws = new WebSocket(websocketUrl());
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (e) => {
        try {
          const event = JSON.parse(String(e.data)) as ServerEvent;
          if (event.type === "ready") {
            attempts.current = 0;
            setConnected(true);
            resync(); // catch up on anything missed while offline
            return;
          }
          handle(event);
        } catch {
          // Ignore malformed frames.
        }
      };
      ws.onclose = () => {
        setConnected(false);
        if (stopped) return;
        const delay = Math.min(30_000, 1000 * 2 ** attempts.current++);
        retryTimer = setTimeout(() => void connect(), delay);
      };
      ws.onerror = () => ws.close();
    };

    void connect();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active" && (!socket || socket.readyState === WebSocket.CLOSED)) {
        attempts.current = 0;
        void connect();
      }
    });

    return () => {
      stopped = true;
      appState.remove();
      if (retryTimer) clearTimeout(retryTimer);
      socket?.close();
    };
  }, [queryClient, userId]);

  const value = useMemo(() => ({ connected, incomingMatch, dismissMatch: () => setIncomingMatch(null) }), [connected, incomingMatch]);
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeValue {
  return useContext(RealtimeContext);
}
