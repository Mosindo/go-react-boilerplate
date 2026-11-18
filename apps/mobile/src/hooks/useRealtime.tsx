import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { realtimeUrl } from "../api/client";
import { realtimeApi } from "../api/endpoints";
import type { AppNotification, Match, Message } from "../api/types";
import { appendMessage, forgetRelationship, setOtherReadAt, upsertChatSummary } from "./queries";
import { queryKeys } from "./queryClient";

type RealtimeEvent =
  | { type: "connection.ready"; data: { userId: string } }
  | { type: "message.created"; data: Message }
  | { type: "conversation.read"; data: { userId: string; readAt: string } }
  | { type: "match.created"; data: Match }
  | { type: "match.removed"; data: { userId: string } }
  | { type: "notification.created"; data: AppNotification };

type MatchListener = (match: Match) => void;

type RealtimeContextValue = {
  connected: boolean;
  /** Chat screens register the conversation they display to skip unread bumps. */
  setActiveConversation: (userId: string | null) => void;
  onMatch: (listener: MatchListener) => () => void;
};

const RealtimeContext = createContext<RealtimeContextValue>({
  connected: false,
  setActiveConversation: () => undefined,
  onMatch: () => () => undefined
});

/**
 * Keeps a WebSocket open while the app is in the foreground and folds server
 * events into the React Query cache. Reconnects with exponential backoff; the
 * screens fall back to polling while disconnected.
 */
export function RealtimeProvider({ userId, children }: { userId: string; children: React.ReactNode }) {
  const qc = useQueryClient();
  const [connected, setConnected] = useState(false);
  const activeConversation = useRef<string | null>(null);
  const matchListeners = useRef(new Set<MatchListener>());

  useEffect(() => {
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let attempts = 0;
    let stopped = false;
    let foreground = AppState.currentState !== "background";

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.created": {
          const message = event.data;
          const other = message.senderUserId === userId ? message.recipientUserId : message.senderUserId;
          appendMessage(qc, other, message);
          const incoming = message.senderUserId !== userId && activeConversation.current !== other;
          upsertChatSummary(qc, other, message, incoming);
          break;
        }
        case "conversation.read":
          setOtherReadAt(qc, event.data.userId, event.data.readAt);
          break;
        case "match.created":
          qc.setQueryData<Match[]>(queryKeys.matches, (prev) =>
            prev ? [event.data, ...prev.filter((m) => m.id !== event.data.id)] : prev
          );
          matchListeners.current.forEach((listener) => listener(event.data));
          break;
        case "match.removed":
          forgetRelationship(qc, event.data.userId);
          break;
        case "notification.created":
          void qc.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        default:
          break;
      }
    };

    const schedule = () => {
      if (stopped || !foreground) {
        return;
      }
      const delay = Math.min(30_000, 1_000 * 2 ** attempts) + Math.random() * 500;
      attempts += 1;
      retryTimer = setTimeout(connect, delay);
    };

    async function connect() {
      if (stopped || !foreground || socket) {
        return;
      }
      try {
        const ticket = await realtimeApi.ticket();
        if (stopped) {
          return;
        }
        const ws = new WebSocket(realtimeUrl(ticket));
        socket = ws;
        ws.onopen = () => {
          attempts = 0;
          setConnected(true);
          // Catch up on anything missed while offline.
          void qc.invalidateQueries({ queryKey: queryKeys.chats });
          void qc.invalidateQueries({ queryKey: queryKeys.matches });
          void qc.invalidateQueries({ queryKey: queryKeys.notifications });
        };
        ws.onmessage = (msg) => {
          try {
            handle(JSON.parse(String(msg.data)) as RealtimeEvent);
          } catch {
            // Ignore malformed frames.
          }
        };
        ws.onclose = () => {
          socket = null;
          setConnected(false);
          schedule();
        };
        ws.onerror = () => ws.close();
      } catch {
        socket = null;
        schedule();
      }
    }

    void connect();
    const sub = AppState.addEventListener("change", (state) => {
      foreground = state === "active";
      if (foreground) {
        attempts = 0;
        void connect();
      } else {
        socket?.close();
      }
    });

    return () => {
      stopped = true;
      sub.remove();
      if (retryTimer) {
        clearTimeout(retryTimer);
      }
      socket?.close();
    };
  }, [qc, userId]);

  const value = useRef<RealtimeContextValue>({
    connected: false,
    setActiveConversation: (id) => {
      activeConversation.current = id;
    },
    onMatch: (listener) => {
      matchListeners.current.add(listener);
      return () => matchListeners.current.delete(listener);
    }
  });

  return <RealtimeContext.Provider value={{ ...value.current, connected }}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  return useContext(RealtimeContext);
}
