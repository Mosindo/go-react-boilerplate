import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode
} from "react";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { realtimeApi } from "../api/endpoints";
import type { Conversation, Message } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { insertMessage, type MessagePages } from "../lib/messages";
import { queryKeys } from "../lib/queryKeys";

export type MatchEvent = {
  matchId: string;
  conversationId: string;
  peerId: string;
  peerName: string;
};

type RealtimeValue = {
  connected: boolean;
  pendingMatch: MatchEvent | null;
  showMatch: (m: MatchEvent) => void;
  dismissMatch: () => void;
};

const RealtimeContext = createContext<RealtimeValue | null>(null);

type ServerEvent =
  | { type: "match.new"; data: MatchEvent }
  | { type: "match.ended"; data: { matchId: string; conversationId: string } }
  | { type: "message.new"; data: Message }
  | {
      type: "message.read";
      data: { conversationId: string; readerId: string; upToMessageId: number };
    }
  | { type: "notification.new"; data: unknown };

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const { status, user } = useAuth();
  const qc = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [pendingMatch, setPendingMatch] = useState<MatchEvent | null>(null);
  const userId = user?.id ?? null;
  const userIdRef = useRef(userId);
  userIdRef.current = userId;

  const handle = useCallback(
    (ev: ServerEvent) => {
      switch (ev.type) {
        case "match.new":
          setPendingMatch(ev.data);
          void qc.invalidateQueries({ queryKey: queryKeys.conversations });
          void qc.invalidateQueries({ queryKey: queryKeys.feed });
          break;
        case "match.ended":
          qc.setQueryData<Conversation[]>(queryKeys.conversations, (old) =>
            old?.filter((c) => c.id !== ev.data.conversationId)
          );
          // An open chat refetches, gets a 404 and shows its "conversation ended" state.
          void qc.invalidateQueries({ queryKey: queryKeys.messages(ev.data.conversationId) });
          break;
        case "message.new": {
          qc.setQueryData<MessagePages>(queryKeys.messages(ev.data.conversationId), (old) =>
            insertMessage(old, ev.data)
          );
          qc.setQueryData<Conversation[]>(queryKeys.conversations, (old) => {
            if (!old) return old;
            const idx = old.findIndex((c) => c.id === ev.data.conversationId);
            const found = old[idx];
            if (!found) {
              void qc.invalidateQueries({ queryKey: queryKeys.conversations });
              return old;
            }
            const mine = ev.data.senderId === userIdRef.current;
            const updated: Conversation = {
              ...found,
              lastMessage: {
                id: ev.data.id,
                senderId: ev.data.senderId,
                body: ev.data.body,
                createdAt: ev.data.createdAt
              },
              activityAt: ev.data.createdAt,
              unreadCount: mine ? 0 : found.unreadCount + 1
            };
            return [updated, ...old.filter((_, i) => i !== idx)];
          });
          break;
        }
        case "message.read":
          if (ev.data.readerId !== userIdRef.current) {
            void qc.invalidateQueries({ queryKey: queryKeys.messages(ev.data.conversationId) });
          }
          break;
        case "notification.new":
          void qc.invalidateQueries({ queryKey: queryKeys.unread });
          void qc.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
      }
    },
    [qc]
  );

  useEffect(() => {
    if (status !== "authenticated" || !API_BASE_URL) return;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let attempt = 0;
    let stopped = false;

    const connect = async () => {
      try {
        const ticket = await realtimeApi.ticket();
        if (stopped) return;
        const url = `${API_BASE_URL.replace(/^http/, "ws")}/ws?ticket=${encodeURIComponent(ticket)}`;
        socket = new WebSocket(url);
        socket.onopen = () => {
          attempt = 0;
          setConnected(true);
          // Anything missed while offline is refetched; the DB is the source of truth.
          void qc.invalidateQueries();
        };
        socket.onmessage = (e: MessageEvent<string>) => {
          try {
            handle(JSON.parse(e.data) as ServerEvent);
          } catch {
            /* ignore malformed frame */
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
    };
    const schedule = () => {
      if (stopped) return;
      const delay = Math.min(30_000, 1000 * 2 ** attempt++);
      timer = setTimeout(() => void connect(), delay);
    };

    void connect();
    return () => {
      stopped = true;
      clearTimeout(timer);
      if (socket) {
        socket.onclose = null;
        socket.close();
      }
      setConnected(false);
    };
  }, [status, qc, handle]);

  const value = useMemo<RealtimeValue>(
    () => ({
      connected,
      pendingMatch,
      showMatch: setPendingMatch,
      dismissMatch: () => setPendingMatch(null)
    }),
    [connected, pendingMatch]
  );
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeValue {
  const ctx = useContext(RealtimeContext);
  if (!ctx) throw new Error("useRealtime must be used inside RealtimeProvider");
  return ctx;
}
