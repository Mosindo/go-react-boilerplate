import type { InfiniteData } from "@tanstack/react-query";
import type { AppNotification, ChatMessage, MatchSummary, Message, NotificationPage, Page } from "../api/types";

export type MatchPages = InfiniteData<Page<MatchSummary>>;
export type MessagePages = InfiniteData<Page<ChatMessage>>;
export type NotificationPages = InfiniteData<NotificationPage>;

function mapItems<P extends { items: unknown[] }>(
  data: InfiniteData<P>,
  fn: (items: P["items"]) => P["items"]
): InfiniteData<P> {
  return { ...data, pages: data.pages.map((page) => ({ ...page, items: fn(page.items) })) };
}

/** A new message updates the conversation preview, bumps it to the top and counts as unread if incoming and not on screen. */
export function applyMessageToMatches(
  data: MatchPages | undefined,
  message: Message,
  myUserId: string,
  openConversationId: string | null
): MatchPages | undefined {
  if (!data) {
    return data;
  }
  const incoming = message.senderId !== myUserId;
  let target: MatchSummary | undefined;
  const stripped = mapItems(data, (items) =>
    items.filter((match) => {
      if (match.conversationId === message.conversationId) {
        target = match;
        return false;
      }
      return true;
    })
  );
  if (!target) {
    return data;
  }
  const updated: MatchSummary = {
    ...target,
    lastMessage: { body: message.body, senderId: message.senderId, createdAt: message.createdAt },
    unreadCount: incoming && openConversationId !== message.conversationId ? target.unreadCount + 1 : target.unreadCount
  };
  const [first, ...rest] = stripped.pages;
  return { ...stripped, pages: [{ ...first, items: [updated, ...first.items] }, ...rest] };
}

export function hasConversation(data: MatchPages | undefined, conversationId: string): boolean {
  return Boolean(data?.pages.some((page) => page.items.some((match) => match.conversationId === conversationId)));
}

export function clearUnread(data: MatchPages | undefined, conversationId: string): MatchPages | undefined {
  if (!data) {
    return data;
  }
  return mapItems(data, (items) =>
    items.map((match) =>
      match.conversationId === conversationId && match.unreadCount > 0 ? { ...match, unreadCount: 0 } : match
    )
  );
}

export function addMatch(data: MatchPages | undefined, match: MatchSummary): MatchPages | undefined {
  if (!data) {
    return data;
  }
  if (hasConversation(data, match.conversationId)) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, items: [match, ...first.items] }, ...rest] };
}

export function removeMatch(data: MatchPages | undefined, matchId: string): MatchPages | undefined {
  return data ? mapItems(data, (items) => items.filter((match) => match.matchId !== matchId)) : data;
}

export function totalUnread(data: MatchPages | undefined): number {
  return data
    ? data.pages.reduce((sum, page) => sum + page.items.reduce((inner, match) => inner + match.unreadCount, 0), 0)
    : 0;
}

/** Adds a message to the first page (newest first). Deduplicates by id; replaces a pending optimistic copy with the same id. */
export function addMessage(data: MessagePages | undefined, message: ChatMessage): MessagePages | undefined {
  if (!data) {
    return data;
  }
  const exists = data.pages.some((page) => page.items.some((item) => item.id === message.id));
  if (exists) {
    return mapItems(data, (items) => items.map((item) => (item.id === message.id ? message : item)));
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, items: [message, ...first.items] }, ...rest] };
}

/** Swaps an optimistic message for the server copy; if the server copy already arrived via realtime, drops the temporary one. */
export function replaceMessage(
  data: MessagePages | undefined,
  tempId: string,
  saved: Message
): MessagePages | undefined {
  if (!data) {
    return data;
  }
  const alreadyHasSaved = data.pages.some((page) => page.items.some((item) => item.id === saved.id));
  return mapItems(data, (items) =>
    alreadyHasSaved
      ? items.filter((item) => item.id !== tempId)
      : items.map((item) => (item.id === tempId ? saved : item))
  );
}

export function setMessageStatus(
  data: MessagePages | undefined,
  id: string,
  status: ChatMessage["status"]
): MessagePages | undefined {
  return data ? mapItems(data, (items) => items.map((item) => (item.id === id ? { ...item, status } : item))) : data;
}

/** The other participant read the thread: all of my outgoing messages become read. */
export function markOutgoingRead(
  data: MessagePages | undefined,
  myUserId: string,
  readAt: string
): MessagePages | undefined {
  if (!data) {
    return data;
  }
  return mapItems(data, (items) =>
    items.map((item) =>
      item.senderId === myUserId && item.readAt === null && item.status === undefined ? { ...item, readAt } : item
    )
  );
}

export function addNotification(
  data: NotificationPages | undefined,
  notification: AppNotification
): NotificationPages | undefined {
  if (!data) {
    return data;
  }
  if (data.pages.some((page) => page.items.some((item) => item.id === notification.id))) {
    return data;
  }
  const [first, ...rest] = data.pages;
  const unread = notification.readAt === null ? 1 : 0;
  return {
    ...data,
    pages: [{ ...first, items: [notification, ...first.items], unreadCount: first.unreadCount + unread }, ...rest]
  };
}

function withUnreadCount(data: NotificationPages, unreadCount: number): NotificationPages {
  return { ...data, pages: data.pages.map((page) => ({ ...page, unreadCount })) };
}

export function markNotificationReadInCache(
  data: NotificationPages | undefined,
  id: string,
  readAt: string
): NotificationPages | undefined {
  if (!data) {
    return data;
  }
  const wasUnread = data.pages.some((page) => page.items.some((item) => item.id === id && item.readAt === null));
  if (!wasUnread) {
    return data;
  }
  const updated = mapItems(data, (items) => items.map((item) => (item.id === id ? { ...item, readAt } : item)));
  return withUnreadCount(updated, Math.max(0, (data.pages[0]?.unreadCount ?? 1) - 1));
}

export function markAllNotificationsReadInCache(
  data: NotificationPages | undefined,
  readAt: string
): NotificationPages | undefined {
  if (!data) {
    return data;
  }
  return withUnreadCount(
    mapItems(data, (items) => items.map((item) => (item.readAt === null ? { ...item, readAt } : item))),
    0
  );
}
