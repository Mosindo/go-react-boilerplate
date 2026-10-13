import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { endpoints } from "../api/endpoints";
import { me } from "../api/auth";
import type { MessagePage } from "../api/platform";
import type { ChatMessage, RealtimeEvent } from "../api/types";
import { wasMatchAnnounced } from "../lib/matchSeen";
import { queryKeys } from "../lib/queryClient";
import { showToast } from "../shared/feedback";
import { peekTokens } from "../store/tokenStore";

type RealtimeContextValue = { connected: boolean };

const RealtimeContext = createContext<RealtimeContextValue>({ connected: false });

export function useRealtime(): RealtimeContextValue {
  return useContext(RealtimeContext);
}

/** Merges a message pushed by the server into an already-loaded conversation. */
export function appendMessageToCache(queryClient: QueryClient, message: ChatMessage): void {
  queryClient.setQueryData<InfiniteData<MessagePage, number | null>>(queryKeys.messages(message.conversationId), (data) => {
    if (!data || data.pages.length === 0) {
      return data;
    }
    const exists = data.pages.some((page) => page.messages.some((existing) => existing.id === message.id));
    if (exists) {
      return data;
    }
    const [first, ...rest] = data.pages;
    return { ...data, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
  });
}

export function markSentMessagesRead(queryClient: QueryClient, conversationId: string, readAt: string): void {
  queryClient.setQueryData<InfiniteData<MessagePage, number | null>>(queryKeys.messages(conversationId), (data) => {
    if (!data) {
      return data;
    }
    const limit = new Date(readAt).getTime();
    return {
      ...data,
      pages: data.pages.map((page) => ({
        ...page,
        messages: page.messages.map((m) => (new Date(m.createdAt).getTime() <= limit ? { ...m, read: true } : m))
      }))
    };
  });
}

function toWebSocketUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}${endpoints.socket}`;
}

export function handleRealtimeEvent(queryClient: QueryClient, event: RealtimeEvent, currentUserId: string): void {
  switch (event.type) {
    case "message":
      appendMessageToCache(queryClient, event.data);
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      if (event.data.senderId !== currentUserId) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      }
      break;
    case "match":
      // Give the local swipe response a moment to announce its own match first.
      setTimeout(() => {
        if (!wasMatchAnnounced(event.data.conversationId)) {
          showToast("C'est un match ! 🎉");
        }
      }, 800);
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      break;
    case "unmatch":
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      break;
    case "read":
      markSentMessagesRead(queryClient, event.data.conversationId, event.data.readAt);
      break;
    case "notification":
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      break;
    default:
      break;
  }
}

/**
 * Keeps one WebSocket open while the user is signed in. The socket only pushes
 * server events; everything else goes through the REST API. On reconnect all
 * live lists are refetched, so nothing is lost while offline.
 */
export function RealtimeProvider({ userId, children }: { userId: string; children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const attempt = useRef(0);

  useEffect(() => {
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let pingTimer: ReturnType<typeof setInterval> | undefined;
    let disposed = false;

    async function connect() {
      if (disposed || !API_BASE_URL) {
        return;
      }
      // A cheap authenticated call renews the access token if it has expired.
      await me().catch(() => undefined);
      const token = peekTokens()?.accessToken;
      if (disposed || !token) {
        return;
      }
      const ws = new WebSocket(toWebSocketUrl());
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (message) => {
        let event: RealtimeEvent;
        try {
          event = JSON.parse(String(message.data)) as RealtimeEvent;
        } catch {
          return;
        }
        if (event.type === "ready") {
          const reconnected = attempt.current > 0;
          attempt.current = 0;
          setConnected(true);
          if (reconnected) {
            void queryClient.invalidateQueries();
          }
          return;
        }
        handleRealtimeEvent(queryClient, event, userId);
      };
      ws.onclose = () => {
        setConnected(false);
        clearInterval(pingTimer);
        if (disposed) {
          return;
        }
        attempt.current += 1;
        retryTimer = setTimeout(() => void connect(), Math.min(30_000, 1000 * 2 ** Math.min(attempt.current, 5)));
      };
      ws.onerror = () => ws.close();
      pingTimer = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "ping" }));
        }
      }, 25_000);
    }

    void connect();
    return () => {
      disposed = true;
      clearTimeout(retryTimer);
      clearInterval(pingTimer);
      socket?.close();
    };
  }, [queryClient, userId]);

  const value = useMemo(() => ({ connected }), [connected]);
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}
