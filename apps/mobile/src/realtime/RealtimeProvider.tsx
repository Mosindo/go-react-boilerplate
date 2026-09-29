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
import { useQueryClient } from "@tanstack/react-query";
import { buildWsUrl, fetchWsTicket } from "../api/realtime";
import type { RealtimeEvent } from "../api/models";
import { useAuth } from "../hooks/useAuth";
import { totalUnreadMessages, unreadNotificationCount } from "../lib/dating/cache";
import { POLL_INTERVAL_MS, computeBackoffMs, parseRealtimeEvent } from "../lib/dating/ws";
import { applyRealtimeEvent } from "./applyEvent";
import {
  CONVERSATIONS_KEY,
  NOTIFICATIONS_KEY,
  messagesKey,
  useConversationsQuery,
  useNotificationsQuery
} from "./queries";

export type RealtimeStatus = "idle" | "connecting" | "open" | "closed";
export type RealtimeListener = (event: RealtimeEvent) => void;

type RealtimeContextValue = {
  status: RealtimeStatus;
  /** Tell the provider which conversation is on screen (its unread count must not grow). */
  setActiveConversation: (conversationId: string | null) => void;
  /** Subscribe to events after the caches have been updated. Returns an unsubscribe function. */
  subscribe: (listener: RealtimeListener) => () => void;
};

type Badges = { unreadNotifications: number; unreadMessages: number };

const noopUnsubscribe = () => undefined;

const RealtimeContext = createContext<RealtimeContextValue>({
  status: "idle",
  setActiveConversation: () => undefined,
  subscribe: () => noopUnsubscribe
});

const BadgesContext = createContext<Badges>({
  unreadNotifications: 0,
  unreadMessages: 0
});

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const { accessToken, user } = useAuth();
  const queryClient = useQueryClient();
  const userId = user?.id ?? null;
  const authenticated = Boolean(accessToken && userId);

  const [status, setStatus] = useState<RealtimeStatus>("idle");
  const [foreground, setForeground] = useState<boolean>(AppState.currentState === "active");

  const activeRef = useRef<string | null>(null);
  const listenersRef = useRef(new Set<RealtimeListener>());
  const userIdRef = useRef<string | null>(userId);
  useEffect(() => {
    userIdRef.current = userId;
  }, [userId]);

  const setActiveConversation = useCallback((conversationId: string | null) => {
    activeRef.current = conversationId;
  }, []);

  const subscribe = useCallback((listener: RealtimeListener) => {
    listenersRef.current.add(listener);
    return () => {
      listenersRef.current.delete(listener);
    };
  }, []);

  // The provider owns the two list queries so badges stay correct even when their tabs never mounted.
  const pollInterval = authenticated && foreground && status !== "open" ? POLL_INTERVAL_MS : false;
  const conversations = useConversationsQuery({
    enabled: authenticated,
    refetchInterval: pollInterval
  });
  const notifications = useNotificationsQuery({
    enabled: authenticated,
    refetchInterval: pollInterval
  });

  useEffect(() => {
    if (!accessToken || !userId) return undefined;

    let disposed = false;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let connecting = false;
    let active = AppState.currentState === "active";

    const clearTimer = () => {
      if (timer !== null) {
        clearTimeout(timer);
        timer = null;
      }
    };

    const detach = (ws: WebSocket) => {
      ws.onopen = null;
      ws.onmessage = null;
      ws.onerror = null;
      ws.onclose = null;
    };

    const closeSocket = () => {
      if (!socket) return;
      const ws = socket;
      socket = null;
      detach(ws);
      try {
        ws.close();
      } catch {
        // already closed
      }
    };

    const resync = () => {
      void queryClient.invalidateQueries({ queryKey: CONVERSATIONS_KEY });
      void queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY });
      const open = activeRef.current;
      if (open) void queryClient.invalidateQueries({ queryKey: messagesKey(open) });
    };

    const scheduleReconnect = () => {
      clearTimer();
      if (disposed || !active) return;
      const delay = computeBackoffMs(attempt);
      attempt += 1;
      timer = setTimeout(() => {
        timer = null;
        void connect();
      }, delay);
    };

    const handleFrame = (raw: unknown) => {
      const event = parseRealtimeEvent(raw);
      if (!event) return;
      applyRealtimeEvent(queryClient, event, {
        myUserId: userIdRef.current ?? "",
        activeConversationId: activeRef.current
      });
      for (const listener of Array.from(listenersRef.current)) {
        try {
          listener(event);
        } catch {
          // a faulty listener must not break the stream
        }
      }
    };

    async function connect(): Promise<void> {
      if (disposed || !active || connecting || socket) return;
      connecting = true;
      setStatus("connecting");
      try {
        const ticket = await fetchWsTicket();
        if (disposed || !active) return;
        const ws = new WebSocket(buildWsUrl(ticket));
        socket = ws;
        ws.onopen = () => {
          if (disposed || socket !== ws) return;
          attempt = 0;
          setStatus("open");
          resync();
        };
        ws.onmessage = (e: { data?: unknown }) => {
          if (socket === ws) handleFrame(e.data);
        };
        ws.onerror = () => {
          // onclose follows; reconnect is scheduled there
        };
        ws.onclose = () => {
          if (socket !== ws) return;
          socket = null;
          detach(ws);
          if (disposed) return;
          setStatus("closed");
          scheduleReconnect();
        };
      } catch {
        if (!disposed) {
          setStatus("closed");
          scheduleReconnect();
        }
      } finally {
        connecting = false;
      }
    }

    const onAppState = (next: AppStateStatus) => {
      const nowActive = next === "active";
      if (nowActive === active) return;
      active = nowActive;
      setForeground(nowActive);
      if (!nowActive) {
        clearTimer();
        closeSocket();
        setStatus("closed");
        return;
      }
      attempt = 0;
      resync();
      void connect();
    };

    const subscription = AppState.addEventListener("change", onAppState);
    void connect();

    return () => {
      disposed = true;
      subscription.remove();
      clearTimer();
      closeSocket();
    };
  }, [accessToken, userId, queryClient]);

  // Signed out: drop everything this module cached.
  useEffect(() => {
    if (authenticated) return;
    activeRef.current = null;
    queryClient.removeQueries({ queryKey: CONVERSATIONS_KEY });
    queryClient.removeQueries({ queryKey: NOTIFICATIONS_KEY });
    queryClient.removeQueries({ queryKey: ["messages"] });
  }, [authenticated, queryClient]);

  const contextValue = useMemo<RealtimeContextValue>(
    () => ({ status: authenticated ? status : "idle", setActiveConversation, subscribe }),
    [authenticated, status, setActiveConversation, subscribe]
  );

  const conversationPages = conversations.data?.pages;
  const notificationPages = notifications.data?.pages;
  const unreadMessages = totalUnreadMessages(conversationPages);
  const unreadNotifications = unreadNotificationCount(notificationPages);
  const badges = useMemo<Badges>(
    () => ({ unreadMessages, unreadNotifications }),
    [unreadMessages, unreadNotifications]
  );

  return (
    <RealtimeContext.Provider value={contextValue}>
      <BadgesContext.Provider value={badges}>{children}</BadgesContext.Provider>
    </RealtimeContext.Provider>
  );
}

export function useBadges(): Badges {
  return useContext(BadgesContext);
}

export function useRealtime(): RealtimeContextValue {
  return useContext(RealtimeContext);
}

/** Subscribe to realtime events for the lifetime of the component (listener may change freely). */
export function useRealtimeEvents(listener: RealtimeListener): void {
  const { subscribe } = useRealtime();
  const ref = useRef(listener);
  useEffect(() => {
    ref.current = listener;
  }, [listener]);
  useEffect(() => subscribe((event) => ref.current(event)), [subscribe]);
}
