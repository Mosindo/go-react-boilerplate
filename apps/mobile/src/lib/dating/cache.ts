import type { AppNotification, ConversationSummary, Message } from "../../api/models";

export type ConversationsPage = { conversations: ConversationSummary[] };
export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number };

type WithConversations = { pages: ConversationsPage[] };
type WithNotifications = { pages: NotificationsPage[] };

export function flattenConversations(pages: readonly ConversationsPage[]): ConversationSummary[] {
  const seen = new Set<string>();
  const out: ConversationSummary[] = [];
  for (const page of pages) {
    for (const c of page.conversations) {
      if (seen.has(c.id)) continue;
      seen.add(c.id);
      out.push(c);
    }
  }
  return out;
}

export function findConversation(
  pages: readonly ConversationsPage[],
  id: string
): ConversationSummary | null {
  for (const page of pages) {
    const found = page.conversations.find((c) => c.id === id);
    if (found) return found;
  }
  return null;
}

/** New matches: conversations nobody wrote in yet. */
export function selectNewMatches(conversations: readonly ConversationSummary[]) {
  return conversations.filter((c) => c.lastMessage === null);
}

/** Conversations that have at least one message. */
export function selectActiveConversations(conversations: readonly ConversationSummary[]) {
  return conversations.filter((c) => c.lastMessage !== null);
}

export function totalUnreadMessages(pages: readonly ConversationsPage[] | undefined): number {
  if (!pages) return 0;
  return flattenConversations(pages).reduce((sum, c) => sum + Math.max(0, c.unreadCount), 0);
}

function withoutConversation(pages: ConversationsPage[], id: string): ConversationsPage[] {
  return pages.map((page) =>
    page.conversations.some((c) => c.id === id)
      ? { ...page, conversations: page.conversations.filter((c) => c.id !== id) }
      : page
  );
}

/** Put a conversation at the very top (replacing any existing copy). */
export function putConversationFirst<D extends WithConversations>(
  data: D,
  conversation: ConversationSummary
): D {
  const pages = withoutConversation(data.pages, conversation.id);
  if (pages.length === 0) return { ...data, pages: [{ conversations: [conversation] }] };
  const [first, ...rest] = pages;
  return { ...data, pages: [{ ...first, conversations: [conversation, ...first.conversations] }, ...rest] };
}

export function removeConversation<D extends WithConversations>(data: D, id: string): D {
  if (!findConversation(data.pages, id)) return data;
  return { ...data, pages: withoutConversation(data.pages, id) };
}

export type NewMessageOutcome<D> = { data: D; found: boolean };

/**
 * A message arrived (or was sent). Updates lastMessage/updatedAt, bumps unread when the message is from the
 * other user and the thread is not open, and moves the conversation to the top.
 * `found: false` means the conversation is not in the cache, so the caller should refetch.
 */
export function applyMessageToConversations<D extends WithConversations>(
  data: D,
  message: Message,
  options: { myUserId: string; isOpen: boolean }
): NewMessageOutcome<D> {
  const existing = findConversation(data.pages, message.conversationId);
  if (!existing) return { data, found: false };
  const isDuplicate = existing.lastMessage?.id === message.id;
  const incoming = message.senderId !== options.myUserId;
  const newer =
    !existing.lastMessage || message.createdAt >= existing.lastMessage.createdAt || isDuplicate;
  const updated: ConversationSummary = {
    ...existing,
    lastMessage: newer ? message : existing.lastMessage,
    updatedAt: newer ? message.createdAt : existing.updatedAt,
    unreadCount:
      incoming && !options.isOpen && !isDuplicate ? existing.unreadCount + 1 : existing.unreadCount
  };
  return { data: putConversationFirst(data, updated), found: true };
}

export function setConversationUnread<D extends WithConversations>(
  data: D,
  id: string,
  unreadCount: number
): D {
  const existing = findConversation(data.pages, id);
  if (!existing || existing.unreadCount === unreadCount) return data;
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      conversations: page.conversations.map((c) => (c.id === id ? { ...c, unreadCount } : c))
    }))
  };
}

/** The other user read my messages: flag my last message in the list as read. */
export function markConversationLastMessageRead<D extends WithConversations>(
  data: D,
  id: string,
  myUserId: string,
  readAt: string
): D {
  const existing = findConversation(data.pages, id);
  const last = existing?.lastMessage;
  if (!existing || !last || last.senderId !== myUserId || last.readAt !== null) return data;
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      conversations: page.conversations.map((c) =>
        c.id === id && c.lastMessage ? { ...c, lastMessage: { ...c.lastMessage, readAt } } : c
      )
    }))
  };
}

export function flattenNotifications(pages: readonly NotificationsPage[]): AppNotification[] {
  const seen = new Set<string>();
  const out: AppNotification[] = [];
  for (const page of pages) {
    for (const n of page.notifications) {
      if (seen.has(n.id)) continue;
      seen.add(n.id);
      out.push(n);
    }
  }
  return out;
}

export function unreadNotificationCount(pages: readonly NotificationsPage[] | undefined): number {
  if (!pages || pages.length === 0) return 0;
  return Math.max(0, pages[0].unreadCount);
}

function withUnread<D extends WithNotifications>(data: D, unreadCount: number): D {
  return { ...data, pages: data.pages.map((p) => ({ ...p, unreadCount })) };
}

/** A notification arrived in realtime: prepend it (deduplicated) and bump the unread count. */
export function prependNotification<D extends WithNotifications>(
  data: D,
  notification: AppNotification
): D {
  const already = data.pages.some((p) => p.notifications.some((n) => n.id === notification.id));
  if (already) return data;
  const current = unreadNotificationCount(data.pages);
  const nextUnread = notification.isRead ? current : current + 1;
  if (data.pages.length === 0) {
    return { ...data, pages: [{ notifications: [notification], unreadCount: nextUnread }] };
  }
  const [first, ...rest] = data.pages;
  return withUnread(
    { ...data, pages: [{ ...first, notifications: [notification, ...first.notifications] }, ...rest] },
    nextUnread
  );
}

export function markNotificationReadInCache<D extends WithNotifications>(data: D, id: string): D {
  let wasUnread = false;
  const pages = data.pages.map((page) => ({
    ...page,
    notifications: page.notifications.map((n) => {
      if (n.id !== id || n.isRead) return n;
      wasUnread = true;
      return { ...n, isRead: true };
    })
  }));
  if (!wasUnread) return data;
  return withUnread({ ...data, pages }, Math.max(0, unreadNotificationCount(data.pages) - 1));
}

export function markAllNotificationsReadInCache<D extends WithNotifications>(data: D): D {
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      unreadCount: 0,
      notifications: page.notifications.map((n) => (n.isRead ? n : { ...n, isRead: true }))
    }))
  };
}
