import React, { useEffect, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL } from "../api/client";
import { queryKeys } from "../api/queryKeys";
import type { ChatMessage } from "../api/types";
import { toWebSocketUrl } from "../lib/format";
import { applyReadReceipt, mergeMessage } from "../lib/messages";
import { getAccessToken } from "../store/tokenStore";
import { useAuth } from "./useAuth";

type ServerEvent =
  | { type: "ready" }
  | { type: "message.new"; data: ChatMessage }
  | { type: "message.read"; data: { matchId: string; readerId: string } }
  | { type: "match.new" | "match.removed"; data: { matchId: string } }
  | { type: "conversations.changed" }
  | { type: "notification.new"; data: { title: string } };

const MAX_BACKOFF_MS = 30_000;

/** The chat screen registers itself here so incoming messages do not toast while it is on screen. */
export const activeChat: { matchId: string | null } = { matchId: null };

export function RealtimeProvider({ children, onToast }: { children: React.ReactNode; onToast: (message: string) => void }) {
  const { user } = useAuth();
  const client = useQueryClient();
  const userId = user?.id ?? null;
  const toastRef = useRef(onToast);
  toastRef.current = onToast;

  useEffect(() => {
    if (!userId || !API_BASE_URL) {
      return;
    }
    let socket: WebSocket | null = null;
    let stopped = false;
    let attempt = 0;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const handle = (event: ServerEvent) => {
      switch (event.type) {
        case "ready":
          attempt = 0;
          // Anything missed while offline is picked up by refetching the lists.
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        case "message.new": {
          const message = event.data;
          client.setQueryData<ChatMessage[]>(queryKeys.messages(message.matchId), (old) =>
            old ? mergeMessage(old, message) : old
          );
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          if (message.senderId !== userId && activeChat.matchId !== message.matchId) {
            toastRef.current("New message");
          }
          break;
        }
        case "message.read":
          client.setQueryData<ChatMessage[]>(queryKeys.messages(event.data.matchId), (old) =>
            old ? applyReadReceipt(old, event.data.readerId, new Date().toISOString()) : old
          );
          break;
        case "match.new":
          toastRef.current("It's a match! Say hello.");
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
        case "match.removed":
        case "conversations.changed":
          void client.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        case "notification.new":
          void client.invalidateQueries({ queryKey: queryKeys.notifications });
          break;
      }
    };

    const connect = async () => {
      if (stopped) {
        return;
      }
      const token = await getAccessToken();
      if (!token || stopped) {
        return;
      }
      const ws = new WebSocket(toWebSocketUrl(API_BASE_URL));
      socket = ws;
      ws.onopen = () => ws.send(JSON.stringify({ type: "auth", token }));
      ws.onmessage = (raw) => {
        try {
          handle(JSON.parse(String(raw.data)) as ServerEvent);
        } catch {
          // Ignore malformed frames.
        }
      };
      ws.onclose = () => {
        if (stopped || socket !== ws) {
          return;
        }
        const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** attempt);
        attempt += 1;
        timer = setTimeout(() => void connect(), delay);
      };
      ws.onerror = () => ws.close();
    };

    void connect();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active" && (!socket || socket.readyState === WebSocket.CLOSED)) {
        attempt = 0;
        void connect();
      }
    });

    return () => {
      stopped = true;
      appState.remove();
      if (timer) {
        clearTimeout(timer);
      }
      const current = socket;
      socket = null;
      current?.close();
    };
  }, [client, userId]);

  return <>{children}</>;
}
