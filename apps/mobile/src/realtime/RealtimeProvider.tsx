import { useQueryClient } from "@tanstack/react-query";
import { createContext, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { AppState } from "react-native";
import { api } from "../api/client";
import type { MatchPreview, RealtimeEvent } from "../api/types";
import { toWebSocketUrl } from "../config/env";
import { showToast } from "../shared/feedback";
import { applyMatchRemoved, applyMessageNew, applyMessagesRead, queryKeys } from "./cache";
import { RealtimeClient, type RealtimeStatus, type SocketLike } from "./client";

type Listener = (event: RealtimeEvent) => void;

type RealtimeContextValue = {
  status: RealtimeStatus;
  /** Subscribes to raw events; returns the unsubscribe function. */
  subscribe: (listener: Listener) => () => void;
  /** Remembers a match already announced to the user so the realtime push does not announce it twice. */
  markMatchAnnounced: (matchId: string) => void;
};

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<RealtimeStatus>("idle");
  const listeners = useRef(new Set<Listener>());
  const announced = useRef(new Set<string>());

  const markMatchAnnounced = useRef((matchId: string) => {
    announced.current.add(matchId);
  }).current;

  useEffect(() => {
    const handleEvent = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.new":
          applyMessageNew(queryClient, event.data);
          break;
        case "messages.read":
          applyMessagesRead(queryClient, event.data.conversationId, event.data.readerUserId, event.data.readAt);
          break;
        case "match.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          announceMatch(event.data, announced.current);
          break;
        case "match.removed":
          applyMatchRemoved(queryClient);
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        default:
          break;
      }
      listeners.current.forEach((listener) => listener(event));
    };

    const client = new RealtimeClient({
      url: toWebSocketUrl(api.baseUrl),
      getAccessToken: () => api.currentAccessToken(),
      refreshAccessToken: () => api.ensureFreshAccessToken(),
      createSocket: (url) => new WebSocket(url) as unknown as SocketLike,
      onEvent: handleEvent,
      onStatus: setStatus,
      // After a reconnect, events may have been missed: refetch everything that is on screen.
      onConnected: (isReconnect) => {
        if (isReconnect) void queryClient.invalidateQueries();
      }
    });
    client.start();

    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active") client.reconnectNow();
    });
    return () => {
      appState.remove();
      client.stop();
    };
  }, [queryClient]);

  const value = useMemo<RealtimeContextValue>(
    () => ({
      status,
      subscribe: (listener) => {
        listeners.current.add(listener);
        return () => listeners.current.delete(listener);
      },
      markMatchAnnounced
    }),
    [status, markMatchAnnounced]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

function announceMatch(match: MatchPreview, announced: Set<string>): void {
  if (announced.has(match.id)) return;
  announced.add(match.id);
  showToast(`Nouveau match avec ${match.user.firstName} !`, "success");
}

export function useRealtime(): RealtimeContextValue {
  const ctx = useContext(RealtimeContext);
  if (!ctx) throw new Error("useRealtime must be used inside RealtimeProvider");
  return ctx;
}

/** Runs `handler` for every realtime event while the component is mounted. */
export function useRealtimeEvents(handler: Listener): void {
  const { subscribe } = useRealtime();
  const latest = useRef(handler);
  latest.current = handler;
  useEffect(() => subscribe((event) => latest.current(event)), [subscribe]);
}
