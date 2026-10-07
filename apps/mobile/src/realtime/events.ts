import type {
  AppNotification,
  Conversation,
  Match,
  Message,
  MessagesPage,
  NotificationsPayload,
  ReadEvent,
  RealtimeEvent
} from "../features/types";
import { upsertMessage } from "../features/chat/messages";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function isMessage(value: unknown): value is Message {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isString(value.conversationId) &&
    isString(value.senderId) &&
    isString(value.content) &&
    isString(value.createdAt)
  );
}

function isReadEvent(value: unknown): value is ReadEvent {
  return (
    isRecord(value) && isString(value.conversationId) && isString(value.readerId) && isString(value.readAt)
  );
}

function isMatch(value: unknown): value is Match {
  return (
    isRecord(value) &&
    isString(value.id) &&
    isString(value.conversationId) &&
    isString(value.createdAt) &&
    isRecord(value.user) &&
    isString(value.user.userId)
  );
}

function isNotification(value: unknown): value is AppNotification {
  return (
    isRecord(value) &&
    isString(value.id) &&
    (value.type === "match" || value.type === "message") &&
    typeof value.isRead === "boolean" &&
    isString(value.createdAt)
  );
}

/** Parse a raw websocket frame. Unknown or malformed frames (including pong) return null. */
export function parseRealtimeEvent(raw: string): RealtimeEvent | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }
  if (!isRecord(parsed) || !isString(parsed.type)) {
    return null;
  }
  const data = parsed.data;
  switch (parsed.type) {
    case "message":
      return isMessage(data) ? { type: "message", data } : null;
    case "read":
      return isReadEvent(data) ? { type: "read", data } : null;
    case "match":
      return isMatch(data) ? { type: "match", data } : null;
    case "notification":
      return isNotification(data)
        ? { type: "notification", data: { ...data, data: isRecord(data.data) ? data.data : {} } }
        : null;
    default:
      return null;
  }
}

/** http(s)://host[:port] -> ws(s)://host[:port]/ws?ticket=... */
export function buildSocketUrl(apiBaseUrl: string, ticket: string): string {
  const base = apiBaseUrl.replace(/\/+$/, "").replace(/^http/i, "ws");
  return `${base}/ws?ticket=${encodeURIComponent(ticket)}`;
}

/** Exponential backoff with a cap; `random` in [0,1) adds up to 25% jitter. */
export function reconnectDelay(attempt: number, random = 0): number {
  const base = Math.min(30_000, 1000 * 2 ** Math.max(0, attempt));
  return Math.round(base * (1 + 0.25 * random));
}

function byUpdatedDesc(a: Conversation, b: Conversation): number {
  return Date.parse(b.updatedAt) - Date.parse(a.updatedAt);
}

export type ConversationUpdate = { conversations: Conversation[]; known: boolean };

/**
 * Apply an incoming message to the conversation list.
 * `known` is false when the conversation is not cached (caller should refetch).
 */
export function applyMessageToConversations(
  conversations: Conversation[],
  message: Message,
  myId: string,
  activeConversationId: string | null
): ConversationUpdate {
  const index = conversations.findIndex((c) => c.id === message.conversationId);
  if (index < 0) {
    return { conversations, known: false };
  }
  const current = conversations[index];
  if (current.lastMessage?.id === message.id) {
    return { conversations, known: true };
  }
  const fromPeer = message.senderId !== myId;
  const bump = fromPeer && activeConversationId !== message.conversationId ? 1 : 0;
  const updated: Conversation = {
    ...current,
    lastMessage: message,
    updatedAt: message.createdAt,
    unreadCount: current.unreadCount + bump
  };
  const next = conversations.slice();
  next[index] = updated;
  return { conversations: next.sort(byUpdatedDesc), known: true };
}

export function applyReadToConversations(
  conversations: Conversation[],
  event: ReadEvent,
  myId: string
): Conversation[] {
  const index = conversations.findIndex((c) => c.id === event.conversationId);
  if (index < 0) {
    return conversations;
  }
  const current = conversations[index];
  let updated: Conversation;
  if (event.readerId === myId) {
    if (current.unreadCount === 0) {
      return conversations;
    }
    updated = { ...current, unreadCount: 0 };
  } else {
    const previous = current.peerLastReadAt ? Date.parse(current.peerLastReadAt) : 0;
    if (Date.parse(event.readAt) <= previous) {
      return conversations;
    }
    updated = { ...current, peerLastReadAt: event.readAt };
  }
  const next = conversations.slice();
  next[index] = updated;
  return next;
}

export function clearConversationUnread(
  conversations: Conversation[],
  conversationId: string
): Conversation[] {
  return applyReadToConversations(
    conversations,
    { conversationId, readerId: "self", readAt: new Date(0).toISOString() },
    "self"
  );
}

export function removeConversation(conversations: Conversation[], conversationId: string): Conversation[] {
  return conversations.filter((c) => c.id !== conversationId);
}

export function totalUnread(conversations: Conversation[] | undefined): number {
  if (!conversations) {
    return 0;
  }
  return conversations.reduce((sum, c) => sum + c.unreadCount, 0);
}

/** Add a message to the first page of a messages infinite-query cache (no-op if present). */
export function applyMessageToPages<P extends MessagesPage, K>(
  data: { pages: P[]; pageParams: K[] },
  message: Message
): { pages: P[]; pageParams: K[] } {
  const exists = data.pages.some((page) => page.items.some((m) => m.id === message.id));
  if (exists || data.pages.length === 0) {
    return data;
  }
  const [first, ...rest] = data.pages;
  return { ...data, pages: [{ ...first, items: upsertMessage(first.items, message) }, ...rest] };
}

export function applyMatchToMatches(matches: Match[], match: Match): Match[] {
  if (matches.some((m) => m.id === match.id)) {
    return matches;
  }
  return [match, ...matches];
}

export function removeMatch(matches: Match[], matchId: string): Match[] {
  return matches.filter((m) => m.id !== matchId);
}

export function removeMatchesForUser(matches: Match[], userId: string): Match[] {
  return matches.filter((m) => m.user.userId !== userId);
}

export function applyNotification(
  payload: NotificationsPayload,
  notification: AppNotification
): NotificationsPayload {
  if (payload.items.some((n) => n.id === notification.id)) {
    return payload;
  }
  return {
    items: [notification, ...payload.items],
    unreadCount: payload.unreadCount + (notification.isRead ? 0 : 1)
  };
}

export function markNotificationReadInPayload(
  payload: NotificationsPayload,
  id: string
): NotificationsPayload {
  const target = payload.items.find((n) => n.id === id);
  if (!target || target.isRead) {
    return payload;
  }
  return {
    items: payload.items.map((n) => (n.id === id ? { ...n, isRead: true } : n)),
    unreadCount: Math.max(0, payload.unreadCount - 1)
  };
}

export function markAllReadInPayload(payload: NotificationsPayload): NotificationsPayload {
  return { items: payload.items.map((n) => (n.isRead ? n : { ...n, isRead: true })), unreadCount: 0 };
}

/** New matches = matches whose conversation has no messages yet. */
export function newMatchesWithoutMessages(matches: Match[], conversations: Conversation[]): Match[] {
  const withMessages = new Set(conversations.filter((c) => c.lastMessage !== null).map((c) => c.id));
  return matches.filter((m) => !withMessages.has(m.conversationId));
}
