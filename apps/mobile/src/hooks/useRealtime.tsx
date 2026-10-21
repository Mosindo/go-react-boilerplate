import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import React, { useEffect } from "react";
import { getValidAccessToken, websocketUrl } from "../api/client";
import type { Message, MessagePage, RealtimeEvent } from "../api/types";
import { showToast } from "../shared/feedback";
import { keys } from "./queries";

const MAX_BACKOFF_MS = 30_000;

/**
 * Keeps one authenticated WebSocket open while the user is signed in and folds server events into
 * the React Query cache. Reconnects with exponential backoff; REST remains the source of truth, so
 * a missed event is repaired by the invalidations done on reconnect.
 */
export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();

  useEffect(() => {
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let attempt = 0;
    let stopped = false;

    const addMessage = (msg: Message) => {
      queryClient.setQueryData<InfiniteData<MessagePage>>(keys.messages(msg.conversationId), (old) => {
        if (!old || old.pages.length === 0) return old;
        const [first, ...rest] = old.pages;
        if (!first || old.pages.some((p) => p.items.some((m) => m.id === msg.id))) return old;
        return { ...old, pages: [{ ...first, items: [msg, ...first.items] }, ...rest] };
      });
    };

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "ready":
          attempt = 0;
          // Catch up on anything missed while offline.
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "message.new":
          addMessage(event.data);
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          break;
        case "match.new":
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          showToast(`C'est un match avec ${event.data.user.firstName} !`, "success");
          break;
        case "conversation.read":
          void queryClient.invalidateQueries({ queryKey: keys.messages(event.data.conversationId) });
          break;
        case "conversation.removed":
          queryClient.removeQueries({ queryKey: keys.messages(event.data.conversationId) });
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
        case "notification.new":
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
        default:
          break;
      }
    };

    const connect = async () => {
      if (stopped) return;
      const token = await getValidAccessToken();
      if (!token || stopped) return;
      const ws = new WebSocket(websocketUrl());
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
        if (stopped) return;
        attempt += 1;
        timer = setTimeout(() => void connect(), Math.min(MAX_BACKOFF_MS, 1000 * 2 ** Math.min(attempt, 5)));
      };
      ws.onerror = () => ws.close();
    };

    void connect();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      socket?.close();
    };
  }, [queryClient]);

  return <>{children}</>;
}
