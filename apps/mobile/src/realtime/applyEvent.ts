import type { QueryClient } from "@tanstack/react-query";
import { queryKeys, type MatchItem, type RealtimeEvent } from "../api";
import {
  appendMessage,
  applyMessageToConversations,
  applyNotification,
  applyReadToConversations,
  markMessagesRead,
  prependMatch,
  removeConversation,
  removeMatchByConversation,
  type NotificationPages,
  type Paged
} from "./cache";
import type { ConversationItem, Message } from "../api";

export type EventContext = {
  currentUserId: string;
  activeConversationId: string | null;
  onMatch: (match: MatchItem) => void;
  /** Called when a message arrives in the thread that is on screen. */
  markRead: (conversationId: string) => void;
};

/** Applies one socket event to the React Query caches. Pure with respect to everything but `queryClient`. */
export function applyRealtimeEvent(queryClient: QueryClient, event: RealtimeEvent, context: EventContext): void {
  switch (event.type) {
    case "message": {
      const message = event.data;
      const isActive = context.activeConversationId === message.conversationId;
      const incoming = message.senderId !== context.currentUserId;

      queryClient.setQueryData<Paged<Message>>(queryKeys.messages(message.conversationId), (data) =>
        appendMessage(data, message)
      );

      let found = false;
      queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) => {
        const result = applyMessageToConversations(data, message, { currentUserId: context.currentUserId, isActive });
        found = result.found;
        return result.data;
      });
      if (!found) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      }

      if (incoming && isActive) {
        context.markRead(message.conversationId);
      } else if (incoming) {
        queryClient.setQueryData<number>(queryKeys.unreadCount, (count) => (count === undefined ? count : count + 1));
      }
      return;
    }
    case "read": {
      const receipt = event.data;
      queryClient.setQueryData<Paged<Message>>(queryKeys.messages(receipt.conversationId), (data) =>
        markMessagesRead(data, receipt)
      );
      queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) =>
        applyReadToConversations(data, receipt, context.currentUserId)
      );
      if (receipt.readerId === context.currentUserId) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.unreadCount });
      }
      return;
    }
    case "match": {
      queryClient.setQueryData<Paged<MatchItem>>(queryKeys.matches, (data) => prependMatch(data, event.data));
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      context.onMatch(event.data);
      return;
    }
    case "conversation_removed": {
      const { conversationId } = event.data;
      queryClient.setQueryData<Paged<ConversationItem>>(queryKeys.conversations, (data) =>
        removeConversation(data, conversationId)
      );
      queryClient.setQueryData<Paged<MatchItem>>(queryKeys.matches, (data) =>
        removeMatchByConversation(data, conversationId)
      );
      queryClient.removeQueries({ queryKey: queryKeys.messages(conversationId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadCount });
      return;
    }
    case "notification": {
      queryClient.setQueryData<NotificationPages>(queryKeys.notifications, (data) =>
        applyNotification(data, event.data)
      );
      return;
    }
  }
}

/** After a reconnection REST is the source of truth: drop everything that may have gone stale. */
export function resyncCaches(queryClient: QueryClient): void {
  void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
  void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
  void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  void queryClient.invalidateQueries({ queryKey: queryKeys.unreadCount });
  void queryClient.invalidateQueries({ queryKey: ["messages"] });
}
