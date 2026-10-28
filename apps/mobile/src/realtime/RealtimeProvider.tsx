import React, { useEffect, useRef } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { WS_URL } from "../api/config";
import { refreshAccessToken } from "../api/client";
import { keys } from "../api/keys";
import { getTokens } from "../store/tokenStore";
import { useAuth } from "../auth/AuthContext";
import {
  applyIncomingMessage,
  applyReadReceipt,
  prependConversation,
  prependNotification,
  clearUnread,
  upsertMessage,
  type ConversationsCache,
  type MessagesCache,
  type NotificationsCache,
  type RealtimeEvent
} from "./cache";
import { RealtimeClient } from "./socket";

/** Conversation currently open on screen, so incoming messages there are not counted as unread. */
let activeConversationId: string | null = null;
export function setActiveConversation(id: string | null): void {
  activeConversationId = id;
}

export function RealtimeProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const userId = user?.id ?? null;
  const clientRef = useRef<RealtimeClient | null>(null);

  useEffect(() => {
    if (!userId) return;

    const handle = (event: RealtimeEvent): void => {
      switch (event.type) {
        case "message": {
          const msg = event.data;
          queryClient.setQueryData<MessagesCache>(keys.messages(msg.conversationId), (c) => upsertMessage(c, msg));
          const current = queryClient.getQueryData<ConversationsCache>(keys.conversations);
          const result = applyIncomingMessage(current, msg, userId, activeConversationId);
          if (result.found) {
            queryClient.setQueryData(keys.conversations, result.cache);
          } else {
            void queryClient.invalidateQueries({ queryKey: keys.conversations });
          }
          break;
        }
        case "match":
          queryClient.setQueryData<ConversationsCache>(keys.conversations, (c) => prependConversation(c, event.data));
          void queryClient.invalidateQueries({ queryKey: keys.conversations });
          break;
        case "read": {
          const { conversationId, readerId, readAt } = event.data;
          queryClient.setQueryData<MessagesCache>(keys.messages(conversationId), (c) =>
            applyReadReceipt(c, readerId, readAt)
          );
          if (readerId === userId) {
            queryClient.setQueryData<ConversationsCache>(keys.conversations, (c) => clearUnread(c, conversationId));
          }
          break;
        }
        case "notification":
          queryClient.setQueryData<NotificationsCache>(keys.notifications, (c) =>
            prependNotification(c, event.data)
          );
          void queryClient.invalidateQueries({ queryKey: keys.notifications });
          break;
      }
    };

    const client = new RealtimeClient({
      url: WS_URL,
      getToken: async () => (await getTokens())?.accessToken ?? null,
      refreshToken: refreshAccessToken,
      onEvent: handle
    });
    clientRef.current = client;
    client.start();

    const sub = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        client.start();
        void queryClient.invalidateQueries({ queryKey: keys.conversations });
        void queryClient.invalidateQueries({ queryKey: keys.notifications });
      } else {
        client.stop();
      }
    });

    return () => {
      sub.remove();
      client.stop();
      clientRef.current = null;
    };
  }, [queryClient, userId]);

  return <>{children}</>;
}
