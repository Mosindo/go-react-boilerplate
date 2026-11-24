import React, { createContext, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { realtimeApi } from "../api/endpoints";
import type { Match, Message } from "../api/types";
import { websocketUrl } from "../config";
import { insertMessage, markOwnMessagesRead } from "../hooks/chatCache";
import { queryKeys } from "../queryClient";
import { showToast } from "../components/Toast";
import { wasMatchAnnounced } from "./matchAnnouncements";

type Status = "idle" | "connecting" | "open" | "closed";

const RealtimeContext = createContext<Status>("idle");

type ServerEvent =
  | { type: "ready" }
  | { type: "message.created"; data: Message }
  | { type: "conversation.read"; data: { conversationId: string; userId: string; lastReadAt: string } }
  | { type: "match.created"; data: Match }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "notification.created"; data: unknown };

/**
 * Keeps one WebSocket per signed-in session and folds server events into the
 * React Query cache. Reconnects with exponential backoff and resynchronises
 * over REST after each reconnection.
 */
export function RealtimeProvider({ userId, children }: { userId: string | null; children: ReactNode }) {
  const client = useQueryClient();
  const [status, setStatus] = useState<Status>("idle");
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!userId) {
      setStatus("idle");
      return;
    }
    let disposed = false;
    let attempt = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    const handle = (event: ServerEvent) => {
      switch (event.type) {
        case "message.created": {
          insertMessage(client, event.data);
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        }
        case "conversation.read":
          markOwnMessagesRead(client, event.data.conversationId, userId, event.data.lastReadAt);
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        case "match.created":
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          // The swipe response may still be in flight: give it a moment to
          // claim the celebration before falling back to a toast.
          setTimeout(() => {
            if (!wasMatchAnnounced(event.data.id)) {
              showToast(`Nouveau match avec ${event.data.user.firstName} ✨`, "success");
            }
          }, 1500);
          break;
        case "match.removed":
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        case "notification.created":
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          void client.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
          break;
        default:
          break;
      }
    };

    const scheduleReconnect = () => {
      if (disposed) return;
      attempt += 1;
      const delay = Math.min(30_000, 1000 * 2 ** Math.min(attempt, 5));
      retryTimer = setTimeout(connect, delay);
    };

    async function connect() {
      if (disposed) return;
      setStatus("connecting");
      let ticket: string;
      try {
        ticket = (await realtimeApi.ticket()).ticket;
      } catch {
        setStatus("closed");
        scheduleReconnect();
        return;
      }
      if (disposed) return;
      const socket = new WebSocket(websocketUrl(`/realtime?ticket=${encodeURIComponent(ticket)}`));
      socketRef.current = socket;
      socket.onopen = () => {
        attempt = 0;
        setStatus("open");
        // Catch up on anything missed while disconnected.
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
        void client.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
      };
      socket.onmessage = (message) => {
        try {
          handle(JSON.parse(String(message.data)) as ServerEvent);
        } catch {
          // ignore malformed frames
        }
      };
      socket.onclose = () => {
        if (socketRef.current === socket) socketRef.current = null;
        setStatus("closed");
        scheduleReconnect();
      };
      socket.onerror = () => socket.close();
    }

    void connect();
    const appState = AppState.addEventListener("change", (next) => {
      if (next === "active" && !socketRef.current) {
        if (retryTimer) clearTimeout(retryTimer);
        attempt = 0;
        void connect();
      }
    });

    return () => {
      disposed = true;
      appState.remove();
      if (retryTimer) clearTimeout(retryTimer);
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, [userId, client]);

  return <RealtimeContext.Provider value={status}>{children}</RealtimeContext.Provider>;
}

export function useRealtimeStatus(): Status {
  return useContext(RealtimeContext);
}
