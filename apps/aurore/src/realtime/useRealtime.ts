import { useEffect, useRef } from "react";
import { useQueryClient, type InfiniteData, type QueryClient } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { realtimeApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { Message, RealtimeEvent } from "../api/types";

function wsUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws?ticket=${encodeURIComponent(ticket)}`;
}

/** Applies a server event to the query cache so every screen updates without refetch storms. */
export function applyRealtimeEvent(qc: QueryClient, ev: RealtimeEvent, onMatch?: (name: string) => void): void {
  switch (ev.type) {
    case "message.new": {
      // The chat screen caches messages as infinite pages, newest page first.
      qc.setQueryData<InfiniteData<Message[], string | undefined>>(keys.messages(ev.data.conversationId), (old) => {
        if (!old || old.pages.some((page) => page.some((m) => m.id === ev.data.id))) return old;
        const [first = [], ...rest] = old.pages;
        return { ...old, pages: [[ev.data, ...first], ...rest] };
      });
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.unread });
      break;
    }
    case "message.read":
      void qc.invalidateQueries({ queryKey: keys.messages(ev.data.conversationId) });
      void qc.invalidateQueries({ queryKey: keys.conversations });
      break;
    case "match.new":
      void qc.invalidateQueries({ queryKey: keys.matches });
      void qc.invalidateQueries({ queryKey: keys.conversations });
      onMatch?.(ev.data.user.firstName);
      break;
    case "match.removed":
      void qc.invalidateQueries({ queryKey: keys.matches });
      void qc.invalidateQueries({ queryKey: keys.conversations });
      break;
    case "notification.new":
      void qc.invalidateQueries({ queryKey: keys.notifications });
      void qc.invalidateQueries({ queryKey: keys.unread });
      break;
  }
}

/**
 * Keeps one WebSocket open while enabled. It authenticates with a one-time ticket (never the
 * access token in a URL), reconnects with backoff, and resynchronises caches after each reconnect.
 */
export function useRealtime(enabled: boolean, onMatch?: (name: string) => void): void {
  const qc = useQueryClient();
  const onMatchRef = useRef(onMatch);
  onMatchRef.current = onMatch;

  useEffect(() => {
    if (!enabled) return;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let stopped = false;

    const connect = async () => {
      try {
        const ticket = await realtimeApi.ticket();
        if (stopped) return;
        socket = new WebSocket(wsUrl(ticket));
        socket.onopen = () => {
          if (attempt > 0) void qc.invalidateQueries(); // missed events while offline
          attempt = 0;
        };
        socket.onmessage = (e: MessageEvent<string>) => {
          try {
            applyRealtimeEvent(qc, JSON.parse(e.data) as RealtimeEvent, (n) => onMatchRef.current?.(n));
          } catch {
            // ignore malformed frames
          }
        };
        socket.onclose = () => schedule();
        socket.onerror = () => socket?.close();
      } catch {
        schedule();
      }
    };

    const schedule = () => {
      if (stopped) return;
      attempt += 1;
      timer = setTimeout(() => void connect(), Math.min(30_000, 1000 * 2 ** Math.min(attempt, 5)));
    };

    void connect();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      socket?.close();
    };
  }, [enabled, qc]);
}
