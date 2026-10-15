import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { endpoints } from "../api/endpoints";
import { markConversationRead, requestSocketTicket } from "../api/platform";
import type { MatchInfo, RealtimeEvent } from "../api/types";
import { appendMessage, queryKeys, setPeerLastRead } from "./queries";
import { useAuth } from "./useAuth";

type RealtimeContextValue = {
  /** The conversation currently on screen: its new messages are marked read immediately. */
  setActiveConversation: (conversationId: string | null) => void;
  /** Latest match announced by the server, to be celebrated once. */
  incomingMatch: MatchInfo | null;
  dismissMatch: () => void;
  /** Records a match the user already saw (from their own swipe) so it is not announced twice. */
  markMatchCelebrated: (matchId: string) => void;
  isMatchCelebrated: (matchId: string) => boolean;
  connected: boolean;
};

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

const MAX_BACKOFF_MS = 30_000;

function socketUrl(ticket: string): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}${endpoints.realtime.socket}?ticket=${encodeURIComponent(ticket)}`;
}

function parseEvent(raw: unknown): RealtimeEvent | null {
  if (typeof raw !== "string") {
    return null;
  }
  try {
    const event = JSON.parse(raw) as { type?: unknown; data?: unknown };
    return typeof event.type === "string" ? (event as RealtimeEvent) : null;
  } catch {
    return null;
  }
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status, user } = useAuth();
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [incomingMatch, setIncomingMatch] = useState<MatchInfo | null>(null);
  const activeConversation = useRef<string | null>(null);

  const setActiveConversation = useCallback((conversationId: string | null) => {
    activeConversation.current = conversationId;
  }, []);
  const dismissMatch = useCallback(() => setIncomingMatch(null), []);
  const celebrated = useRef(new Set<string>());
  const markMatchCelebrated = useCallback((matchId: string) => {
    celebrated.current.add(matchId);
  }, []);
  const isMatchCelebrated = useCallback((matchId: string) => celebrated.current.has(matchId), []);

  const userId = user?.id;

  useEffect(() => {
    if (status !== "signedIn" || !userId) {
      return;
    }

    let closed = false;
    let socket: WebSocket | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;

    const resync = () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      const open = activeConversation.current;
      if (open) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.messages(open) });
      }
    };

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.new": {
          appendMessage(queryClient, event.data);
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
          if (event.data.senderId !== userId && activeConversation.current === event.data.conversationId) {
            void markConversationRead(event.data.conversationId).catch(() => undefined);
          } else if (event.data.senderId !== userId) {
            void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          }
          break;
        }
        case "message.read":
          setPeerLastRead(queryClient, event.data.conversationId, event.data.readAt);
          break;
        case "match.new":
          setIncomingMatch(event.data);
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        case "match.removed":
          queryClient.removeQueries({ queryKey: queryKeys.messages(event.data.conversationId) });
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
      }
    };

    const connect = async () => {
      if (closed) {
        return;
      }
      try {
        const ticket = await requestSocketTicket();
        if (closed) {
          return;
        }
        socket = new WebSocket(socketUrl(ticket));
        socket.onopen = () => {
          attempt = 0;
          setConnected(true);
          resync();
        };
        socket.onmessage = (message) => {
          const event = parseEvent(message.data);
          if (event) {
            handle(event);
          }
        };
        socket.onclose = () => {
          setConnected(false);
          scheduleReconnect();
        };
        socket.onerror = () => {
          socket?.close();
        };
      } catch {
        scheduleReconnect();
      }
    };

    const scheduleReconnect = () => {
      if (closed || retryTimer) {
        return;
      }
      const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt);
      attempt += 1;
      retryTimer = setTimeout(() => {
        retryTimer = null;
        void connect();
      }, delay);
    };

    void connect();

    // Mobile OSes suspend sockets in the background: reconnect and resync on return.
    const appStateSubscription = AppState.addEventListener("change", (next) => {
      if (next === "active" && (!socket || socket.readyState > WebSocket.OPEN)) {
        attempt = 0;
        if (retryTimer) {
          clearTimeout(retryTimer);
          retryTimer = null;
        }
        void connect();
      }
    });

    return () => {
      closed = true;
      appStateSubscription.remove();
      if (retryTimer) {
        clearTimeout(retryTimer);
      }
      if (socket) {
        socket.onclose = null;
        socket.close();
      }
      setConnected(false);
    };
  }, [queryClient, status, userId]);

  const value = useMemo(
    () => ({ setActiveConversation, incomingMatch, dismissMatch, markMatchCelebrated, isMatchCelebrated, connected }),
    [connected, dismissMatch, incomingMatch, isMatchCelebrated, markMatchCelebrated, setActiveConversation]
  );

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  const context = useContext(RealtimeContext);
  if (!context) {
    throw new Error("useRealtime must be used within RealtimeProvider");
  }
  return context;
}
