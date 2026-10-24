import React, { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL, getFreshAccessToken } from "../api/client";
import type { RealtimeEvent } from "../api/types";
import { useToast } from "../ui/Toast";
import { useAuth } from "../auth/AuthProvider";
import { backoffDelay } from "./backoff";
import {
  insertMessage,
  invalidateChat,
  markMessagesRead,
  queryKeys,
  type MessagePages,
} from "./cache";

function wsUrl(): string {
  return `${API_BASE_URL.replace(/^http/, "ws")}/ws`;
}

/**
 * Keeps one authenticated WebSocket open while signed in and applies server events to the
 * React Query cache. Reconnects with backoff and resynchronizes everything after a gap.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const { status } = useAuth();
  const queryClient = useQueryClient();
  const toast = useToast();
  const toastRef = useRef(toast);
  toastRef.current = toast;

  useEffect(() => {
    if (status !== "signedIn") return;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let attempt = 0;
    let stopped = false;

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "ready":
          if (attempt > 0) void queryClient.invalidateQueries(); // resync after a reconnect
          attempt = 0;
          break;
        case "message.new":
          queryClient.setQueryData<MessagePages>(
            queryKeys.messages(event.data.conversationId),
            (d) => insertMessage(d, event.data),
          );
          invalidateChat(queryClient);
          break;
        case "messages.read":
          queryClient.setQueryData<MessagePages>(
            queryKeys.messages(event.data.conversationId),
            (d) => markMessagesRead(d, event.data.readerId, event.data.readAt),
          );
          break;
        case "match.new":
          invalidateChat(queryClient);
          toastRef.current.show({
            message: `Nouveau match avec ${event.data.user.firstName} !`,
            kind: "success",
          });
          break;
        case "match.removed":
          invalidateChat(queryClient);
          void queryClient.invalidateQueries({
            queryKey: queryKeys.messages(event.data.conversationId),
          });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          void queryClient.invalidateQueries({ queryKey: queryKeys.notificationsUnread });
          break;
      }
    };

    const connect = async () => {
      const token = await getFreshAccessToken();
      if (stopped) return;
      if (!token) {
        schedule();
        return;
      }
      const ws = new WebSocket(wsUrl());
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (e) => {
        try {
          handle(JSON.parse(String(e.data)) as RealtimeEvent);
        } catch {
          // ignore malformed frames
        }
      };
      ws.onclose = () => {
        if (!stopped) schedule();
      };
      ws.onerror = () => ws.close();
    };

    const schedule = () => {
      timer = setTimeout(() => void connect(), backoffDelay(attempt++));
    };

    void connect();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      socket?.close();
    };
  }, [status, queryClient]);

  return <>{children}</>;
}
