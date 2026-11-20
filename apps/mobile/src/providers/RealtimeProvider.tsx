import React, { createContext, useContext, useEffect, useState } from "react";
import { realtime } from "../api/realtime";
import type { MessagesPage, RealtimeEvent } from "../api/types";
import { showToast } from "../ui/Toast";
import { matchCelebrations } from "../features/discovery/matchCelebrations";
import { keys, queryClient } from "./queryClient";
import type { InfiniteData } from "@tanstack/react-query";

const RealtimeContext = createContext(false);

function handleEvent(event: RealtimeEvent) {
  switch (event.type) {
    case "message.created": {
      const message = event.data;
      queryClient.setQueryData<InfiniteData<MessagesPage>>(keys.messages(message.conversationId), (data) => {
        if (!data || data.pages.length === 0) {
          return data;
        }
        const [first, ...rest] = data.pages;
        if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) {
          return data;
        }
        return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
      });
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      break;
    }
    case "conversation.read":
      queryClient.setQueryData<InfiniteData<MessagesPage>>(keys.messages(event.data.conversationId), (data) =>
        data ? { ...data, pages: data.pages.map((p) => ({ ...p, otherLastReadAt: event.data.lastReadAt })) } : data
      );
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      break;
    case "match.created":
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      void queryClient.invalidateQueries({ queryKey: keys.likes });
      if (!matchCelebrations.consume(event.data.user.userId)) {
        showToast(`Nouveau match avec ${event.data.user.firstName} !`, "success");
      }
      break;
    case "match.removed":
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      void queryClient.invalidateQueries({ queryKey: keys.likes });
      if (event.data.conversationId) {
        void queryClient.invalidateQueries({ queryKey: keys.conversation(event.data.conversationId) });
      }
      break;
    case "notification.created":
      void queryClient.invalidateQueries({ queryKey: keys.notifications });
      break;
    default:
      break;
  }
}

/** Connects the WebSocket while signed in and folds events into the query cache. */
export function RealtimeProvider({ enabled, children }: { enabled: boolean; children: React.ReactNode }) {
  const [connected, setConnected] = useState(false);
  useEffect(() => {
    if (!enabled) {
      return;
    }
    const offStatus = realtime.onStatus((value) => {
      setConnected(value);
      if (value) {
        // Catch up on anything missed while offline.
        void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
        void queryClient.invalidateQueries({ queryKey: keys.notifications });
      }
    });
    const offEvents = realtime.subscribe(handleEvent);
    realtime.start();
    return () => {
      offStatus();
      offEvents();
      realtime.stop();
    };
  }, [enabled]);
  return <RealtimeContext.Provider value={enabled && connected}>{children}</RealtimeContext.Provider>;
}

/** True when live updates flow; screens poll as a fallback otherwise. */
export function useRealtimeConnected(): boolean {
  return useContext(RealtimeContext);
}
