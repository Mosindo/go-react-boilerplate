import { useEffect } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import type { Message, MessagesPage } from "../api/types";
import { getTokens } from "../store/tokenStore";
import { keys } from "./queries";
import { setRealtimeConnected } from "./realtimeStatus";

type ServerEvent =
  | { type: "ready" }
  | { type: "message.new"; data: Message }
  | { type: "message.read"; data: { conversationId: string; userId: string; readAt: string } }
  | { type: "match.new" | "match.removed"; data: { matchId: string; userId?: string } }
  | { type: "notification.new"; data: unknown };

/**
 * Keeps one WebSocket open while signed in and turns server events into
 * React Query cache updates. Reconnects with exponential backoff and pauses
 * while the app is in the background.
 */
export function useRealtime(currentUserId: string, tokenVersion: number) {
  const qc = useQueryClient();

  useEffect(() => {
    if (!API_BASE_URL) return;
    let socket: WebSocket | null = null;
    let retry = 0;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    const handle = (event: ServerEvent) => {
      switch (event.type) {
        case "ready":
          setRealtimeConnected(true);
          retry = 0;
          break;
        case "message.new": {
          const msg = event.data;
          const otherId = msg.senderUserId === currentUserId ? msg.recipientUserId : msg.senderUserId;
          qc.setQueryData<InfiniteData<MessagesPage>>(keys.messages(otherId), (data) => {
            if (!data) return data;
            const [first, ...rest] = data.pages;
            if (!first || first.messages.some((m) => m.id === msg.id)) return data;
            return { ...data, pages: [{ ...first, messages: [...first.messages, msg] }, ...rest] };
          });
          void qc.invalidateQueries({ queryKey: keys.chats });
          void qc.invalidateQueries({ queryKey: keys.unreadChats });
          void qc.invalidateQueries({ queryKey: keys.matches });
          break;
        }
        case "message.read":
          qc.setQueryData<InfiniteData<MessagesPage>>(keys.messages(event.data.userId), (data) =>
            data ? { ...data, pages: data.pages.map((p, i) => (i === 0 ? { ...p, otherLastReadAt: event.data.readAt } : p)) } : data
          );
          break;
        case "match.new":
        case "match.removed":
          void qc.invalidateQueries({ queryKey: keys.matches });
          void qc.invalidateQueries({ queryKey: keys.chats });
          void qc.invalidateQueries({ queryKey: keys.unreadChats });
          break;
        case "notification.new":
          void qc.invalidateQueries({ queryKey: keys.notifications });
          break;
      }
    };

    const connect = async () => {
      const tokens = await getTokens();
      if (stopped || !tokens) return;
      const url = API_BASE_URL.replace(/^http/, "ws") + "/ws";
      socket = new WebSocket(url);
      socket.onopen = () => socket?.send(JSON.stringify({ type: "auth", token: tokens.accessToken }));
      socket.onmessage = (e) => {
        try {
          handle(JSON.parse(String(e.data)) as ServerEvent);
        } catch {
          // Ignore malformed frames.
        }
      };
      socket.onclose = () => {
        setRealtimeConnected(false);
        if (stopped) return;
        const delay = Math.min(30_000, 1000 * 2 ** retry++);
        timer = setTimeout(() => void connect(), delay);
      };
      socket.onerror = () => socket?.close();
    };

    const disconnect = () => {
      if (timer) clearTimeout(timer);
      timer = null;
      socket?.close();
      socket = null;
      setRealtimeConnected(false);
    };

    void connect();
    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active" && !socket) {
        stopped = false;
        retry = 0;
        void connect();
        void qc.invalidateQueries();
      } else if (state === "background") {
        stopped = true;
        disconnect();
      }
    });

    return () => {
      stopped = true;
      sub.remove();
      disconnect();
    };
  }, [qc, currentUserId, tokenVersion]);
}
