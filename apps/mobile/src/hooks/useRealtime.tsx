import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import { realtimeApi } from "../api/platform";
import { API_BASE_URL } from "../api/client";
import { queryKeys } from "../api/queryClient";
import type { Message, RealtimeEvent } from "../api/types";
import { showToast } from "../shared/feedback/store";
import { useAuth } from "./useAuth";

export type MessagesPage = { messages: Message[]; hasMore: boolean };

type RealtimeValue = { connected: boolean };

const RealtimeContext = createContext<RealtimeValue>({ connected: false });

let activeConversationId: string | null = null;

/** The open conversation screen registers itself so its own messages do not toast. */
export function setActiveConversation(conversationId: string | null): void {
  activeConversationId = conversationId;
}

export function toSocketUrl(baseUrl: string, ticket: string): string {
  return `${baseUrl.replace(/^http/, "ws")}/ws?ticket=${encodeURIComponent(ticket)}`;
}

/** Applies a server event to the React Query cache. Exported for tests. */
export function applyRealtimeEvent(queryClient: QueryClient, event: RealtimeEvent, currentUserId: string): void {
  switch (event.type) {
    case "message": {
      const message = event.data;
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(message.conversationId), (existing) => {
        if (!existing || existing.pages.length === 0) {
          return existing;
        }
        const exists = existing.pages.some((page) => page.messages.some((m) => m.id === message.id));
        if (exists) {
          return existing;
        }
        // page 0 holds the newest messages
        const [first, ...rest] = existing.pages;
        return { ...existing, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      if (message.senderId !== currentUserId && activeConversationId !== message.conversationId) {
        showToast("Nouveau message");
      }
      return;
    }
    case "read": {
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(event.data.conversationId), (existing) =>
        existing
          ? {
              ...existing,
              pages: existing.pages.map((page) => ({
                ...page,
                messages: page.messages.map((m) => (m.senderId === currentUserId && !m.readAt ? { ...m, readAt: event.data.readAt } : m))
              }))
            }
          : existing
      );
      return;
    }
    case "match":
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      showToast("C'est un match ! 🎉");
      return;
    case "unmatch":
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      return;
    case "notification":
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      return;
  }
}

function parseEvent(raw: string): RealtimeEvent | null {
  try {
    const parsed = JSON.parse(raw) as RealtimeEvent;
    return typeof parsed?.type === "string" ? parsed : null;
  } catch {
    return null;
  }
}

/**
 * Keeps one WebSocket open while signed in. Reconnects with exponential
 * backoff; while offline the screens fall back to polling (see `connected`).
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { user, isAuthenticated } = useAuth();
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const userId = user?.id ?? null;
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!isAuthenticated || !userId) {
      return undefined;
    }
    let disposed = false;
    let attempt = 0;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;

    const scheduleRetry = () => {
      if (disposed) {
        return;
      }
      const delay = Math.min(30_000, 1000 * 2 ** attempt++);
      retryTimer = setTimeout(connect, delay);
    };

    async function connect() {
      try {
        const { ticket } = await realtimeApi.ticket();
        if (disposed) {
          return;
        }
        const socket = new WebSocket(toSocketUrl(API_BASE_URL, ticket));
        socketRef.current = socket;
        socket.onopen = () => {
          attempt = 0;
          setConnected(true);
          // events may have been missed while offline
          void queryClient.invalidateQueries();
        };
        socket.onmessage = (message) => {
          const event = parseEvent(String(message.data));
          if (event) {
            applyRealtimeEvent(queryClient, event, userId as string);
          }
        };
        socket.onclose = () => {
          setConnected(false);
          socketRef.current = null;
          scheduleRetry();
        };
        socket.onerror = () => socket.close();
      } catch {
        scheduleRetry();
      }
    }

    void connect();
    return () => {
      disposed = true;
      clearTimeout(retryTimer);
      socketRef.current?.close();
      socketRef.current = null;
      setConnected(false);
    };
  }, [isAuthenticated, userId, queryClient]);

  const value = useMemo(() => ({ connected }), [connected]);
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeValue {
  return useContext(RealtimeContext);
}
