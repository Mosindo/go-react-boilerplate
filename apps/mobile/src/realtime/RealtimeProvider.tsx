import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL, currentAccessToken } from "../api/client";
import type { MatchItem, RealtimeEvent } from "../api/types";
import { queryKeys } from "../api/keys";
import { toWebSocketUrl } from "../lib/url";
import { applyRealtimeEvent } from "./events";
import { setActiveConversation, getActiveConversation } from "./activeConversation";

type RealtimeState = { connected: boolean; lastMatch: MatchItem | null; clearLastMatch: () => void };

const RealtimeContext = createContext<RealtimeState>({
  connected: false,
  lastMatch: null,
  clearLastMatch: () => undefined
});

export function useRealtime(): RealtimeState {
  return useContext(RealtimeContext);
}

const PING_INTERVAL_MS = 25_000;
const MAX_BACKOFF_MS = 30_000;

export function backoffDelay(attempt: number): number {
  return Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt);
}

export function parseRealtimeEvent(raw: string): RealtimeEvent | null {
  try {
    const parsed = JSON.parse(raw) as { type?: unknown };
    return typeof parsed.type === "string" ? (parsed as RealtimeEvent) : null;
  } catch {
    return null;
  }
}

export function RealtimeProvider({ userId, children }: { userId: string; children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [lastMatch, setLastMatch] = useState<MatchItem | null>(null);
  const stopped = useRef(false);

  useEffect(() => {
    stopped.current = false;
    let socket: WebSocket | null = null;
    let pingTimer: ReturnType<typeof setInterval> | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let hadConnection = false;

    const cleanupTimers = () => {
      if (pingTimer) clearInterval(pingTimer);
      pingTimer = null;
    };

    const connect = async () => {
      if (stopped.current || !API_BASE_URL) return;
      const token = await currentAccessToken();
      if (!token || stopped.current) return;
      const ws = new WebSocket(toWebSocketUrl(API_BASE_URL));
      socket = ws;

      ws.onopen = () => {
        ws.send(JSON.stringify({ type: "auth", token }));
      };
      ws.onmessage = (message) => {
        const event = parseRealtimeEvent(String(message.data));
        if (!event) return;
        if (event.type === "ready") {
          attempt = 0;
          setConnected(true);
          cleanupTimers();
          pingTimer = setInterval(() => ws.send(JSON.stringify({ type: "ping" })), PING_INTERVAL_MS);
          if (hadConnection) {
            // Anything missed while offline is recovered with a refetch.
            void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
            void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
            void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
            void queryClient.invalidateQueries({ queryKey: ["messages"] });
          }
          hadConnection = true;
          return;
        }
        if (event.type === "match.new") setLastMatch(event.match);
        applyRealtimeEvent(queryClient, event, { userId, activeConversationId: getActiveConversation() });
      };
      ws.onclose = () => {
        cleanupTimers();
        setConnected(false);
        if (stopped.current) return;
        retryTimer = setTimeout(() => void connect(), backoffDelay(attempt));
        attempt += 1;
      };
      ws.onerror = () => ws.close();
    };

    void connect();

    return () => {
      stopped.current = true;
      cleanupTimers();
      if (retryTimer) clearTimeout(retryTimer);
      socket?.close();
      setActiveConversation(null);
    };
  }, [queryClient, userId]);

  const value = useMemo<RealtimeState>(
    () => ({ connected, lastMatch, clearLastMatch: () => setLastMatch(null) }),
    [connected, lastMatch]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}
