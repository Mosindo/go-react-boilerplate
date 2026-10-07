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
import { AppState, type AppStateStatus } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { fetchRealtimeTicket } from "../api/chat";
import { useAuth } from "../hooks/useAuth";
import { useConversationsQuery, useNotificationsQuery } from "../features/hooks";
import { queryKeys } from "../features/queryKeys";
import type {
  Conversation,
  Match,
  MessagesPage,
  NotificationsPayload,
  RealtimeEvent,
  RealtimeEventData,
  RealtimeEventType
} from "../features/types";
import {
  applyMatchToMatches,
  applyMessageToConversations,
  applyMessageToPages,
  applyNotification,
  applyReadToConversations,
  buildSocketUrl,
  parseRealtimeEvent,
  reconnectDelay,
  totalUnread
} from "./events";

export type RealtimeStatus = "idle" | "connecting" | "open" | "closed";

type Handler<T extends RealtimeEventType> = (data: RealtimeEventData<T>) => void;

type RealtimeContextValue = {
  status: RealtimeStatus;
  subscribe: <T extends RealtimeEventType>(type: T, handler: Handler<T>) => () => void;
  /** Conversation currently on screen; its incoming messages do not raise the unread badge. */
  setActiveConversation: (conversationId: string | null) => void;
};

const PING_INTERVAL_MS = 25_000;

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

type AnyHandler = (data: never) => void;

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated, user } = useAuth();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<RealtimeStatus>("idle");

  const handlersRef = useRef<Map<RealtimeEventType, Set<AnyHandler>>>(new Map());
  const activeConversationRef = useRef<string | null>(null);
  const myId = user?.id ?? null;

  const subscribe = useCallback(<T extends RealtimeEventType>(type: T, handler: Handler<T>) => {
    const set = handlersRef.current.get(type) ?? new Set<AnyHandler>();
    set.add(handler as AnyHandler);
    handlersRef.current.set(type, set);
    return () => {
      set.delete(handler as AnyHandler);
    };
  }, []);

  const setActiveConversation = useCallback((conversationId: string | null) => {
    activeConversationRef.current = conversationId;
  }, []);

  useEffect(() => {
    if (!isAuthenticated || !myId) {
      setStatus("idle");
      return undefined;
    }

    let disposed = false;
    let socket: WebSocket | null = null;
    let pingTimer: ReturnType<typeof setInterval> | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let hasConnectedBefore = false;
    let connecting = false;

    const emit = (event: RealtimeEvent) => {
      const set = handlersRef.current.get(event.type);
      if (!set) {
        return;
      }
      for (const handler of Array.from(set)) {
        (handler as (data: unknown) => void)(event.data);
      }
    };

    const refetchAll = () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      void queryClient.invalidateQueries({ queryKey: ["messages"] });
    };

    const applyToCaches = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message": {
          const message = event.data;
          let known = true;
          queryClient.setQueryData<Conversation[]>(queryKeys.conversations, (current) => {
            if (!current) {
              known = false;
              return current;
            }
            const result = applyMessageToConversations(current, message, myId, activeConversationRef.current);
            known = result.known;
            return result.conversations;
          });
          if (!known) {
            void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          }
          queryClient.setQueryData<InfiniteData<MessagesPage>>(
            queryKeys.messages(message.conversationId),
            (current) => (current ? applyMessageToPages(current, message) : current)
          );
          break;
        }
        case "read":
          queryClient.setQueryData<Conversation[]>(queryKeys.conversations, (current) =>
            current ? applyReadToConversations(current, event.data, myId) : current
          );
          break;
        case "match": {
          queryClient.setQueryData<Match[]>(queryKeys.matches, (current) =>
            current ? applyMatchToMatches(current, event.data) : current
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        }
        case "notification":
          queryClient.setQueryData<NotificationsPayload>(queryKeys.notifications, (current) =>
            current ? applyNotification(current, event.data) : current
          );
          break;
        default:
          break;
      }
    };

    const stopPing = () => {
      if (pingTimer) {
        clearInterval(pingTimer);
        pingTimer = null;
      }
    };

    const closeSocket = () => {
      stopPing();
      const current = socket;
      socket = null;
      if (current) {
        current.onopen = null;
        current.onmessage = null;
        current.onerror = null;
        current.onclose = null;
        try {
          current.close();
        } catch {
          // already closed
        }
      }
    };

    const scheduleReconnect = () => {
      if (disposed || retryTimer) {
        return;
      }
      const delay = reconnectDelay(attempt, Math.random());
      attempt += 1;
      retryTimer = setTimeout(() => {
        retryTimer = null;
        void connect();
      }, delay);
    };

    const connect = async () => {
      if (disposed || connecting || socket) {
        return;
      }
      connecting = true;
      setStatus("connecting");
      try {
        const { ticket } = await fetchRealtimeTicket();
        if (disposed) {
          return;
        }
        const ws = new WebSocket(buildSocketUrl(API_BASE_URL, ticket));
        socket = ws;
        ws.onopen = () => {
          attempt = 0;
          setStatus("open");
          stopPing();
          pingTimer = setInterval(() => {
            if (ws.readyState === 1) {
              ws.send(JSON.stringify({ type: "ping" }));
            }
          }, PING_INTERVAL_MS);
          if (hasConnectedBefore) {
            refetchAll();
          }
          hasConnectedBefore = true;
        };
        ws.onmessage = (frame: { data: unknown }) => {
          if (typeof frame.data !== "string") {
            return;
          }
          const event = parseRealtimeEvent(frame.data);
          if (!event) {
            return;
          }
          applyToCaches(event);
          emit(event);
        };
        ws.onerror = () => {
          // onclose follows and drives the reconnect.
        };
        ws.onclose = () => {
          if (socket === ws) {
            socket = null;
          }
          stopPing();
          if (!disposed) {
            setStatus("closed");
            scheduleReconnect();
          }
        };
      } catch {
        setStatus("closed");
        scheduleReconnect();
      } finally {
        connecting = false;
      }
    };

    const onAppState = (next: AppStateStatus) => {
      if (disposed) {
        return;
      }
      if (next === "active") {
        if (!socket && !connecting) {
          if (retryTimer) {
            clearTimeout(retryTimer);
            retryTimer = null;
          }
          attempt = 0;
          void connect();
        }
      } else if (next === "background") {
        if (retryTimer) {
          clearTimeout(retryTimer);
          retryTimer = null;
        }
        closeSocket();
        setStatus("closed");
      }
    };

    const subscription = AppState.addEventListener("change", onAppState);
    void connect();

    return () => {
      disposed = true;
      subscription.remove();
      if (retryTimer) {
        clearTimeout(retryTimer);
      }
      closeSocket();
      setStatus("idle");
    };
  }, [isAuthenticated, myId, queryClient]);

  const value = useMemo<RealtimeContextValue>(
    () => ({ status, subscribe, setActiveConversation }),
    [status, subscribe, setActiveConversation]
  );

  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  const ctx = useContext(RealtimeContext);
  if (!ctx) {
    throw new Error("useRealtime must be used inside <RealtimeProvider>");
  }
  return ctx;
}

/** Subscribe to one event type for the lifetime of the component. */
export function useRealtimeEvent<T extends RealtimeEventType>(type: T, handler: Handler<T>): void {
  const { subscribe } = useRealtime();
  const handlerRef = useRef(handler);
  handlerRef.current = handler;
  useEffect(() => subscribe(type, (data) => handlerRef.current(data)), [subscribe, type]);
}

export function useUnreadMessagesCount(): number {
  const { isAuthenticated } = useAuth();
  const query = useConversationsQuery();
  return isAuthenticated ? totalUnread(query.data) : 0;
}

export function useUnreadNotificationsCount(): number {
  const { isAuthenticated } = useAuth();
  const query = useNotificationsQuery();
  return isAuthenticated ? (query.data?.unreadCount ?? 0) : 0;
}
