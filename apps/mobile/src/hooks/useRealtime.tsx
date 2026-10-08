import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { wsUrl } from "../api/client";
import { keys } from "../api/keys";
import { getWsTicket } from "../api/platform";
import type { RealtimeEvent } from "../api/types";
import { applyRealtimeEvent, parseEvent, reconnectDelay } from "../realtime/events";
import { useAuth } from "./useAuth";

type MatchEvent = Extract<RealtimeEvent, { type: "match.new" }>["data"];

type RealtimeValue = {
  connected: boolean;
  /** Latest match pushed by the server, for an in-app celebration; cleared with `dismissMatch`. */
  incomingMatch: MatchEvent | null;
  dismissMatch: () => void;
  /** Shows the celebration once per match, whichever of the swipe response or the push arrives first. */
  announceMatch: (match: MatchEvent) => void;
};

const RealtimeContext = createContext<RealtimeValue>({ connected: false, incomingMatch: null, dismissMatch: () => undefined, announceMatch: () => undefined });

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const client = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [incomingMatch, setIncomingMatch] = useState<MatchEvent | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const announced = useRef(new Set<string>());

  const announceMatch = useCallback((match: MatchEvent) => {
    if (announced.current.has(match.matchId)) return;
    announced.current.add(match.matchId);
    setIncomingMatch(match);
  }, []);

  useEffect(() => {
    if (status !== "signedIn") return;
    let stopped = false;
    let attempt = 0;
    let timer: ReturnType<typeof setTimeout> | undefined;

    const refreshEverything = () => {
      for (const key of [keys.conversations, keys.unread, keys.matches, keys.notifications, keys.notificationsUnread]) {
        void client.invalidateQueries({ queryKey: key });
      }
    };

    const connect = async () => {
      try {
        const ticket = await getWsTicket();
        if (stopped) return;
        const socket = new WebSocket(wsUrl(ticket));
        socketRef.current = socket;
        socket.onopen = () => {
          attempt = 0;
          setConnected(true);
          // Anything pushed while we were offline was missed: resync the lists.
          refreshEverything();
        };
        socket.onmessage = (message) => {
          const event = typeof message.data === "string" ? parseEvent(message.data) : null;
          if (!event) return;
          applyRealtimeEvent(client, event);
          if (event.type === "match.new") {
            announceMatch(event.data);
            void client.invalidateQueries({ queryKey: keys.notificationsUnread });
          }
        };
        socket.onclose = () => {
          setConnected(false);
          if (!stopped) timer = setTimeout(connect, reconnectDelay(attempt++));
        };
        socket.onerror = () => socket.close();
      } catch {
        if (!stopped) timer = setTimeout(connect, reconnectDelay(attempt++));
      }
    };

    void connect();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active") refreshEverything();
    });

    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      appState.remove();
      socketRef.current?.close();
      socketRef.current = null;
      setConnected(false);
    };
  }, [status, client, announceMatch]);

  const value = useMemo<RealtimeValue>(
    () => ({ connected, incomingMatch, dismissMatch: () => setIncomingMatch(null), announceMatch }),
    [connected, incomingMatch, announceMatch]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeValue {
  return useContext(RealtimeContext);
}
