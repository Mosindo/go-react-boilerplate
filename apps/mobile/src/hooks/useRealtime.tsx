import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { AppState } from "react-native";
import type { InfiniteData } from "@tanstack/react-query";
import { openRealtimeSocket, parseEvent, type RealtimeEvent } from "../api/realtime";
import type { Match, MessagesPage } from "../api/types";
import { queryClient, queryKeys } from "./queryClient";

type RealtimeState = { connected: boolean };
const RealtimeContext = createContext<RealtimeState>({ connected: false });

type MatchListener = (match: Match) => void;
const matchListeners = new Set<MatchListener>();

/** Subscribe to matches created by the *other* person while this user is online. */
export function onIncomingMatch(listener: MatchListener): () => void {
  matchListeners.add(listener);
  return () => {
    matchListeners.delete(listener);
  };
}

export function applyRealtimeEvent(event: RealtimeEvent, currentUserId: string) {
  switch (event.type) {
    case "message.created": {
      const message = event.data;
      queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(message.conversationId), (data) => {
        if (!data || data.pages.length === 0) return data;
        const exists = data.pages.some((p) => p.messages.some((m) => m.id === message.id));
        if (exists) return data;
        const [first, ...rest] = data.pages;
        return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      if (message.senderId !== currentUserId) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.unreadConversations });
      }
      break;
    }
    case "conversation.read": {
      const receipt = event.data;
      if (receipt.userId !== currentUserId) {
        queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(receipt.conversationId), (data) => {
          if (!data || data.pages.length === 0) return data;
          const [first, ...rest] = data.pages;
          return { ...data, pages: [{ ...first, otherLastReadAt: receipt.lastReadAt }, ...rest] };
        });
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadConversations });
      break;
    }
    case "match.created":
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
      matchListeners.forEach((l) => l(event.data));
      break;
    case "match.removed":
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadConversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      void queryClient.invalidateQueries({ queryKey: queryKeys.messages(event.data.conversationId) });
      break;
    case "notification.created":
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
      if (event.data.type === "like") {
        void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
      }
      break;
    default:
      break;
  }
}

/** Keeps one WebSocket open while the app is in the foreground, reconnecting with backoff. */
export function RealtimeProvider({ userId, children }: { userId: string; children: React.ReactNode }) {
  const [connected, setConnected] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    let cancelled = false;
    let attempt = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let foreground = AppState.currentState !== "background";

    const scheduleReconnect = () => {
      if (cancelled || !foreground) return;
      const delay = Math.min(30_000, 1000 * 2 ** attempt) + Math.random() * 500;
      attempt += 1;
      retryTimer = setTimeout(connect, delay);
    };

    const connect = async () => {
      if (cancelled || socketRef.current) return;
      try {
        const socket = await openRealtimeSocket();
        if (cancelled) {
          socket.close();
          return;
        }
        socketRef.current = socket;
        socket.onmessage = (msg) => {
          const event = parseEvent(msg.data);
          if (!event) return;
          if (event.type === "ready") {
            attempt = 0;
            setConnected(true);
            // Resynchronise anything that may have been missed while offline.
            void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
            void queryClient.invalidateQueries({ queryKey: ["unread"] });
            return;
          }
          applyRealtimeEvent(event, userId);
        };
        socket.onclose = () => {
          socketRef.current = null;
          setConnected(false);
          scheduleReconnect();
        };
        socket.onerror = () => socket.close();
      } catch {
        scheduleReconnect();
      }
    };

    const sub = AppState.addEventListener("change", (next) => {
      foreground = next !== "background";
      if (foreground) {
        attempt = 0;
        void connect();
      } else {
        socketRef.current?.close();
      }
    });

    void connect();
    return () => {
      cancelled = true;
      sub.remove();
      if (retryTimer) clearTimeout(retryTimer);
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [userId]);

  return <RealtimeContext.Provider value={{ connected }}>{children}</RealtimeContext.Provider>;
}

export function useRealtimeStatus(): RealtimeState {
  return useContext(RealtimeContext);
}
