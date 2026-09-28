import type { QueryClient } from "@tanstack/react-query";
import { markConversationRead } from "../api/matches";
import { queryKeys } from "../api/queryKeys";
import type { RealtimeEvent } from "../api/types";
import {
  addMatch,
  addMessage,
  addNotification,
  applyMessageToMatches,
  clearUnread,
  hasConversation,
  markOutgoingRead,
  removeMatch,
  type MatchPages,
  type MessagePages,
  type NotificationPages
} from "../domain/realtimeCache";

export type RealtimeContext = {
  myUserId: string;
  openConversationId: string | null;
  notify: (message: string) => void;
};

/** Applies one server event to the react-query caches. */
export function applyRealtimeEvent(queryClient: QueryClient, event: RealtimeEvent, context: RealtimeContext): void {
  const { myUserId, openConversationId, notify } = context;
  switch (event.type) {
    case "message.new": {
      const message = event.data;
      const matches = queryClient.getQueryData<MatchPages>(queryKeys.matches);
      if (matches && !hasConversation(matches, message.conversationId)) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      }
      queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) =>
        applyMessageToMatches(data, message, myUserId, openConversationId)
      );
      queryClient.setQueryData<MessagePages>(queryKeys.messages(message.conversationId), (data) =>
        addMessage(data, message)
      );
      if (message.senderId !== myUserId && openConversationId === message.conversationId) {
        queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) => clearUnread(data, message.conversationId));
        void markConversationRead(message.conversationId).catch(() => undefined);
      }
      return;
    }
    case "messages.read": {
      const { conversationId, readerId } = event.data;
      if (readerId === myUserId) {
        queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) => clearUnread(data, conversationId));
      } else {
        queryClient.setQueryData<MessagePages>(queryKeys.messages(conversationId), (data) =>
          markOutgoingRead(data, myUserId, new Date().toISOString())
        );
      }
      return;
    }
    case "match.new": {
      const known = hasConversation(queryClient.getQueryData<MatchPages>(queryKeys.matches), event.data.conversationId);
      queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) => addMatch(data, event.data));
      if (!known) {
        notify(`It's a match with ${event.data.user.firstName}!`);
      }
      return;
    }
    case "match.removed":
      queryClient.setQueryData<MatchPages>(queryKeys.matches, (data) => removeMatch(data, event.data.matchId));
      queryClient.removeQueries({ queryKey: queryKeys.messages(event.data.conversationId) });
      return;
    case "notification.new":
      queryClient.setQueryData<NotificationPages>(queryKeys.notifications, (data) => addNotification(data, event.data));
      return;
  }
}
