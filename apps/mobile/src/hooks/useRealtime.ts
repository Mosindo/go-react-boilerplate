import { useEffect, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { RealtimeClient, type RealtimeEvent } from "../api/realtime";
import type { MessagesPage } from "../api/types";
import { showToast } from "../shared/feedback";
import { isConversationActive } from "./presence";
import { insertMessage, keys } from "./queries";
import { useAuth } from "./useAuth";

type Handlers = { onMatch?: (event: Extract<RealtimeEvent, { type: "match.created" }>) => void };

/** Keeps React Query caches in sync with server events. */
export function useRealtime(handlers: Handlers = {}) {
  const { getAccessToken, refresh, status } = useAuth();
  const client = useQueryClient();
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;

  useEffect(() => {
    if (status !== "signedIn") {
      return;
    }
    const realtime = new RealtimeClient(getAccessToken, refresh);
    const unsubscribe = realtime.subscribe((event) => {
      switch (event.type) {
        case "ready":
          // Catch up on anything missed while disconnected.
          void client.invalidateQueries({ queryKey: keys.matches });
          void client.invalidateQueries({ queryKey: keys.unread });
          break;
        case "message.created":
          client.setQueryData<InfiniteData<MessagesPage, string | undefined>>(keys.messages(event.data.conversationId), (data) =>
            insertMessage(data, event.data)
          );
          void client.invalidateQueries({ queryKey: keys.matches });
          break;
        case "conversation.read":
          client.setQueryData<InfiniteData<MessagesPage, string | undefined>>(keys.messages(event.data.conversationId), (data) => {
            if (!data || data.pages.length === 0 || data.pages[0].otherUserId !== event.data.userId) {
              return data;
            }
            const [first, ...rest] = data.pages;
            return { ...data, pages: [{ ...first, otherLastReadAt: event.data.lastReadAt }, ...rest] };
          });
          void client.invalidateQueries({ queryKey: keys.matches });
          break;
        case "match.created":
          void client.invalidateQueries({ queryKey: keys.matches });
          handlersRef.current.onMatch?.(event);
          break;
        case "match.removed":
          void client.invalidateQueries({ queryKey: keys.matches });
          break;
        case "notification.created":
          void client.invalidateQueries({ queryKey: keys.notifications });
          void client.invalidateQueries({ queryKey: keys.unread });
          if (event.data.type === "message" && !isConversationActive(event.data.data.conversationId)) {
            showToast(event.data.body);
          }
          break;
      }
    });
    realtime.start();
    const appState = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        realtime.poke();
      }
    });
    return () => {
      appState.remove();
      unsubscribe();
      realtime.stop();
    };
  }, [client, getAccessToken, refresh, status]);
}
