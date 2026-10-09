import type { InfiniteData, QueryClient } from '@tanstack/react-query';
import { fr } from '../i18n/fr';
import { qk } from '../query/keys';
import type {
  AppNotification,
  Conversation,
  Message,
  MessagesPage,
  NotificationsPage,
  ProfileSummary,
  RealtimeEvent,
} from '../api/types';

export type MessagesData = InfiniteData<MessagesPage, string | undefined>;

export type RealtimeContext = {
  myUserId: string;
  activeConversationId: string | null;
};

export type RealtimeEffect =
  | {
      kind: 'toast';
      message: string;
      conversationId?: string;
      user?: Pick<ProfileSummary, 'id' | 'firstName' | 'mainPhotoUrl'>;
    }
  | { kind: 'conversationRemoved'; conversationId: string };

/* ---------------------------- messages cache ---------------------------- */

function mapPages(data: MessagesData, fn: (m: Message) => Message): MessagesData {
  return { ...data, pages: data.pages.map((p) => ({ ...p, messages: p.messages.map(fn) })) };
}

function hasMessage(data: MessagesData, id: string): boolean {
  return data.pages.some((p) => p.messages.some((m) => m.id === id));
}

/** Insert (newest page, end) or replace an existing message by id. */
export function upsertMessage(qc: QueryClient, message: Message): void {
  qc.setQueryData<MessagesData>(qk.messages(message.conversationId), (old) => {
    if (!old) return old;
    if (hasMessage(old, message.id)) {
      return mapPages(old, (m) =>
        m.id === message.id ? { ...m, ...message, localState: undefined } : m,
      );
    }
    const [first, ...rest] = old.pages;
    if (!first) return old;
    return { ...old, pages: [{ ...first, messages: [...first.messages, message] }, ...rest] };
  });
}

/**
 * A message of mine coming from the websocket: if the optimistic copy is still pending, swap it in
 * place so the bubble never shows up twice while the POST response is in flight.
 */
export function upsertOwnMessage(qc: QueryClient, message: Message): void {
  qc.setQueryData<MessagesData>(qk.messages(message.conversationId), (old) => {
    if (!old || hasMessage(old, message.id)) return old;
    let swapped = false;
    const pages = old.pages.map((p) => ({
      ...p,
      messages: p.messages.map((m) => {
        if (!swapped && m.localState === 'sending' && m.body === message.body) {
          swapped = true;
          return message;
        }
        return m;
      }),
    }));
    if (swapped) return { ...old, pages };
    return old;
  });
  // Nothing pending (message sent from another device): regular insert.
  const current = qc.getQueryData<MessagesData>(qk.messages(message.conversationId));
  if (current && !hasMessage(current, message.id)) upsertMessage(qc, message);
}

export function addOptimisticMessage(qc: QueryClient, message: Message): void {
  upsertMessage(qc, message);
}

/** Swap the optimistic message by the server one, never leaving a duplicate. */
export function resolveOptimisticMessage(qc: QueryClient, tempId: string, real: Message): void {
  qc.setQueryData<MessagesData>(qk.messages(real.conversationId), (old) => {
    if (!old) return old;
    const realExists = hasMessage(old, real.id);
    const pages = old.pages.map((p) => ({
      ...p,
      messages: p.messages.flatMap((m) => {
        if (m.id === tempId) return realExists ? [] : [real];
        return [m];
      }),
    }));
    return { ...old, pages };
  });
}

export function failOptimisticMessage(
  qc: QueryClient,
  conversationId: string,
  tempId: string,
): void {
  qc.setQueryData<MessagesData>(qk.messages(conversationId), (old) =>
    old
      ? mapPages(old, (m) => (m.id === tempId ? { ...m, localState: 'failed' as const } : m))
      : old,
  );
}

export function removeOptimisticMessage(
  qc: QueryClient,
  conversationId: string,
  tempId: string,
): void {
  qc.setQueryData<MessagesData>(qk.messages(conversationId), (old) =>
    old
      ? {
          ...old,
          pages: old.pages.map((p) => ({
            ...p,
            messages: p.messages.filter((m) => m.id !== tempId),
          })),
        }
      : old,
  );
}

export function markSentMessagesRead(
  qc: QueryClient,
  conversationId: string,
  myUserId: string,
  readAt: string,
): void {
  qc.setQueryData<MessagesData>(qk.messages(conversationId), (old) =>
    old
      ? mapPages(old, (m) =>
          m.senderId === myUserId && !m.readAt && !m.localState ? { ...m, readAt } : m,
        )
      : old,
  );
}

/* ------------------------- conversations cache -------------------------- */

function sortConversations(list: Conversation[]): Conversation[] {
  return [...list].sort((a, b) =>
    a.updatedAt < b.updatedAt ? 1 : a.updatedAt > b.updatedAt ? -1 : 0,
  );
}

export function applyMessageToConversations(
  qc: QueryClient,
  message: Message,
  ctx: RealtimeContext,
  options: { countUnread: boolean } = { countUnread: true },
): void {
  const list = qc.getQueryData<Conversation[]>(qk.conversations);
  const known = list?.some((c) => c.id === message.conversationId);
  if (!list || !known) {
    void qc.invalidateQueries({ queryKey: qk.conversations });
    return;
  }
  const incoming = message.senderId !== ctx.myUserId;
  qc.setQueryData<Conversation[]>(qk.conversations, (old) =>
    old
      ? sortConversations(
          old.map((c) =>
            c.id === message.conversationId
              ? {
                  ...c,
                  lastMessage: {
                    id: message.id,
                    body: message.body,
                    senderId: message.senderId,
                    createdAt: message.createdAt,
                  },
                  updatedAt: message.createdAt,
                  unreadCount:
                    incoming && options.countUnread && ctx.activeConversationId !== c.id
                      ? c.unreadCount + 1
                      : c.unreadCount,
                }
              : c,
          ),
        )
      : old,
  );
}

export function clearConversationUnread(qc: QueryClient, conversationId: string): void {
  qc.setQueryData<Conversation[]>(qk.conversations, (old) =>
    old ? old.map((c) => (c.id === conversationId ? { ...c, unreadCount: 0 } : c)) : old,
  );
}

export function removeConversation(qc: QueryClient, conversationId: string): void {
  qc.setQueryData<Conversation[]>(qk.conversations, (old) =>
    old ? old.filter((c) => c.id !== conversationId) : old,
  );
  qc.removeQueries({ queryKey: qk.messages(conversationId) });
}

export function totalUnread(list: Conversation[] | undefined): number {
  return list ? list.reduce((sum, c) => sum + c.unreadCount, 0) : 0;
}

/* ------------------------- notifications cache -------------------------- */

export function prependNotification(qc: QueryClient, n: AppNotification): void {
  const current = qc.getQueryData<NotificationsPage>(qk.notifications);
  if (!current) {
    void qc.invalidateQueries({ queryKey: qk.notifications });
    return;
  }
  if (current.notifications.some((x) => x.id === n.id)) return;
  qc.setQueryData<NotificationsPage>(qk.notifications, {
    notifications: [n, ...current.notifications],
    unreadCount: current.unreadCount + (n.isRead ? 0 : 1),
  });
}

export function markNotificationReadLocal(qc: QueryClient, id: string): void {
  qc.setQueryData<NotificationsPage>(qk.notifications, (old) => {
    if (!old) return old;
    const target = old.notifications.find((n) => n.id === id);
    if (!target || target.isRead) return old;
    return {
      notifications: old.notifications.map((n) =>
        n.id === id ? { ...n, isRead: true, readAt: new Date().toISOString() } : n,
      ),
      unreadCount: Math.max(0, old.unreadCount - 1),
    };
  });
}

export function markAllNotificationsReadLocal(qc: QueryClient): void {
  const now = new Date().toISOString();
  qc.setQueryData<NotificationsPage>(qk.notifications, (old) =>
    old
      ? {
          notifications: old.notifications.map((n) =>
            n.isRead ? n : { ...n, isRead: true, readAt: now },
          ),
          unreadCount: 0,
        }
      : old,
  );
}

/* ----------------------------- event router ----------------------------- */

/** Applies a realtime event to the react-query caches. Returns an optional UI side effect. */
export function applyRealtimeEvent(
  qc: QueryClient,
  event: RealtimeEvent,
  ctx: RealtimeContext,
): RealtimeEffect | null {
  switch (event.type) {
    case 'message.new': {
      if (event.data.senderId === ctx.myUserId) upsertOwnMessage(qc, event.data);
      else upsertMessage(qc, event.data);
      applyMessageToConversations(qc, event.data, ctx);
      return null;
    }
    case 'message.read': {
      const { conversationId, readerId, readAt } = event.data;
      if (readerId === ctx.myUserId) clearConversationUnread(qc, conversationId);
      else markSentMessagesRead(qc, conversationId, ctx.myUserId, readAt);
      return null;
    }
    case 'match.new': {
      void qc.invalidateQueries({ queryKey: qk.conversations });
      void qc.invalidateQueries({ queryKey: qk.matches });
      void qc.invalidateQueries({ queryKey: qk.likes });
      return {
        kind: 'toast',
        message: fr.match.toast(event.data.user.firstName),
        conversationId: event.data.conversationId,
        user: event.data.user,
      };
    }
    case 'match.removed': {
      removeConversation(qc, event.data.conversationId);
      void qc.invalidateQueries({ queryKey: qk.matches });
      return { kind: 'conversationRemoved', conversationId: event.data.conversationId };
    }
    case 'notification.new': {
      const n = event.data;
      prependNotification(qc, n);
      const conversationId = n.data?.conversationId;
      if (n.type === 'message') {
        if (conversationId && conversationId === ctx.activeConversationId) return null;
        return {
          kind: 'toast',
          message: n.body
            ? `${fr.messages.newMessageToast} : ${n.body}`
            : fr.messages.newMessageToast,
          conversationId,
        };
      }
      // Match toasts are produced by `match.new`; avoid showing two.
      return null;
    }
    default:
      return null;
  }
}

/** After a (re)connection: pull everything that may have changed while offline. */
export function refetchAfterReconnect(qc: QueryClient): void {
  void qc.invalidateQueries({ queryKey: qk.conversations });
  void qc.invalidateQueries({ queryKey: ['messages'] });
  void qc.invalidateQueries({ queryKey: qk.notifications });
  void qc.invalidateQueries({ queryKey: qk.likes });
  void qc.invalidateQueries({ queryKey: qk.matches });
}
