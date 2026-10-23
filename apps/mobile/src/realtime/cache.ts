import type { InfiniteData } from "@tanstack/react-query";
import type {
  MatchesPage,
  MatchSummary,
  Message,
  MessagesPage,
  NotificationItem,
  NotificationsPage
} from "../api/types";

export type Paged<P> = InfiniteData<P, string | null>;

/* ------------------------------ matches ------------------------------ */

export function findMatch(data: Paged<MatchesPage> | undefined, matchId: string) {
  return data?.pages.flatMap((page) => page.matches).find((match) => match.id === matchId);
}

function mapMatches(
  data: Paged<MatchesPage>,
  fn: (matches: MatchSummary[]) => MatchSummary[]
): Paged<MatchesPage> {
  return { ...data, pages: data.pages.map((page) => ({ ...page, matches: fn(page.matches) })) };
}

/** Moves a match to the top of the first page (most recent activity first). */
function bringToTop(data: Paged<MatchesPage>, match: MatchSummary): Paged<MatchesPage> {
  const without = mapMatches(data, (matches) => matches.filter((item) => item.id !== match.id));
  const [first, ...rest] = without.pages;
  if (!first) return { ...without, pages: [{ matches: [match], nextCursor: null }] };
  return { ...without, pages: [{ ...first, matches: [match, ...first.matches] }, ...rest] };
}

/**
 * message.new: updates last message / activity / unread. Returns null when the match is unknown
 * locally (caller should refetch the list).
 */
export function applyMessageNewToMatches(
  data: Paged<MatchesPage> | undefined,
  message: Message,
  myUserId: string | null,
  activeMatchId: string | null
): Paged<MatchesPage> | null | undefined {
  if (!data) return undefined;
  const existing = findMatch(data, message.matchId);
  if (!existing) return null;
  const incoming = message.senderId !== myUserId;
  const isOpen = activeMatchId === message.matchId;
  const next: MatchSummary = {
    ...existing,
    lastActivityAt: message.createdAt,
    lastMessage: {
      content: message.content,
      senderId: message.senderId,
      createdAt: message.createdAt
    },
    unreadCount: incoming && !isOpen ? existing.unreadCount + 1 : existing.unreadCount
  };
  return bringToTop(data, next);
}

export function applyMatchNew(
  data: Paged<MatchesPage> | undefined,
  match: MatchSummary
): Paged<MatchesPage> | undefined {
  if (!data) return undefined;
  const exists = findMatch(data, match.id);
  if (exists) return mapMatches(data, (list) => list.map((m) => (m.id === match.id ? match : m)));
  return bringToTop(data, match);
}

export function applyMatchRemoved(
  data: Paged<MatchesPage> | undefined,
  matchId: string
): Paged<MatchesPage> | undefined {
  if (!data) return undefined;
  return mapMatches(data, (matches) => matches.filter((match) => match.id !== matchId));
}

export function clearUnread(
  data: Paged<MatchesPage> | undefined,
  matchId: string
): Paged<MatchesPage> | undefined {
  if (!data) return undefined;
  return mapMatches(data, (matches) =>
    matches.map((match) => (match.id === matchId ? { ...match, unreadCount: 0 } : match))
  );
}

export function totalUnread(data: Paged<MatchesPage> | undefined): number {
  if (!data) return 0;
  return data.pages.reduce(
    (sum, page) => sum + page.matches.reduce((acc, match) => acc + match.unreadCount, 0),
    0
  );
}

/* ------------------------------ messages ----------------------------- */

export function flattenMessages(data: Paged<MessagesPage> | undefined): Message[] {
  return data?.pages.flatMap((page) => page.messages) ?? [];
}

/** Inserts a message at the top (newest first), ignoring duplicates (REST response vs socket echo). */
export function appendMessage(
  data: Paged<MessagesPage> | undefined,
  message: Message
): Paged<MessagesPage> | undefined {
  if (!data) return undefined;
  if (flattenMessages(data).some((item) => item.id === message.id)) return data;
  const [first, ...rest] = data.pages;
  if (!first) return { ...data, pages: [{ messages: [message], nextCursor: null }] };
  return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

/** message.read: everything the reader received (sent by someone else) becomes read. */
export function markMessagesRead(
  data: Paged<MessagesPage> | undefined,
  readerId: string,
  readAt: string
): Paged<MessagesPage> | undefined {
  if (!data) return undefined;
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      messages: page.messages.map((message) =>
        message.senderId !== readerId && message.readAt === null ? { ...message, readAt } : message
      )
    }))
  };
}

/* --------------------------- notifications --------------------------- */

function mapNotifications(
  data: Paged<NotificationsPage>,
  fn: (items: NotificationItem[]) => NotificationItem[],
  unreadCount: (current: number) => number
): Paged<NotificationsPage> {
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      notifications: fn(page.notifications),
      unreadCount: unreadCount(page.unreadCount)
    }))
  };
}

export function prependNotification(
  data: Paged<NotificationsPage> | undefined,
  notification: NotificationItem
): Paged<NotificationsPage> | undefined {
  if (!data) return undefined;
  const exists = data.pages.some((page) =>
    page.notifications.some((item) => item.id === notification.id)
  );
  if (exists) return data;
  const bump = notification.readAt === null ? 1 : 0;
  const withCount = mapNotifications(
    data,
    (items) => items,
    (count) => count + bump
  );
  const [first, ...rest] = withCount.pages;
  if (!first) {
    return {
      ...withCount,
      pages: [{ notifications: [notification], unreadCount: bump, nextCursor: null }]
    };
  }
  return {
    ...withCount,
    pages: [{ ...first, notifications: [notification, ...first.notifications] }, ...rest]
  };
}

export function markNotificationReadInCache(
  data: Paged<NotificationsPage> | undefined,
  id: string,
  readAt: string
): Paged<NotificationsPage> | undefined {
  if (!data) return undefined;
  const target = data.pages.flatMap((p) => p.notifications).find((item) => item.id === id);
  if (!target || target.readAt !== null) return data;
  return mapNotifications(
    data,
    (items) => items.map((item) => (item.id === id ? { ...item, readAt } : item)),
    (count) => Math.max(0, count - 1)
  );
}

export function markAllNotificationsReadInCache(
  data: Paged<NotificationsPage> | undefined,
  readAt: string
): Paged<NotificationsPage> | undefined {
  if (!data) return undefined;
  return mapNotifications(
    data,
    (items) => items.map((item) => (item.readAt === null ? { ...item, readAt } : item)),
    () => 0
  );
}

export function notificationUnreadCount(data: Paged<NotificationsPage> | undefined): number {
  return data?.pages[0]?.unreadCount ?? 0;
}
