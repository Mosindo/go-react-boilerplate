import React, { useEffect, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { freshAccessToken } from "../api/client";
import { realtimeUrl } from "../api/config";
import { queryKeys } from "../api/queryKeys";
import type { ChatMessage, MessagesPage, UserSummary } from "../api/types";
import { useAuth } from "../features/auth/AuthProvider";
import { navigateToConversation } from "../navigation/navigationRef";
import { showToast } from "../ui";
import { getActiveConversation } from "./activeConversation";

type RealtimeEvent =
  | { type: "ready" | "pong" }
  | { type: "message.new"; data: { message: ChatMessage; fromUserId: string; toUserId: string } }
  | { type: "message.read"; data: { userId: string; readAt: string } }
  | { type: "match.new"; data: { id: string; user: UserSummary } }
  | { type: "match.removed"; data: { userId: string } }
  | { type: "notification.new"; data: unknown };

/**
 * Keeps a WebSocket open while signed in and folds server events into the
 * React Query cache. Reconnects with exponential backoff, refreshes the access
 * token when the server closes the socket on expiry, and pauses in background.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status, accessToken, user } = useAuth();
  const queryClient = useQueryClient();
  const tokenRef = useRef(accessToken);
  tokenRef.current = accessToken;

  useEffect(() => {
    if (status !== "signedIn" || !user) {
      return;
    }
    let socket: WebSocket | null = null;
    let stopped = false;
    let attempt = 0;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let pingTimer: ReturnType<typeof setInterval> | null = null;
    const me = user.id;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "message.new": {
          const { message, fromUserId, toUserId } = event.data;
          const otherId = fromUserId === me ? toUserId : fromUserId;
          queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(otherId), (data) => {
            if (!data || data.pages.length === 0) return data;
            const [first, ...rest] = data.pages;
            if (first.messages.some((m) => m.id === message.id)) return data;
            return { ...data, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
          });
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
          if (fromUserId !== me && getActiveConversation() !== fromUserId) {
            showToast("Nouveau message", "info", () => navigateToConversation(fromUserId));
          }
          break;
        }
        case "message.read":
          void queryClient.invalidateQueries({ queryKey: queryKeys.messages(event.data.userId) });
          break;
        case "match.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
          break;
        case "match.removed":
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
          queryClient.removeQueries({ queryKey: queryKeys.messages(event.data.userId) });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        default:
          break;
      }
    };

    const scheduleReconnect = (refresh: boolean) => {
      if (stopped) return;
      const delay = Math.min(30_000, 1000 * 2 ** attempt) + Math.random() * 500;
      attempt += 1;
      retryTimer = setTimeout(() => void connect(refresh), delay);
    };

    const connect = async (refresh = false) => {
      if (stopped) return;
      const token = refresh ? await freshAccessToken() : tokenRef.current;
      if (!token || stopped) return;
      const ws = new WebSocket(realtimeUrl());
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (msg) => {
        try {
          const event = JSON.parse(String(msg.data)) as RealtimeEvent;
          if (event.type === "ready") {
            attempt = 0;
            // Catch up on anything missed while disconnected.
            void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
            void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          }
          handle(event);
        } catch {
          // Ignore malformed frames.
        }
      };
      ws.onclose = (e) => {
        if (pingTimer) clearInterval(pingTimer);
        if (socket === ws) socket = null;
        scheduleReconnect(e.code === 4401);
      };
      ws.onerror = () => ws.close();
      if (pingTimer) clearInterval(pingTimer);
      pingTimer = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: "ping" }));
      }, 25_000);
    };

    void connect();
    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active" && !socket && !stopped) {
        if (retryTimer) clearTimeout(retryTimer);
        attempt = 0;
        void connect();
      }
    });

    return () => {
      stopped = true;
      sub.remove();
      if (retryTimer) clearTimeout(retryTimer);
      if (pingTimer) clearInterval(pingTimer);
      socket?.close();
    };
    // Reconnect only when the signed-in user changes; token refreshes are
    // picked up through tokenRef on the next (re)connection.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status, user?.id, queryClient]);

  return <>{children}</>;
}
