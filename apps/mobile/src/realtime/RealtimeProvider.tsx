import React, { createContext, useContext, useEffect, useRef, useState } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { wsUrl } from "../api/config";
import { getFreshAccessToken } from "../api/http";
import { useAuth } from "../auth/AuthContext";
import type { MatchView, MessagesPage, RealtimeEvent } from "../api/types";

type RealtimeValue = {
  connected: boolean;
  /** A match received while the app is open (shown as a celebratory sheet). */
  incomingMatch: MatchView | null;
  dismissMatch: () => void;
};

const RealtimeContext = createContext<RealtimeValue>({
  connected: false,
  incomingMatch: null,
  dismissMatch: () => undefined,
});

export function useRealtime(): RealtimeValue {
  return useContext(RealtimeContext);
}

/** Applies a server event to the react-query cache. Exported for unit testing. */
export function applyEvent(qc: QueryClient, ev: RealtimeEvent, onMatch?: (m: MatchView) => void): void {
  switch (ev.type) {
    case "message": {
      qc.setQueryData<{ pages: MessagesPage[]; pageParams: unknown[] }>(
        ["messages", ev.data.conversationId],
        (old) => {
          if (!old || old.pages.length === 0) return old;
          const first = old.pages[0];
          if (!first || first.messages.some((m) => m.id === ev.data.id)) return old;
          return {
            ...old,
            pages: [{ ...first, messages: [...first.messages, ev.data] }, ...old.pages.slice(1)],
          };
        },
      );
      void qc.invalidateQueries({ queryKey: ["conversations"] });
      break;
    }
    case "read":
      void qc.invalidateQueries({ queryKey: ["messages", ev.data.conversationId] });
      break;
    case "match":
      onMatch?.(ev.data);
      void qc.invalidateQueries({ queryKey: ["matches"] });
      void qc.invalidateQueries({ queryKey: ["conversations"] });
      void qc.invalidateQueries({ queryKey: ["notifications"] });
      void qc.invalidateQueries({ queryKey: ["likes"] });
      break;
    case "unmatch":
      void qc.invalidateQueries({ queryKey: ["matches"] });
      void qc.invalidateQueries({ queryKey: ["conversations"] });
      break;
    case "notification":
      void qc.invalidateQueries({ queryKey: ["notifications"] });
      void qc.invalidateQueries({ queryKey: ["likes"] });
      break;
    default:
      break;
  }
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const qc = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [incomingMatch, setIncomingMatch] = useState<MatchView | null>(null);
  const attempt = useRef(0);

  useEffect(() => {
    if (status !== "signedIn") return;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    const connect = async () => {
      const token = await getFreshAccessToken();
      if (stopped) return;
      if (!token) {
        schedule();
        return;
      }
      socket = new WebSocket(wsUrl());
      socket.onopen = () => socket?.send(JSON.stringify({ type: "auth", token }));
      socket.onmessage = (e) => {
        try {
          const ev = JSON.parse(String(e.data)) as RealtimeEvent;
          if (ev.type === "ready") {
            attempt.current = 0;
            setConnected(true);
            // Re-sync anything missed while offline.
            void qc.invalidateQueries();
          } else {
            applyEvent(qc, ev, setIncomingMatch);
          }
        } catch {
          // ignore malformed frames
        }
      };
      socket.onclose = () => {
        setConnected(false);
        if (!stopped) schedule();
      };
      socket.onerror = () => socket?.close();
    };

    const schedule = () => {
      const delay = Math.min(30_000, 1000 * 2 ** attempt.current++);
      timer = setTimeout(() => void connect(), delay);
    };

    void connect();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      socket?.close();
      setConnected(false);
    };
  }, [status, qc]);

  return (
    <RealtimeContext.Provider
      value={{ connected, incomingMatch, dismissMatch: () => setIncomingMatch(null) }}
    >
      {children}
    </RealtimeContext.Provider>
  );
}
