import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import { API_BASE_URL, getFreshAccessToken } from "../api/client";
import type { Match, Message, MessagesPage } from "../api/types";

type RealtimeContextValue = {
  connected: boolean;
  incomingMatch: Match | null;
  dismissMatch: () => void;
};

const RealtimeContext = createContext<RealtimeContextValue>({
  connected: false,
  incomingMatch: null,
  dismissMatch: () => undefined
});

type ServerEvent = { type: string; data?: unknown };

const MAX_BACKOFF_MS = 30_000;

function websocketUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}

/** Prepends a freshly received message to the cached first page of a conversation. */
export function insertMessage(
  cache: InfiniteData<MessagesPage> | undefined,
  message: Message
): InfiniteData<MessagesPage> | undefined {
  if (!cache || cache.pages.length === 0) return cache;
  if (cache.pages.some((page) => page.messages.some((m) => m.id === message.id))) return cache;
  const [first, ...rest] = cache.pages;
  if (!first) return cache;
  return { ...cache, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

function handleEvent(queryClient: QueryClient, event: ServerEvent, onMatch: (match: Match) => void): void {
  switch (event.type) {
    case "message.new": {
      const message = event.data as Message;
      queryClient.setQueryData<InfiniteData<MessagesPage>>(["messages", message.conversationId], (cache) =>
        insertMessage(cache, message)
      );
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
      break;
    }
    case "message.read": {
      const { conversationId } = event.data as { conversationId: string };
      void queryClient.invalidateQueries({ queryKey: ["messages", conversationId] });
      break;
    }
    case "match.new":
      onMatch(event.data as Match);
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      void queryClient.invalidateQueries({ queryKey: ["likes"] });
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
      break;
    case "match.removed": {
      const { conversationId } = event.data as { conversationId: string };
      queryClient.removeQueries({ queryKey: ["messages", conversationId] });
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      break;
    }
    case "notification.new":
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
      break;
    default:
      break;
  }
}

/**
 * Keeps one authenticated WebSocket alive while mounted: reconnects with
 * exponential backoff, re-syncs the caches after every reconnect and when the
 * app returns to the foreground.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [incomingMatch, setIncomingMatch] = useState<Match | null>(null);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let stopped = false;
    let attempt = 0;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let pingTimer: ReturnType<typeof setInterval> | undefined;

    const scheduleRetry = () => {
      if (stopped) return;
      const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt);
      attempt += 1;
      retryTimer = setTimeout(() => void connect(), delay);
    };

    const connect = async () => {
      if (stopped || socketRef.current) return;
      const token = await getFreshAccessToken();
      if (!token || stopped) return;

      const socket = new WebSocket(websocketUrl());
      socketRef.current = socket;

      socket.onopen = () => socket.send(JSON.stringify({ type: "auth", token }));
      socket.onmessage = (message) => {
        let event: ServerEvent;
        try {
          event = JSON.parse(String(message.data)) as ServerEvent;
        } catch {
          return;
        }
        if (event.type === "ready") {
          attempt = 0;
          setConnected(true);
          // Anything missed while offline is picked up from the REST API.
          void queryClient.invalidateQueries();
          return;
        }
        handleEvent(queryClient, event, setIncomingMatch);
      };
      socket.onclose = () => {
        socketRef.current = null;
        setConnected(false);
        scheduleRetry();
      };
      socket.onerror = () => socket.close();
    };

    void connect();
    pingTimer = setInterval(() => {
      const socket = socketRef.current;
      if (socket && socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: "ping" }));
    }, 25_000);

    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        if (!socketRef.current) {
          attempt = 0;
          void connect();
        }
        void queryClient.invalidateQueries();
      }
    });

    return () => {
      stopped = true;
      clearTimeout(retryTimer);
      clearInterval(pingTimer);
      appState.remove();
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [queryClient]);

  const dismissMatch = useCallback(() => setIncomingMatch(null), []);
  const value = useMemo(
    () => ({ connected, incomingMatch, dismissMatch }),
    [connected, incomingMatch, dismissMatch]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  return useContext(RealtimeContext);
}
