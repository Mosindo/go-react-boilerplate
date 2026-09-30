import React, { createContext, useContext, useEffect, useMemo, useState } from "react";
import { type InfiniteData } from "@tanstack/react-query";
import { RealtimeClient, type RealtimeEvent } from "../api/realtime";
import type { Message, MessagesPage } from "../api/types";
import { queryClient, queryKeys } from "../queryClient";

type RealtimeContextValue = {
  connected: boolean;
  subscribe: (listener: (event: RealtimeEvent) => void) => () => void;
};

const RealtimeContext = createContext<RealtimeContextValue>({ connected: false, subscribe: () => () => undefined });

function appendMessage(message: Message) {
  queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(message.conversationId), (data) => {
    if (!data || data.pages.length === 0) {
      return data;
    }
    const [first, ...rest] = data.pages;
    if (first.messages.some((m) => m.id === message.id)) {
      return data;
    }
    return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
  });
}

/** Applies realtime events to the React Query cache. */
function handleEvent(event: RealtimeEvent, userId: string) {
  switch (event.type) {
    case "message.created":
      if (event.data && !event.truncated) {
        appendMessage(event.data as Message);
      } else if (event.conversationId) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.messages(event.conversationId) });
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      break;
    case "conversation.read":
      if (event.conversationId) {
        const { lastReadAt, readerId } = (event.data ?? {}) as { lastReadAt?: string; readerId?: string };
        if (lastReadAt && readerId && readerId !== userId) {
          queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(event.conversationId), (data) =>
            data ? { ...data, pages: data.pages.map((p) => ({ ...p, otherLastReadAt: lastReadAt })) } : data
          );
        }
      }
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      break;
    case "match.created":
    case "match.removed":
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      break;
    case "notification.created":
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      break;
    default:
      break;
  }
}

export function RealtimeProvider({ children, userId }: { children: React.ReactNode; userId: string }) {
  const client = useMemo(() => new RealtimeClient(), []);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const offStatus = client.onStatus((value) => {
      setConnected(value);
      if (value) {
        // Resync anything missed while disconnected.
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      }
    });
    const offEvents = client.subscribe((event) => handleEvent(event, userId));
    client.start();
    return () => {
      offStatus();
      offEvents();
      client.stop();
    };
  }, [client, userId]);

  const value = useMemo(() => ({ connected, subscribe: (l: (e: RealtimeEvent) => void) => client.subscribe(l) }), [client, connected]);
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  return useContext(RealtimeContext);
}
