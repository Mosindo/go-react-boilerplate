import React, { createContext, useContext, useEffect, useMemo, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { WS_URL } from "../config";
import { getTokens } from "../auth/session";
import type { Message } from "../api/types";

export type RealtimeEvent =
  | { type: "message"; data: Message }
  | { type: "read"; data: { conversationId: string; readerId: string; upToMessageId: number } }
  | { type: "match"; data: { matchId: string; conversationId: string } }
  | { type: "unmatch"; data: { matchId: string } };

type Handler = (event: RealtimeEvent) => void;

type RealtimeApi = { subscribe: (handler: Handler) => () => void };

const RealtimeContext = createContext<RealtimeApi | null>(null);

const KNOWN = new Set(["message", "read", "match", "unmatch"]);

/**
 * Keeps one authenticated WebSocket alive (exponential backoff) and keeps the
 * list caches fresh. Screens that need live data subscribe to events; after a
 * reconnect everything is refetched, so dropped events can never leave stale UI.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const handlers = useRef(new Set<Handler>());

  useEffect(() => {
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let stopped = false;

    const refreshLists = () => {
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
      void queryClient.invalidateQueries({ queryKey: ["notifications"] });
      void queryClient.invalidateQueries({ queryKey: ["likes"] });
    };

    const connect = () => {
      const tokens = getTokens();
      if (stopped || !tokens) {
        return;
      }
      socket = new WebSocket(WS_URL);
      socket.onopen = () => {
        socket?.send(JSON.stringify({ type: "auth", token: tokens.accessToken }));
      };
      socket.onmessage = (raw) => {
        let parsed: { type?: string; data?: unknown };
        try {
          parsed = JSON.parse(String(raw.data)) as { type?: string; data?: unknown };
        } catch {
          return;
        }
        if (parsed.type === "ready") {
          attempt = 0;
          refreshLists();
          return;
        }
        if (parsed.type && KNOWN.has(parsed.type)) {
          const event = parsed as RealtimeEvent;
          handlers.current.forEach((handler) => handler(event));
          refreshLists();
        }
      };
      socket.onclose = () => {
        if (stopped) {
          return;
        }
        const delay = Math.min(30_000, 1000 * 2 ** attempt);
        attempt += 1;
        timer = setTimeout(connect, delay);
      };
      socket.onerror = () => socket?.close();
    };

    connect();
    return () => {
      stopped = true;
      if (timer) {
        clearTimeout(timer);
      }
      socket?.close();
    };
  }, [queryClient]);

  const api = useMemo<RealtimeApi>(
    () => ({
      subscribe: (handler) => {
        handlers.current.add(handler);
        return () => {
          handlers.current.delete(handler);
        };
      }
    }),
    []
  );

  return <RealtimeContext.Provider value={api}>{children}</RealtimeContext.Provider>;
}

export function useRealtimeEvents(handler: Handler): void {
  const ctx = useContext(RealtimeContext);
  const saved = useRef(handler);
  saved.current = handler;
  useEffect(() => ctx?.subscribe((event) => saved.current(event)), [ctx]);
}
