import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { markConversationRead } from "../api/chat";
import { fetchRealtimeUrl } from "../api/platform";
import type { Message, MessagesPage, RealtimeEvent } from "../api/types";
import { showToast } from "../shared/feedback";
import { queryKeys } from "./queries";

type RealtimeContextValue = {
  /** Tells the provider which conversation is on screen, so incoming messages are read at once. */
  setActiveConversation: (conversationId: string | null) => void;
};

const RealtimeContext = createContext<RealtimeContextValue>({ setActiveConversation: () => undefined });

const MIN_BACKOFF_MS = 1_000;
const MAX_BACKOFF_MS = 30_000;

function appendMessage(
  data: InfiniteData<MessagesPage> | undefined,
  message: Message
): InfiniteData<MessagesPage> | undefined {
  if (!data || data.pages.length === 0) {
    return data;
  }
  const exists = data.pages.some((page) => page.messages.some((m) => m.id === message.id));
  if (exists) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

/** Keeps one WebSocket open while signed in and folds server events into the query cache. */
export function RealtimeProvider({ children, userId }: { children: React.ReactNode; userId: string }) {
  const client = useQueryClient();
  const activeConversation = useRef<string | null>(null);

  const handle = useCallback(
    (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.new": {
          const message = event.data;
          client.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(message.conversationId), (old) =>
            appendMessage(old, message)
          );
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          if (message.senderId !== userId && activeConversation.current === message.conversationId) {
            void markConversationRead(message.conversationId).then(() =>
              client.invalidateQueries({ queryKey: queryKeys.conversations })
            );
          }
          break;
        }
        case "message.read":
          client.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(event.data.conversationId), (old) =>
            old && old.pages.length > 0
              ? { ...old, pages: [{ ...old.pages[0], otherLastReadAt: event.data.readAt }, ...old.pages.slice(1)] }
              : old
          );
          break;
        case "match.new":
          showToast(`C'est un match avec ${event.data.firstName} !`, "success");
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        case "match.removed":
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        case "notification.new":
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        default:
          break;
      }
    },
    [client, userId]
  );

  useEffect(() => {
    let closed = false;
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let backoff = MIN_BACKOFF_MS;

    const scheduleReconnect = () => {
      if (closed || retryTimer) {
        return;
      }
      retryTimer = setTimeout(() => {
        retryTimer = null;
        void connect();
      }, backoff);
      backoff = Math.min(backoff * 2, MAX_BACKOFF_MS);
    };

    const connect = async () => {
      if (closed || (socket && socket.readyState <= WebSocket.OPEN)) {
        return;
      }
      try {
        const url = await fetchRealtimeUrl();
        if (closed) {
          return;
        }
        const ws = new WebSocket(url);
        socket = ws;
        ws.onopen = () => {
          backoff = MIN_BACKOFF_MS;
          // Events missed while offline: resynchronise through the REST API.
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          if (activeConversation.current) {
            void client.invalidateQueries({ queryKey: queryKeys.messages(activeConversation.current) });
          }
        };
        ws.onmessage = (message) => {
          try {
            handle(JSON.parse(String(message.data)) as RealtimeEvent);
          } catch {
            // Ignore malformed frames.
          }
        };
        ws.onclose = () => {
          socket = null;
          scheduleReconnect();
        };
        ws.onerror = () => ws.close();
      } catch {
        scheduleReconnect();
      }
    };

    void connect();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        void connect();
      }
    });

    return () => {
      closed = true;
      appState.remove();
      if (retryTimer) {
        clearTimeout(retryTimer);
      }
      socket?.close();
    };
  }, [client, handle]);

  const value = useMemo<RealtimeContextValue>(
    () => ({
      setActiveConversation: (conversationId) => {
        activeConversation.current = conversationId;
      }
    }),
    []
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  return useContext(RealtimeContext);
}
