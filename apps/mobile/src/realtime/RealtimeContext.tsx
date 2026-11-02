import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { realtimeApi } from "../api/endpoints";
import { useAuth } from "../auth/AuthContext";

export type RealtimeEvent = { type: string; data: Record<string, unknown> };
type Handler = (event: RealtimeEvent) => void;

type RealtimeValue = {
  connected: boolean;
  subscribe: (handler: Handler) => () => void;
};

const RealtimeContext = createContext<RealtimeValue>({ connected: false, subscribe: () => () => undefined });

function wsUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws?ticket=${encodeURIComponent(ticket)}`;
}

function parseEvent(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== "string") return null;
  try {
    const value = JSON.parse(raw) as { type?: unknown; data?: unknown };
    if (typeof value.type !== "string") return null;
    const data = typeof value.data === "object" && value.data !== null ? (value.data as Record<string, unknown>) : {};
    return { type: value.type, data };
  } catch {
    return null;
  }
}

/**
 * One WebSocket per signed-in user. It authenticates with a single-use ticket,
 * reconnects with exponential backoff and when the app returns to foreground.
 * REST remains the source of truth: on every (re)connect the caches are
 * invalidated so nothing missed while offline is lost.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const queryClient = useQueryClient();
  const handlers = useRef(new Set<Handler>());
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (status !== "in") return;
    let socket: WebSocket | null = null;
    let retry = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let stopped = false;

    const connect = async () => {
      if (stopped) return;
      try {
        const { ticket } = await realtimeApi.ticket();
        if (stopped) return;
        const ws = new WebSocket(wsUrl(ticket));
        socket = ws;
        ws.onopen = () => {
          retry = 0;
          setConnected(true);
          void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          void queryClient.invalidateQueries({ queryKey: ["notifications"] });
        };
        ws.onmessage = (message: WebSocketMessageEvent) => {
          const event = parseEvent(message.data);
          if (!event) return;
          if (
            event.type === "message.new" ||
            event.type === "message.read" ||
            event.type === "match.new" ||
            event.type === "match.removed"
          ) {
            void queryClient.invalidateQueries({ queryKey: ["conversations"] });
          }
          if (event.type === "notification.new") {
            void queryClient.invalidateQueries({ queryKey: ["notifications"] });
          }
          handlers.current.forEach((handler) => handler(event));
        };
        ws.onclose = () => {
          setConnected(false);
          if (stopped) return;
          retry += 1;
          timer = setTimeout(() => void connect(), Math.min(30_000, 1000 * 2 ** Math.min(retry, 5)));
        };
        ws.onerror = () => ws.close();
      } catch {
        retry += 1;
        timer = setTimeout(() => void connect(), Math.min(30_000, 1000 * 2 ** Math.min(retry, 5)));
      }
    };

    void connect();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active" && (!socket || socket.readyState === WebSocket.CLOSED)) {
        if (timer) clearTimeout(timer);
        void connect();
      }
    });

    return () => {
      stopped = true;
      appState.remove();
      if (timer) clearTimeout(timer);
      socket?.close();
      setConnected(false);
    };
  }, [status, queryClient]);

  const value = useMemo<RealtimeValue>(
    () => ({
      connected,
      subscribe: (handler) => {
        handlers.current.add(handler);
        return () => {
          handlers.current.delete(handler);
        };
      }
    }),
    [connected]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeValue {
  return useContext(RealtimeContext);
}

/** Subscribes a component to realtime events for as long as it is mounted. */
export function useRealtimeEvents(handler: Handler): void {
  const { subscribe } = useRealtime();
  const ref = useRef(handler);
  ref.current = handler;
  useEffect(() => subscribe((event) => ref.current(event)), [subscribe]);
}
