import type { QueryClient } from "@tanstack/react-query";
import type { RealtimeEvent } from "../api/models";
import {
  applyMessageToConversations,
  markConversationLastMessageRead,
  prependNotification,
  putConversationFirst,
  removeConversation
} from "../lib/dating/cache";
import { markMineRead, upsertMessage } from "../lib/dating/messages";
import {
  CONVERSATIONS_KEY,
  NOTIFICATIONS_KEY,
  messagesKey,
  type ConversationsData,
  type MessagesData,
  type NotificationsData
} from "./queries";

type Context = { myUserId: string; activeConversationId: string | null };

/** Fold one realtime event into the react-query caches. Idempotent: replays never duplicate data. */
export function applyRealtimeEvent(qc: QueryClient, event: RealtimeEvent, ctx: Context): void {
  switch (event.type) {
    case "message.new": {
      const message = event.data.message;
      qc.setQueryData<MessagesData>(messagesKey(message.conversationId), (d) =>
        d ? upsertMessage(d, message) : d
      );
      let found = true;
      qc.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) => {
        if (!d) return d;
        const out = applyMessageToConversations(d, message, {
          myUserId: ctx.myUserId,
          isOpen: ctx.activeConversationId === message.conversationId
        });
        found = out.found;
        return out.data;
      });
      if (!found) void qc.invalidateQueries({ queryKey: CONVERSATIONS_KEY });
      return;
    }
    case "conversation.read": {
      const { conversationId, readAt } = event.data;
      qc.setQueryData<MessagesData>(messagesKey(conversationId), (d) =>
        d ? markMineRead(d, ctx.myUserId, readAt) : d
      );
      qc.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
        d ? markConversationLastMessageRead(d, conversationId, ctx.myUserId, readAt) : d
      );
      return;
    }
    case "match.new": {
      qc.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
        d ? putConversationFirst(d, event.data.conversation) : d
      );
      return;
    }
    case "match.removed": {
      const { conversationId } = event.data;
      qc.setQueryData<ConversationsData>(CONVERSATIONS_KEY, (d) =>
        d ? removeConversation(d, conversationId) : d
      );
      qc.removeQueries({ queryKey: messagesKey(conversationId) });
      return;
    }
    case "notification.new": {
      qc.setQueryData<NotificationsData>(NOTIFICATIONS_KEY, (d) =>
        d ? prependNotification(d, event.data.notification) : d
      );
      return;
    }
  }
}
