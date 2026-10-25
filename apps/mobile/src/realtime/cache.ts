import type { InfiniteData } from "@tanstack/react-query";
import type {
  AppNotification,
  ConversationItem,
  MatchItem,
  Message,
  NotificationsPage,
  Page,
  ReadReceipt
} from "../api";

export type Paged<T> = InfiniteData<Page<T>, string | undefined>;
export type NotificationPages = InfiniteData<NotificationsPage, string | undefined>;

const LOCAL_ID_PREFIX = "local-";

export function createLocalId(sequence: number): string {
  return `${LOCAL_ID_PREFIX}${sequence}`;
}

export function isLocalMessage(message: Message): boolean {
  return message.id.startsWith(LOCAL_ID_PREFIX);
}

function mapItems<T>(data: Paged<T>, update: (items: T[]) => T[]): Paged<T> {
  return { ...data, pages: data.pages.map((page) => ({ ...page, items: update(page.items) })) };
}

function containsMessage(data: Paged<Message>, id: string): boolean {
  return data.pages.some((page) => page.items.some((item) => item.id === id));
}

/** Inserts a message at the head of the newest-first thread; ignores ones already present. */
export function appendMessage(data: Paged<Message> | undefined, message: Message): Paged<Message> | undefined {
  if (!data || data.pages.length === 0 || containsMessage(data, message.id)) {
    return data;
  }
  const [first, ...rest] = data.pages;
  if (!first) {
    return data;
  }
  return { ...data, pages: [{ ...first, items: [message, ...first.items] }, ...rest] };
}

/** Swaps an optimistic message for the server copy (or just drops it when the real one already arrived). */
export function replaceMessage(
  data: Paged<Message> | undefined,
  localId: string,
  message: Message
): Paged<Message> | undefined {
  if (!data) {
    return data;
  }
  if (containsMessage(data, message.id)) {
    return removeMessage(data, localId);
  }
  return mapItems(data, (items) => items.map((item) => (item.id === localId ? message : item)));
}

export function removeMessage(data: Paged<Message> | undefined, messageId: string): Paged<Message> | undefined {
  return data ? mapItems(data, (items) => items.filter((item) => item.id !== messageId)) : data;
}

/** Marks as read every message written by someone other than the reader, up to the receipt time. */
export function markMessagesRead(data: Paged<Message> | undefined, receipt: ReadReceipt): Paged<Message> | undefined {
  if (!data) {
    return data;
  }
  const readAtMs = Date.parse(receipt.readAt);
  return mapItems(data, (items) =>
    items.map((item) =>
      item.senderId !== receipt.readerId && item.readAt === null && Date.parse(item.createdAt) <= readAtMs
        ? { ...item, readAt: receipt.readAt }
        : item
    )
  );
}

/** Marks every incoming message as read locally (after calling the mark-read endpoint). */
export function markIncomingRead(
  data: Paged<Message> | undefined,
  currentUserId: string,
  readAt: string
): Paged<Message> | undefined {
  return data
    ? mapItems(data, (items) =>
        items.map((item) => (item.senderId !== currentUserId && item.readAt === null ? { ...item, readAt } : item))
      )
    : data;
}

export type ConversationUpdate = { data: Paged<ConversationItem> | undefined; found: boolean };

/** Updates preview, ordering and unread count of the conversation a message belongs to. */
export function applyMessageToConversations(
  data: Paged<ConversationItem> | undefined,
  message: Message,
  options: { currentUserId: string; isActive: boolean }
): ConversationUpdate {
  if (!data) {
    return { data, found: false };
  }
  let target: ConversationItem | undefined;
  for (const page of data.pages) {
    target = page.items.find((item) => item.id === message.conversationId) ?? target;
  }
  if (!target) {
    return { data, found: false };
  }
  if (target.lastMessage.id === message.id) {
    return { data, found: true };
  }
  const incoming = message.senderId !== options.currentUserId;
  const updated: ConversationItem = {
    ...target,
    lastMessage: message,
    updatedAt: message.createdAt,
    unreadCount: incoming && !options.isActive ? target.unreadCount + 1 : target.unreadCount
  };
  const without = mapItems(data, (items) => items.filter((item) => item.id !== message.conversationId));
  const [first, ...rest] = without.pages;
  if (!first) {
    return { data, found: true };
  }
  return { data: { ...without, pages: [{ ...first, items: [updated, ...first.items] }, ...rest] }, found: true };
}

export function applyReadToConversations(
  data: Paged<ConversationItem> | undefined,
  receipt: ReadReceipt,
  currentUserId: string
): Paged<ConversationItem> | undefined {
  if (!data) {
    return data;
  }
  return mapItems(data, (items) =>
    items.map((item) => {
      if (item.id !== receipt.conversationId) {
        return item;
      }
      if (receipt.readerId === currentUserId) {
        return { ...item, unreadCount: 0 };
      }
      return item.lastMessage.senderId === currentUserId
        ? { ...item, lastMessage: { ...item.lastMessage, readAt: receipt.readAt } }
        : item;
    })
  );
}

export function clearConversationUnread(
  data: Paged<ConversationItem> | undefined,
  conversationId: string
): Paged<ConversationItem> | undefined {
  return data
    ? mapItems(data, (items) => items.map((item) => (item.id === conversationId ? { ...item, unreadCount: 0 } : item)))
    : data;
}

export function removeConversation(
  data: Paged<ConversationItem> | undefined,
  conversationId: string
): Paged<ConversationItem> | undefined {
  return data ? mapItems(data, (items) => items.filter((item) => item.id !== conversationId)) : data;
}

export function findConversation(
  data: Paged<ConversationItem> | undefined,
  conversationId: string
): ConversationItem | undefined {
  return data?.pages.flatMap((page) => page.items).find((item) => item.id === conversationId);
}

export function prependMatch(data: Paged<MatchItem> | undefined, match: MatchItem): Paged<MatchItem> | undefined {
  if (
    !data ||
    data.pages.length === 0 ||
    data.pages.some((page) => page.items.some((item) => item.matchId === match.matchId))
  ) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return first ? { ...data, pages: [{ ...first, items: [match, ...first.items] }, ...rest] } : data;
}

export function removeMatchByConversation(
  data: Paged<MatchItem> | undefined,
  conversationId: string
): Paged<MatchItem> | undefined {
  return data ? mapItems(data, (items) => items.filter((item) => item.conversationId !== conversationId)) : data;
}

export function removeMatch(data: Paged<MatchItem> | undefined, matchId: string): Paged<MatchItem> | undefined {
  return data ? mapItems(data, (items) => items.filter((item) => item.matchId !== matchId)) : data;
}

export function findMatchByConversation(
  data: Paged<MatchItem> | undefined,
  conversationId: string
): MatchItem | undefined {
  return data?.pages.flatMap((page) => page.items).find((item) => item.conversationId === conversationId);
}

function mapNotificationPages(
  data: NotificationPages,
  update: (items: AppNotification[]) => AppNotification[],
  unreadDelta: (before: number) => number
): NotificationPages {
  return {
    ...data,
    pages: data.pages.map((page, index) => ({
      ...page,
      items: update(page.items),
      unreadCount: index === 0 ? Math.max(0, unreadDelta(page.unreadCount)) : page.unreadCount
    }))
  };
}

/** Message notifications are coalesced server-side: an unread one for the same conversation is replaced. */
export function applyNotification(
  data: NotificationPages | undefined,
  notification: AppNotification
): NotificationPages | undefined {
  if (
    !data ||
    data.pages.length === 0 ||
    data.pages.some((page) => page.items.some((item) => item.id === notification.id))
  ) {
    return data;
  }
  let replacedUnread = false;
  const isCoalesced = (item: AppNotification) =>
    notification.type === "message" &&
    item.type === "message" &&
    item.readAt === null &&
    item.conversationId === notification.conversationId;
  const stripped = mapNotificationPages(
    data,
    (items) =>
      items.filter((item) => {
        if (isCoalesced(item)) {
          replacedUnread = true;
          return false;
        }
        return true;
      }),
    (before) => before
  );
  const [first, ...rest] = stripped.pages;
  if (!first) {
    return data;
  }
  const increment = notification.readAt === null && !replacedUnread ? 1 : 0;
  return {
    ...stripped,
    pages: [{ ...first, items: [notification, ...first.items], unreadCount: first.unreadCount + increment }, ...rest]
  };
}

export function markNotificationRead(
  data: NotificationPages | undefined,
  id: string,
  readAt: string
): NotificationPages | undefined {
  if (!data) {
    return data;
  }
  let changed = false;
  const next = mapNotificationPages(
    data,
    (items) =>
      items.map((item) => {
        if (item.id === id && item.readAt === null) {
          changed = true;
          return { ...item, readAt };
        }
        return item;
      }),
    (before) => before
  );
  if (!changed) {
    return data;
  }
  return mapNotificationPages(
    next,
    (items) => items,
    (before) => before - 1
  );
}

export function markAllNotificationsRead(
  data: NotificationPages | undefined,
  readAt: string
): NotificationPages | undefined {
  return data
    ? mapNotificationPages(
        data,
        (items) => items.map((item) => (item.readAt === null ? { ...item, readAt } : item)),
        () => 0
      )
    : data;
}
