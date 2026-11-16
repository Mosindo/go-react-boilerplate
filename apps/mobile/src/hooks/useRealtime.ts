import { useEffect } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { realtimeApi } from "../api/social";
import type { AppNotification, Match, Message } from "../api/types";
import { keys, markOwnMessagesRead, upsertMessage } from "./queries";

export type RealtimeEvent =
  | { type: "message.created"; data: Message }
  | { type: "match.created"; data: Match }
  | { type: "match.removed"; data: { matchId: string } }
  | { type: "conversation.read"; data: { matchId: string; readerId: string; readAt: string } }
  | { type: "notification.created"; data: AppNotification };

type Listener = (event: RealtimeEvent) => void;
const listeners = new Set<Listener>();

/** Lets screens react to live events (e.g. mark a conversation read). */
export function subscribeRealtime(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function wsUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/realtime?ticket=${encodeURIComponent(ticket)}`;
}

/**
 * Keeps one WebSocket open while signed in. Reconnects with exponential
 * backoff and when the app returns to the foreground; on reconnect, caches
 * are invalidated so anything missed while offline is refetched.
 */
export function useRealtime(enabled: boolean) {
  const qc = useQueryClient();

  useEffect(() => {
    if (!enabled) {
      return;
    }
    let socket: WebSocket | null = null;
    let stopped = false;
    let attempt = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.created":
          qc.setQueryData(keys.messages(event.data.matchId), (cache: Parameters<typeof upsertMessage>[0]) => upsertMessage(cache, event.data));
          void qc.invalidateQueries({ queryKey: keys.matches });
          break;
        case "conversation.read":
          qc.setQueryData(keys.messages(event.data.matchId), (cache: Parameters<typeof upsertMessage>[0]) =>
            markOwnMessagesRead(cache, event.data.readerId, event.data.readAt)
          );
          break;
        case "match.created":
        case "match.removed":
          void qc.invalidateQueries({ queryKey: keys.matches });
          if (event.type === "match.removed") {
            qc.removeQueries({ queryKey: keys.messages(event.data.matchId) });
            qc.removeQueries({ queryKey: keys.match(event.data.matchId) });
          }
          break;
        case "notification.created":
          void qc.invalidateQueries({ queryKey: keys.notifications });
          break;
      }
      listeners.forEach((l) => l(event));
    };

    const scheduleReconnect = () => {
      if (stopped || retryTimer) {
        return;
      }
      const delay = Math.min(30_000, 1000 * 2 ** attempt);
      attempt += 1;
      retryTimer = setTimeout(() => {
        retryTimer = null;
        void connect();
      }, delay);
    };

    const connect = async () => {
      if (stopped || (socket && socket.readyState <= 1)) {
        return;
      }
      try {
        const { ticket } = await realtimeApi.ticket();
        if (stopped) {
          return;
        }
        const ws = new WebSocket(wsUrl(ticket));
        socket = ws;
        ws.onopen = () => {
          if (attempt > 0) {
            void qc.invalidateQueries({ queryKey: keys.matches });
            void qc.invalidateQueries({ queryKey: ["messages"] });
            void qc.invalidateQueries({ queryKey: keys.notifications });
          }
          attempt = 0;
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
          scheduleReconnect();
        };
        ws.onerror = () => ws.close();
      } catch {
        scheduleReconnect();
      }
    };

    void connect();
    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        attempt = 0;
        void connect();
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
  }, [enabled, qc]);
}
