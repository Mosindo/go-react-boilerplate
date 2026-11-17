import React, { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { AppState } from "react-native";
import { useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { queryKeys } from "../api/queryKeys";
import type { AppNotification, Conversation, Match, Message, MessagesPage } from "../api/types";
import { showToast } from "../ui/Toast";
import { RealtimeClient, type RealtimeEvent } from "./RealtimeClient";
import { insertMessage } from "./cache";

type RealtimeContextValue = {
  connected: boolean;
  subscribe: (listener: (event: RealtimeEvent) => void) => () => void;
};

const RealtimeContext = createContext<RealtimeContextValue | null>(null);

type Props = {
  children: ReactNode;
  onOpenConversation: (conversationId: string) => void;
  activeConversationRef: { current: string | null };
};

/** Keeps React Query caches in sync with server-pushed events. */
export function RealtimeProvider({ children, onOpenConversation, activeConversationRef }: Props) {
  const queryClient = useQueryClient();
  const client = useMemo(() => new RealtimeClient(), []);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    client.start();
    const offStatus = client.onStatus((isConnected) => {
      setConnected(isConnected);
      if (isConnected) {
        // Resync anything missed while offline.
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      }
    });
    const offEvents = client.subscribe((event) => {
      switch (event.type) {
        case "message.created": {
          const message = event.data as Message;
          queryClient.setQueryData<InfiniteData<MessagesPage>>(queryKeys.messages(message.conversationId), (data) =>
            insertMessage(data, message)
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        }
        case "conversation.read": {
          const read = event.data as { conversationId: string; userId: string; readAt: string };
          queryClient.setQueryData<Conversation>(queryKeys.conversation(read.conversationId), (conv) =>
            conv && conv.profile.userId === read.userId ? { ...conv, otherLastReadAt: read.readAt } : conv
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          break;
        }
        case "match.created": {
          const match = event.data as Match;
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          showToast(`Nouveau match avec ${match.profile.firstName} !`, "match", () => onOpenConversation(match.conversationId));
          break;
        }
        case "match.ended": {
          const ended = event.data as { conversationId: string };
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
          queryClient.removeQueries({ queryKey: queryKeys.messages(ended.conversationId) });
          void queryClient.invalidateQueries({ queryKey: queryKeys.conversation(ended.conversationId) });
          break;
        }
        case "notification.created": {
          const notification = event.data as AppNotification;
          void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
          const conversationId = notification.data.conversationId;
          if (notification.type === "message.new" && conversationId && activeConversationRef.current !== conversationId) {
            showToast(notification.body, "info", () => onOpenConversation(conversationId));
          }
          break;
        }
        default:
          break;
      }
    });

    const appState = AppState.addEventListener("change", (status) => {
      if (status === "active") client.start();
    });

    return () => {
      offStatus();
      offEvents();
      appState.remove();
      client.stop();
    };
  }, [client, queryClient, onOpenConversation, activeConversationRef]);

  const value = useMemo(() => ({ connected, subscribe: (l: (e: RealtimeEvent) => void) => client.subscribe(l) }), [client, connected]);
  return <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>;
}

export function useRealtime(): RealtimeContextValue {
  const ctx = useContext(RealtimeContext);
  if (!ctx) throw new Error("useRealtime must be used inside RealtimeProvider");
  return ctx;
}
