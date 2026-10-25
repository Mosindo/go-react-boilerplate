import { useEffect } from "react";
import { AppState } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { API_BASE_URL, chatApi, http, queryKeys } from "../api";
import { useCurrentUser } from "../features/auth/AuthProvider";
import { announceMatch } from "../features/matches/matchStore";
import { getActiveConversation } from "./activeConversation";
import { applyRealtimeEvent, resyncCaches } from "./applyEvent";
import { markIncomingRead, clearConversationUnread, type Paged } from "./cache";
import { toWebSocketUrl } from "./protocol";
import { RealtimeService } from "./service";
import type { ConversationItem, Message } from "../api";

/** Keeps one authenticated socket open while the app is in the foreground and feeds the query caches. */
export function useRealtime(): void {
  const queryClient = useQueryClient();
  const { id: currentUserId } = useCurrentUser();

  useEffect(() => {
    const markRead = (conversationId: string) => {
      chatApi
        .markRead(conversationId)
        .then(() => {
          const readAt = new Date().toISOString();
          queryClient.setQueryData<Paged<Message>>(queryKeys.messages(conversationId), (data) =>
            markIncomingRead(data, currentUserId, readAt)
          );
          queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) =>
            clearConversationUnread(data, conversationId)
          );
          void queryClient.invalidateQueries({ queryKey: queryKeys.unreadCount });
        })
        .catch(() => undefined);
    };

    const service = new RealtimeService({
      url: toWebSocketUrl(API_BASE_URL),
      getToken: (forceRefresh) => http.getFreshAccessToken(forceRefresh),
      onEvent: (event) =>
        applyRealtimeEvent(queryClient, event, {
          currentUserId,
          activeConversationId: getActiveConversation(),
          onMatch: announceMatch,
          markRead
        }),
      onResync: () => resyncCaches(queryClient)
    });

    service.start();
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") {
        service.start({ resync: true });
      } else if (state === "background") {
        service.stop();
      }
    });

    return () => {
      subscription.remove();
      service.stop();
    };
  }, [queryClient, currentUserId]);
}
