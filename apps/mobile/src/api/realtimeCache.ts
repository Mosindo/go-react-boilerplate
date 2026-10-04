import type { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "./queryKeys";
import type { AppNotification, MatchSummary, Message, RealtimeEvent } from "./types";
import { hasItem, mapPages, prependItem, removeItems, replaceItem, type CursorData } from "../lib/pagination";

type MessagesData = CursorData<"messages", Message>;
type ConversationsData = CursorData<"conversations", MatchSummary>;
type MatchesData = CursorData<"matches", MatchSummary>;
type NotificationsData = CursorData<"notifications", AppNotification>;

export type RealtimeContext = {
  /** Current user id, to tell my messages from theirs. */
  myUserId: string;
  /** Conversation currently on screen: its messages are considered read immediately. */
  getActiveConversationId: () => string | null;
};

/** Inserts a message at the top of a conversation's cache unless it is already there. */
export function insertMessage(client: QueryClient, message: Message): void {
  client.setQueryData<MessagesData>(queryKeys.messages(message.conversationId), (data) => {
    if (!data) return data; // not loaded yet: the first fetch will include it
    if (hasItem(data, "messages", message.id)) return data;
    return prependItem(data, "messages", message);
  });
}

function bumpUnread(client: QueryClient, delta: number) {
  client.setQueryData<number>(queryKeys.conversationsUnread, (count) => Math.max(0, (count ?? 0) + delta));
}

/**
 * Applies one server event to the React Query caches. Pure with respect to its inputs
 * (QueryClient + event), so it is unit-tested without any socket.
 */
export function applyRealtimeEvent(client: QueryClient, event: RealtimeEvent, ctx: RealtimeContext): void {
  switch (event.type) {
    case "message.new": {
      const message = event.data;
      const mine = message.senderId === ctx.myUserId;
      const isActive = ctx.getActiveConversationId() === message.conversationId;
      const unreadDelta = !mine && !isActive ? 1 : 0;
      insertMessage(client, message);

      const conversations = client.getQueryData<ConversationsData>(queryKeys.conversations);
      const existing = conversations?.pages
        .flatMap((page) => page.conversations)
        .find((c) => c.conversationId === message.conversationId);
      if (existing) {
        const updated: MatchSummary = {
          ...existing,
          lastMessage: message,
          unreadCount: existing.unreadCount + unreadDelta,
          updatedAt: message.createdAt
        };
        client.setQueryData<ConversationsData>(queryKeys.conversations, (data) =>
          data ? prependItem(data, "conversations", updated) : data
        );
      } else {
        // First message of a match we have no row for (or list not loaded): refetch.
        void client.invalidateQueries({ queryKey: queryKeys.conversations });
      }
      // The match row (used by the "new matches" strip) now has a last message.
      if (existing) {
        client.setQueryData<MatchesData>(queryKeys.matches, (data) =>
          replaceItem(data, "matches", existing.id, (m) => ({ ...m, lastMessage: message }))
        );
      }
      if (unreadDelta) bumpUnread(client, unreadDelta);
      return;
    }
    case "message.read": {
      const { conversationId, readAt } = event.data;
      client.setQueryData<MessagesData>(queryKeys.messages(conversationId), (data) =>
        data
          ? mapPages(data, "messages", (items) =>
              items.map((m) => (m.senderId === ctx.myUserId && !m.readAt ? { ...m, readAt } : m))
            )
          : data
      );
      client.setQueryData<ConversationsData>(queryKeys.conversations, (data) =>
        data
          ? mapPages(data, "conversations", (items) =>
              items.map((c) =>
                c.conversationId === conversationId && c.lastMessage && c.lastMessage.senderId === ctx.myUserId
                  ? { ...c, lastMessage: { ...c.lastMessage, readAt: c.lastMessage.readAt ?? readAt } }
                  : c
              )
            )
          : data
      );
      return;
    }
    case "match.new": {
      client.setQueryData<MatchesData>(queryKeys.matches, (data) => prependItem(data, "matches", event.data));
      return;
    }
    case "match.removed": {
      const { matchId, conversationId } = event.data;
      client.setQueryData<MatchesData>(queryKeys.matches, (data) =>
        removeItems(data, "matches", (m) => m.id === matchId)
      );
      client.setQueryData<ConversationsData>(queryKeys.conversations, (data) =>
        removeItems(data, "conversations", (c) => c.id === matchId || c.conversationId === conversationId)
      );
      client.removeQueries({ queryKey: queryKeys.messages(conversationId) });
      void client.invalidateQueries({ queryKey: queryKeys.conversationsUnread });
      return;
    }
    case "notification.new": {
      const notification = event.data;
      client.setQueryData<NotificationsData>(queryKeys.notifications, (data) =>
        data ? prependItem(data, "notifications", notification) : data
      );
      if (!notification.readAt) {
        client.setQueryData<number>(queryKeys.notificationsUnread, (count) => (count ?? 0) + 1);
      }
      return;
    }
  }
}

/** After a reconnect events may have been missed: refresh everything user-visible. */
export function invalidateAfterReconnect(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: queryKeys.conversationsAll });
  void client.invalidateQueries({ queryKey: queryKeys.matchesAll });
  void client.invalidateQueries({ queryKey: queryKeys.notificationsAll });
  void client.invalidateQueries({ queryKey: queryKeys.messagesAll });
}
