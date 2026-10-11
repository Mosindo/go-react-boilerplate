import type { InfiniteData, QueryClient } from '@tanstack/react-query';
import { queryKeys } from '../api/queryKeys';
import type {
  AppNotification,
  Conversation,
  ConversationsPage,
  Message,
  MessagesPage,
  NotificationsPage,
  RealtimeEvent,
} from '../api/types';

export type ConversationsData = InfiniteData<ConversationsPage>;
export type MessagesData = InfiniteData<MessagesPage>;
export type NotificationsData = InfiniteData<NotificationsPage>;

/* ------------------------- pure cache reducers (tested) ------------------------- */

function mapConversations(
  data: ConversationsData,
  fn: (c: Conversation) => Conversation,
): ConversationsData {
  return {
    ...data,
    pages: data.pages.map((p) => ({ ...p, conversations: p.conversations.map(fn) })),
  };
}

export function hasConversation(data: ConversationsData | undefined, id: string): boolean {
  return !!data?.pages.some((p) => p.conversations.some((c) => c.id === id));
}

function newestFirst(a: Conversation, b: Conversation): number {
  return b.lastActivityAt.localeCompare(a.lastActivityAt);
}

/** Applies a new message to the conversation list: last message, activity, unread, and reorder. */
export function conversationsWithMessage(
  data: ConversationsData,
  message: Message,
  myUserId: string,
  isActive: boolean,
): ConversationsData {
  const updated = mapConversations(data, (c) => {
    if (c.id !== message.conversationId) return c;
    if (c.lastMessage && c.lastActivityAt > message.createdAt) return c;
    const incoming = message.senderId !== myUserId;
    return {
      ...c,
      lastMessage: { body: message.body, senderId: message.senderId, createdAt: message.createdAt },
      lastActivityAt: message.createdAt,
      unreadCount: incoming && !isActive ? c.unreadCount + 1 : c.unreadCount,
    };
  });
  // Keep the first page sorted so the conversation that just moved rises to the top.
  const [first, ...rest] = updated.pages;
  if (!first) return updated;
  return {
    ...updated,
    pages: [{ ...first, conversations: [...first.conversations].sort(newestFirst) }, ...rest],
  };
}

export function conversationsWithNew(
  data: ConversationsData,
  conversation: Conversation,
): ConversationsData {
  if (hasConversation(data, conversation.id)) return data;
  const [first, ...rest] = data.pages;
  if (!first) return data;
  return {
    ...data,
    pages: [
      { ...first, conversations: [conversation, ...first.conversations].sort(newestFirst) },
      ...rest,
    ],
  };
}

export function conversationsWithout(data: ConversationsData, id: string): ConversationsData {
  return {
    ...data,
    pages: data.pages.map((p) => ({
      ...p,
      conversations: p.conversations.filter((c) => c.id !== id),
    })),
  };
}

export function conversationsMarkedRead(data: ConversationsData, id: string): ConversationsData {
  return mapConversations(data, (c) =>
    c.id === id && c.unreadCount > 0 ? { ...c, unreadCount: 0 } : c,
  );
}

export function totalUnread(data: ConversationsData | undefined): number {
  if (!data) return 0;
  return data.pages.reduce(
    (sum, p) => sum + p.conversations.reduce((s, c) => s + c.unreadCount, 0),
    0,
  );
}

export function messagesWith(data: MessagesData, message: Message): MessagesData {
  if (data.pages.some((p) => p.messages.some((m) => m.id === message.id))) return data;
  const [first, ...rest] = data.pages;
  if (!first) return data;
  return { ...data, pages: [{ ...first, messages: [message, ...first.messages] }, ...rest] };
}

export function messagesWithReadAt(data: MessagesData, readAt: string): MessagesData {
  return {
    ...data,
    pages: data.pages.map((p) => ({
      ...p,
      otherLastReadAt: p.otherLastReadAt && p.otherLastReadAt > readAt ? p.otherLastReadAt : readAt,
    })),
  };
}

export function notificationsWith(
  data: NotificationsData,
  notification: AppNotification,
): NotificationsData {
  if (data.pages.some((p) => p.notifications.some((n) => n.id === notification.id))) return data;
  const unreadCount = (data.pages[0]?.unreadCount ?? 0) + (notification.isRead ? 0 : 1);
  return {
    ...data,
    pages: data.pages.map((p, index) => ({
      ...p,
      unreadCount,
      notifications: index === 0 ? [notification, ...p.notifications] : p.notifications,
    })),
  };
}

export function notificationsMarkedRead(data: NotificationsData, id: string): NotificationsData {
  const target = data.pages.flatMap((p) => p.notifications).find((n) => n.id === id);
  if (!target || target.isRead) return data;
  const unreadCount = Math.max(0, (data.pages[0]?.unreadCount ?? 0) - 1);
  return {
    ...data,
    pages: data.pages.map((p) => ({
      ...p,
      unreadCount,
      notifications: p.notifications.map((n) => (n.id === id ? { ...n, isRead: true } : n)),
    })),
  };
}

export function notificationsAllRead(data: NotificationsData): NotificationsData {
  return {
    ...data,
    pages: data.pages.map((p) => ({
      ...p,
      unreadCount: 0,
      notifications: p.notifications.map((n) => (n.isRead ? n : { ...n, isRead: true })),
    })),
  };
}

/* ---------------------------- event -> cache wiring ---------------------------- */

export interface RealtimeContext {
  myUserId: string;
  activeConversationId: string | null;
}

export interface RealtimeEffects {
  newMatch?: Conversation;
  newNotification?: AppNotification;
  /** A message from the other person while its chat is not open. */
  incomingMessage?: Message;
  removedConversationId?: string;
}

/** Updates react-query caches from a realtime event. Returns side effects for the UI (toasts). */
export function applyRealtimeEvent(
  qc: QueryClient,
  event: RealtimeEvent,
  ctx: RealtimeContext,
): RealtimeEffects {
  switch (event.type) {
    case 'message.new': {
      const message = event.data;
      const isActive = ctx.activeConversationId === message.conversationId;
      qc.setQueryData<MessagesData>(queryKeys.messages(message.conversationId), (old) =>
        old ? messagesWith(old, message) : old,
      );
      const conversations = qc.getQueryData<ConversationsData>(queryKeys.conversations);
      if (conversations && hasConversation(conversations, message.conversationId)) {
        qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
          old ? conversationsWithMessage(old, message, ctx.myUserId, isActive) : old,
        );
      } else {
        void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      }
      return message.senderId !== ctx.myUserId && !isActive ? { incomingMessage: message } : {};
    }
    case 'message.read': {
      const { conversationId, readerId, readAt } = event.data;
      if (readerId === ctx.myUserId) {
        qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
          old ? conversationsMarkedRead(old, conversationId) : old,
        );
      } else {
        qc.setQueryData<MessagesData>(queryKeys.messages(conversationId), (old) =>
          old ? messagesWithReadAt(old, readAt) : old,
        );
      }
      return {};
    }
    case 'match.new': {
      qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
        old ? conversationsWithNew(old, event.data) : old,
      );
      void qc.invalidateQueries({ queryKey: queryKeys.conversations });
      void qc.invalidateQueries({ queryKey: queryKeys.likes });
      return { newMatch: event.data };
    }
    case 'notification.new': {
      qc.setQueryData<NotificationsData>(queryKeys.notifications, (old) =>
        old ? notificationsWith(old, event.data) : old,
      );
      void qc.invalidateQueries({ queryKey: queryKeys.notifications });
      if (event.data.type === 'new_like') {
        void qc.invalidateQueries({ queryKey: queryKeys.likes });
      }
      return { newNotification: event.data };
    }
    case 'conversation.removed': {
      const { conversationId } = event.data;
      qc.setQueryData<ConversationsData>(queryKeys.conversations, (old) =>
        old ? conversationsWithout(old, conversationId) : old,
      );
      qc.removeQueries({ queryKey: queryKeys.messages(conversationId) });
      return { removedConversationId: conversationId };
    }
  }
}

/** After a reconnect we may have missed frames: refetch everything realtime touches. */
export function refetchAfterReconnect(qc: QueryClient): void {
  void qc.invalidateQueries({ queryKey: queryKeys.conversations });
  void qc.invalidateQueries({ queryKey: queryKeys.notifications });
  void qc.invalidateQueries({ queryKey: queryKeys.likes });
  void qc.invalidateQueries({ queryKey: ['messages'] });
}
