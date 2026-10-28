import type { InfiniteData } from "@tanstack/react-query";
import type {
  AppNotification,
  Conversation,
  ConversationsPage,
  Message,
  MessagesPage,
  NotificationsPage
} from "../api/types";

export type ConversationsCache = InfiniteData<ConversationsPage, string | null>;
export type MessagesCache = InfiniteData<MessagesPage, string | null>;
export type NotificationsCache = InfiniteData<NotificationsPage, string | null>;

export type RealtimeEvent =
  | { type: "message"; data: Message }
  | { type: "match"; data: Conversation }
  | { type: "read"; data: { conversationId: string; readerId: string; readAt: string } }
  | { type: "notification"; data: AppNotification };

export function parseRealtimeEvent(raw: string): RealtimeEvent | null {
  try {
    const parsed = JSON.parse(raw) as { type?: string; data?: unknown };
    if (
      (parsed.type === "message" || parsed.type === "match" || parsed.type === "read" || parsed.type === "notification") &&
      parsed.data &&
      typeof parsed.data === "object"
    ) {
      return parsed as RealtimeEvent;
    }
  } catch {
    // ignore malformed frames
  }
  return null;
}

export function upsertMessage(cache: MessagesCache | undefined, message: Message): MessagesCache | undefined {
  if (!cache) return cache;
  const exists = cache.pages.some((page) => page.messages.some((m) => m.id === message.id));
  if (exists) {
    return {
      ...cache,
      pages: cache.pages.map((page) => ({
        ...page,
        messages: page.messages.map((m) => (m.id === message.id ? message : m))
      }))
    };
  }
  const [first, ...rest] = cache.pages;
  if (!first) return cache;
  return { ...cache, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

/** The reader has seen everything: mark messages from the other participant as read. */
export function applyReadReceipt(
  cache: MessagesCache | undefined,
  readerId: string,
  readAt: string
): MessagesCache | undefined {
  if (!cache) return cache;
  return {
    ...cache,
    pages: cache.pages.map((page) => ({
      ...page,
      messages: page.messages.map((m) => (m.senderId !== readerId && !m.readAt ? { ...m, readAt } : m))
    }))
  };
}

export type ConversationUpdate = { cache: ConversationsCache | undefined; found: boolean };

export function applyIncomingMessage(
  cache: ConversationsCache | undefined,
  message: Message,
  myUserId: string,
  activeConversationId: string | null
): ConversationUpdate {
  if (!cache) return { cache, found: false };
  let target: Conversation | null = null;
  const pages = cache.pages.map((page) => ({
    ...page,
    conversations: page.conversations.filter((c) => {
      if (c.id === message.conversationId) {
        target = c;
        return false;
      }
      return true;
    })
  }));
  const found = target as Conversation | null;
  if (!found) return { cache, found: false };
  const incoming = message.senderId !== myUserId;
  const unread = incoming && activeConversationId !== found.id ? found.unreadCount + 1 : found.unreadCount;
  const updated: Conversation = { ...found, lastMessage: message, unreadCount: unread };
  const [first, ...rest] = pages;
  if (!first) return { cache, found: false };
  return {
    cache: { ...cache, pages: [{ ...first, conversations: [updated, ...first.conversations] }, ...rest] },
    found: true
  };
}

export function prependConversation(cache: ConversationsCache | undefined, conv: Conversation): ConversationsCache | undefined {
  if (!cache) return cache;
  const exists = cache.pages.some((p) => p.conversations.some((c) => c.id === conv.id));
  if (exists) return cache;
  const [first, ...rest] = cache.pages;
  if (!first) return cache;
  return { ...cache, pages: [{ ...first, conversations: [conv, ...first.conversations] }, ...rest] };
}

export function clearUnread(cache: ConversationsCache | undefined, conversationId: string): ConversationsCache | undefined {
  if (!cache) return cache;
  return {
    ...cache,
    pages: cache.pages.map((page) => ({
      ...page,
      conversations: page.conversations.map((c) => (c.id === conversationId ? { ...c, unreadCount: 0 } : c))
    }))
  };
}

export function removeConversation(cache: ConversationsCache | undefined, conversationId: string): ConversationsCache | undefined {
  if (!cache) return cache;
  return {
    ...cache,
    pages: cache.pages.map((page) => ({
      ...page,
      conversations: page.conversations.filter((c) => c.id !== conversationId)
    }))
  };
}

export function prependNotification(cache: NotificationsCache | undefined, n: AppNotification): NotificationsCache | undefined {
  if (!cache) return cache;
  if (cache.pages.some((p) => p.notifications.some((x) => x.id === n.id))) return cache;
  const [first, ...rest] = cache.pages;
  if (!first) return cache;
  return {
    ...cache,
    pages: [
      { ...first, notifications: [n, ...first.notifications], unreadCount: first.unreadCount + (n.readAt ? 0 : 1) },
      ...rest
    ]
  };
}

export function markNotificationsRead(cache: NotificationsCache | undefined, id: string | "all"): NotificationsCache | undefined {
  if (!cache) return cache;
  const now = new Date().toISOString();
  let cleared = 0;
  const pages = cache.pages.map((page) => ({
    ...page,
    notifications: page.notifications.map((n) => {
      if (!n.readAt && (id === "all" || n.id === id)) {
        cleared += 1;
        return { ...n, readAt: now };
      }
      return n;
    })
  }));
  return {
    ...cache,
    pages: pages.map((page) => ({
      ...page,
      unreadCount: id === "all" ? 0 : Math.max(0, page.unreadCount - cleared)
    }))
  };
}

export function totalUnread(cache: ConversationsCache | undefined): number {
  return cache?.pages.reduce((sum, p) => sum + p.conversations.reduce((s, c) => s + c.unreadCount, 0), 0) ?? 0;
}
