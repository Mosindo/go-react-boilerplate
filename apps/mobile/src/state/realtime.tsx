import { useEffect, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import { websocketUrl } from "../api/config";
import { realtimeApi } from "../api/endpoints";
import { queryKeys } from "../api/queryKeys";
import type { Match, Message, MessagesPage } from "../api/types";
import { useMatchCelebration } from "./matchCelebration";
import { useSession } from "./session";

type Event =
  | { type: "ready"; payload: null }
  | { type: "message.new"; payload: Message }
  | { type: "conversation.read"; payload: { conversationId: string; readAt: string } }
  | { type: "conversation.closed"; payload: { conversationId: string; matchId: string } }
  | { type: "match.new"; payload: Match }
  | { type: "notifications.unread"; payload: { unreadCount: number } };

export function addMessageToCache(client: QueryClient, message: Message): void {
  client.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(message.conversationId), (data) => {
    if (!data || data.pages.length === 0) return data;
    if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) return data;
    const [first, ...rest] = data.pages;
    return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
  });
}

export function markReadInCache(client: QueryClient, conversationId: string, readAt: string): void {
  const limit = new Date(readAt).getTime();
  client.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(conversationId), (data) => {
    if (!data) return data;
    return {
      ...data,
      pages: data.pages.map((p) => ({
        ...p,
        partnerLastReadAt: readAt,
        messages: p.messages.map((m) => (m.isMine && !m.isRead && new Date(m.createdAt).getTime() <= limit ? { ...m, isRead: true } : m))
      }))
    };
  });
}

/**
 * Keeps one WebSocket open while signed in and applies server events to the
 * React Query cache. Reconnects with exponential backoff and resynchronises
 * over REST after every (re)connection, so missed events are never lost.
 */
export function useRealtime(): void {
  const { status } = useSession();
  const client = useQueryClient();
  const { celebrate } = useMatchCelebration();
  const celebrateRef = useRef(celebrate);
  celebrateRef.current = celebrate;

  useEffect(() => {
    if (status !== "signedIn") return;
    let socket: WebSocket | null = null;
    let stopped = false;
    let attempt = 0;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const handle = (event: Event) => {
      switch (event.type) {
        case "ready":
          attempt = 0;
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          void client.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
          break;
        case "message.new":
          addMessageToCache(client, event.payload);
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        case "conversation.read":
          markReadInCache(client, event.payload.conversationId, event.payload.readAt);
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        case "conversation.closed":
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          void client.invalidateQueries({ queryKey: queryKeys.conversation(event.payload.conversationId) });
          break;
        case "match.new":
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          void client.invalidateQueries({ queryKey: queryKeys.likesReceived });
          celebrateRef.current(event.payload);
          break;
        case "notifications.unread":
          client.setQueryData(queryKeys.unreadNotifications, event.payload.unreadCount);
          void client.invalidateQueries({ queryKey: queryKeys.notifications, exact: true });
          break;
      }
    };

    const schedule = () => {
      if (stopped) return;
      const delay = Math.min(30_000, 1000 * 2 ** attempt) + Math.random() * 500;
      attempt += 1;
      timer = setTimeout(connect, delay);
    };

    async function connect() {
      if (stopped) return;
      try {
        const ticket = await realtimeApi.ticket();
        if (stopped) return;
        socket = new WebSocket(websocketUrl(`/realtime?ticket=${encodeURIComponent(ticket)}`));
        socket.onmessage = (msg) => {
          try {
            handle(JSON.parse(String(msg.data)) as Event);
          } catch {
            // Ignore malformed frames.
          }
        };
        socket.onclose = () => {
          socket = null;
          schedule();
        };
        socket.onerror = () => socket?.close();
      } catch {
        schedule();
      }
    }

    void connect();
    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active" && !socket && !stopped) {
        if (timer) clearTimeout(timer);
        attempt = 0;
        void connect();
      }
    });

    return () => {
      stopped = true;
      sub.remove();
      if (timer) clearTimeout(timer);
      socket?.close();
    };
  }, [status, client]);
}
