import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { websocketUrl } from "../api/config";
import { realtimeApi } from "../api/endpoints";
import type { MatchPayload, RealtimeEvent } from "../api/types";
import { addMessageToCache, keys } from "./queries";

type RealtimeState = {
  connected: boolean;
  /** Latest match received from the other side (shown as a celebration). */
  incomingMatch: MatchPayload | null;
  dismissIncomingMatch: () => void;
  removedMatchIds: Set<string>;
};

const RealtimeContext = createContext<RealtimeState>({
  connected: false,
  incomingMatch: null,
  dismissIncomingMatch: () => undefined,
  removedMatchIds: new Set()
});

/**
 * Keeps one WebSocket open while signed in and folds server events into the
 * React Query cache. Reconnects with exponential backoff; screens fall back to
 * polling while `connected` is false, so nothing is missed.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [incomingMatch, setIncomingMatch] = useState<MatchPayload | null>(null);
  const [removedMatchIds, setRemoved] = useState<Set<string>>(new Set());
  const attempt = useRef(0);

  useEffect(() => {
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.created":
          addMessageToCache(qc, event.data.message);
          void qc.invalidateQueries({ queryKey: keys.conversations });
          break;
        case "conversation.read":
          void qc.invalidateQueries({ queryKey: keys.messages(event.data.matchId) });
          void qc.invalidateQueries({ queryKey: keys.conversations });
          break;
        case "match.created":
          void qc.invalidateQueries({ queryKey: keys.conversations });
          setIncomingMatch(event.data);
          break;
        case "match.removed":
          setRemoved((prev) => new Set(prev).add(event.data.matchId));
          qc.removeQueries({ queryKey: keys.messages(event.data.matchId) });
          void qc.invalidateQueries({ queryKey: keys.conversations });
          break;
        case "notification.created":
          void qc.invalidateQueries({ queryKey: keys.notifications });
          break;
      }
    };

    const schedule = () => {
      if (stopped) {
        return;
      }
      const delay = Math.min(30_000, 1000 * 2 ** attempt.current);
      attempt.current += 1;
      timer = setTimeout(connect, delay);
    };

    async function connect() {
      try {
        const { ticket } = await realtimeApi.ticket();
        if (stopped) {
          return;
        }
        socket = new WebSocket(websocketUrl(`/realtime?ticket=${encodeURIComponent(ticket)}`));
        socket.onopen = () => {
          attempt.current = 0;
          setConnected(true);
          // Resync anything that happened while offline.
          void qc.invalidateQueries({ queryKey: keys.conversations });
          void qc.invalidateQueries({ queryKey: keys.notifications });
        };
        socket.onmessage = (msg) => {
          try {
            handle(JSON.parse(String(msg.data)) as RealtimeEvent);
          } catch {
            // Ignore malformed frames.
          }
        };
        socket.onclose = () => {
          setConnected(false);
          schedule();
        };
        socket.onerror = () => socket?.close();
      } catch {
        schedule();
      }
    }

    void connect();
    return () => {
      stopped = true;
      if (timer) {
        clearTimeout(timer);
      }
      socket?.close();
    };
  }, [qc]);

  const value: RealtimeState = {
    connected,
    incomingMatch,
    dismissIncomingMatch: () => setIncomingMatch(null),
    removedMatchIds
  };
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeState {
  return useContext(RealtimeContext);
}
