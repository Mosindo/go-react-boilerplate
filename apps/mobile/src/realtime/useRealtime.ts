import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getValidAccessToken } from "../api/client";
import type { Message, RealtimeEvent } from "../api/types";
import { WS_URL } from "../config";
import { keys } from "../features/keys";
import { prependMessage } from "../features/messageCache";

/**
 * Keeps one authenticated WebSocket open while the user is signed in, and turns server
 * events into cache updates. REST remains the source of truth: if an event is missed
 * (dropped socket), queries refetch on the next screen focus or reconnect.
 */
export function useRealtime(userId: string | null, onMatch?: (conversationId: string) => void): void {
  const queryClient = useQueryClient();
  const matchHandler = useRef(onMatch);
  useEffect(() => {
    matchHandler.current = onMatch;
  }, [onMatch]);

  useEffect(() => {
    if (!userId) {
      return;
    }
    let socket: WebSocket | null = null;
    let retry = 0;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const refreshLists = () => {
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
    };

    const handle = (event: RealtimeEvent) => {
      switch (event.type) {
        case "ready":
          retry = 0;
          refreshLists(); // catch up on anything missed while disconnected
          break;
        case "message": {
          const message: Message = event.data;
          prependMessage(queryClient, message);
          refreshLists();
          break;
        }
        case "read":
          void queryClient.invalidateQueries({ queryKey: keys.messages(event.data.conversationId) });
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          break;
        case "match":
          refreshLists();
          void queryClient.invalidateQueries({ queryKey: keys.discover });
          matchHandler.current?.(event.data.conversationId);
          break;
        case "unmatch":
          refreshLists();
          break;
        default:
          break;
      }
    };

    const connect = async () => {
      if (stopped) return;
      const token = await getValidAccessToken();
      if (!token || stopped) return;
      const ws = new WebSocket(WS_URL);
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (e) => {
        try {
          handle(JSON.parse(String(e.data)) as RealtimeEvent);
        } catch {
          // Ignore malformed frames.
        }
      };
      ws.onclose = () => {
        socket = null;
        if (stopped) return;
        const delay = Math.min(30_000, 1000 * 2 ** retry++);
        timer = setTimeout(() => void connect(), delay);
      };
      ws.onerror = () => ws.close();
    };

    void connect();
    return () => {
      stopped = true;
      if (timer) clearTimeout(timer);
      socket?.close();
    };
  }, [userId, queryClient]);
}
