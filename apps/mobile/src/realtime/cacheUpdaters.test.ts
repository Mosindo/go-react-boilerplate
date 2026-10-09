import { QueryClient } from '@tanstack/react-query';
import { qk } from '../query/keys';
import type { AppNotification, Conversation, Message, NotificationsPage } from '../api/types';
import {
  applyRealtimeEvent,
  failOptimisticMessage,
  markAllNotificationsReadLocal,
  markNotificationReadLocal,
  resolveOptimisticMessage,
  totalUnread,
  upsertMessage,
  type MessagesData,
} from './cacheUpdaters';

const ME = 'me';
const OTHER = 'other';
const ctx = { myUserId: ME, activeConversationId: null as string | null };

const msg = (id: string, over: Partial<Message> = {}): Message => ({
  id,
  conversationId: 'c1',
  senderId: OTHER,
  body: `body-${id}`,
  createdAt: '2026-10-09T10:00:00Z',
  readAt: null,
  ...over,
});

const conv = (id: string, over: Partial<Conversation> = {}): Conversation => ({
  id,
  matchId: `m-${id}`,
  user: { id: `u-${id}`, firstName: id, age: 30, mainPhotoUrl: null },
  lastMessage: null,
  unreadCount: 0,
  updatedAt: '2026-10-09T09:00:00Z',
  ...over,
});

function seedMessages(qc: QueryClient, pages: Message[][]) {
  qc.setQueryData<MessagesData>(qk.messages('c1'), {
    pages: pages.map((messages, i) => ({ messages, hasMore: i < pages.length - 1 })),
    pageParams: pages.map((_p, i) => (i === 0 ? undefined : 'x')),
  });
}
const flat = (qc: QueryClient) =>
  qc.getQueryData<MessagesData>(qk.messages('c1'))?.pages.flatMap((p) => p.messages) ?? [];

describe('realtime cache updaters', () => {
  let qc: QueryClient;
  beforeEach(() => {
    qc = new QueryClient();
  });
  afterEach(() => qc.clear());

  it('message.new appends to the newest page once (idempotent) and updates the conversation list', () => {
    seedMessages(qc, [[msg('1')], [msg('0')]]);
    qc.setQueryData(qk.conversations, [
      conv('c1'),
      conv('c2', { updatedAt: '2026-10-09T09:30:00Z' }),
    ]);
    const event = {
      type: 'message.new',
      data: msg('2', { createdAt: '2026-10-09T10:05:00Z' }),
    } as const;
    applyRealtimeEvent(qc, event, ctx);
    applyRealtimeEvent(qc, event, ctx);
    expect(flat(qc).map((m) => m.id)).toEqual(['1', '2', '0']);
    const list = qc.getQueryData<Conversation[]>(qk.conversations)!;
    expect(list[0]).toMatchObject({ id: 'c1', unreadCount: 2, lastMessage: { id: '2' } });
    expect(list.map((c) => c.id)).toEqual(['c1', 'c2']);
  });

  it('does not count unread for my own messages or the open conversation', () => {
    qc.setQueryData(qk.conversations, [conv('c1')]);
    applyRealtimeEvent(qc, { type: 'message.new', data: msg('5', { senderId: ME }) }, ctx);
    applyRealtimeEvent(
      qc,
      { type: 'message.new', data: msg('6') },
      { ...ctx, activeConversationId: 'c1' },
    );
    expect(qc.getQueryData<Conversation[]>(qk.conversations)![0]?.unreadCount).toBe(0);
  });

  it('invalidates the conversation list when the conversation is unknown', () => {
    qc.setQueryData(qk.conversations, [conv('other')]);
    const spy = jest.spyOn(qc, 'invalidateQueries');
    applyRealtimeEvent(qc, { type: 'message.new', data: msg('7') }, ctx);
    expect(spy).toHaveBeenCalledWith({ queryKey: qk.conversations });
  });

  it('message.read marks my sent messages as read (and clears unread when I am the reader)', () => {
    seedMessages(qc, [[msg('1', { senderId: ME }), msg('2', { senderId: OTHER })]]);
    applyRealtimeEvent(
      qc,
      { type: 'message.read', data: { conversationId: 'c1', readerId: OTHER, readAt: 'T' } },
      ctx,
    );
    expect(flat(qc).map((m) => m.readAt)).toEqual(['T', null]);

    qc.setQueryData(qk.conversations, [conv('c1', { unreadCount: 3 })]);
    applyRealtimeEvent(
      qc,
      { type: 'message.read', data: { conversationId: 'c1', readerId: ME, readAt: 'T' } },
      ctx,
    );
    expect(qc.getQueryData<Conversation[]>(qk.conversations)![0]?.unreadCount).toBe(0);
  });

  it('match.new invalidates lists and returns a toast effect', () => {
    const spy = jest.spyOn(qc, 'invalidateQueries');
    const effect = applyRealtimeEvent(
      qc,
      {
        type: 'match.new',
        data: {
          matchId: 'm',
          conversationId: 'c9',
          user: { id: 'u', firstName: 'Léa', age: 28, mainPhotoUrl: null },
        },
      },
      ctx,
    );
    expect(effect).toMatchObject({ kind: 'toast', conversationId: 'c9' });
    expect(spy).toHaveBeenCalledWith({ queryKey: qk.conversations });
    expect(spy).toHaveBeenCalledWith({ queryKey: qk.matches });
  });

  it('match.removed drops the conversation and its messages', () => {
    qc.setQueryData(qk.conversations, [conv('c1'), conv('c2')]);
    seedMessages(qc, [[msg('1')]]);
    const effect = applyRealtimeEvent(
      qc,
      { type: 'match.removed', data: { matchId: 'm', conversationId: 'c1' } },
      ctx,
    );
    expect(effect).toEqual({ kind: 'conversationRemoved', conversationId: 'c1' });
    expect(qc.getQueryData<Conversation[]>(qk.conversations)!.map((c) => c.id)).toEqual(['c2']);
    expect(qc.getQueryData(qk.messages('c1'))).toBeUndefined();
  });

  describe('notifications', () => {
    const notif = (id: string, over: Partial<AppNotification> = {}): AppNotification => ({
      id,
      type: 'message',
      title: 't',
      body: 'Salut',
      data: { conversationId: 'c1' },
      isRead: false,
      readAt: null,
      createdAt: '2026-10-09T10:00:00Z',
      ...over,
    });
    const seed = () =>
      qc.setQueryData<NotificationsPage>(qk.notifications, {
        notifications: [notif('a')],
        unreadCount: 1,
      });

    it('notification.new prepends, bumps the unread count, dedupes and toasts', () => {
      seed();
      const e = { type: 'notification.new', data: notif('b') } as const;
      const effect = applyRealtimeEvent(qc, e, ctx);
      applyRealtimeEvent(qc, e, ctx);
      const page = qc.getQueryData<NotificationsPage>(qk.notifications)!;
      expect(page.notifications.map((n) => n.id)).toEqual(['b', 'a']);
      expect(page.unreadCount).toBe(2);
      expect(effect).toMatchObject({ kind: 'toast', conversationId: 'c1' });
    });

    it('does not toast for the conversation being read', () => {
      seed();
      const effect = applyRealtimeEvent(
        qc,
        { type: 'notification.new', data: notif('c') },
        { ...ctx, activeConversationId: 'c1' },
      );
      expect(effect).toBeNull();
    });

    it('marks one / all as read locally', () => {
      qc.setQueryData<NotificationsPage>(qk.notifications, {
        notifications: [notif('a'), notif('b')],
        unreadCount: 2,
      });
      markNotificationReadLocal(qc, 'a');
      markNotificationReadLocal(qc, 'a');
      expect(qc.getQueryData<NotificationsPage>(qk.notifications)).toMatchObject({
        unreadCount: 1,
      });
      markAllNotificationsReadLocal(qc);
      const page = qc.getQueryData<NotificationsPage>(qk.notifications)!;
      expect(page.unreadCount).toBe(0);
      expect(page.notifications.every((n) => n.isRead)).toBe(true);
    });
  });

  describe('optimistic messages', () => {
    it('replaces the temp message by the server one', () => {
      seedMessages(qc, [[msg('1')]]);
      upsertMessage(qc, msg('tmp-1', { senderId: ME, localState: 'sending' }));
      resolveOptimisticMessage(qc, 'tmp-1', msg('100', { senderId: ME }));
      expect(flat(qc).map((m) => m.id)).toEqual(['1', '100']);
      expect(flat(qc)[1]?.localState).toBeUndefined();
    });

    it('never duplicates when the websocket delivered the real message first', () => {
      seedMessages(qc, [[]]);
      upsertMessage(qc, msg('tmp-1', { senderId: ME, localState: 'sending' }));
      upsertMessage(qc, msg('100', { senderId: ME }));
      resolveOptimisticMessage(qc, 'tmp-1', msg('100', { senderId: ME }));
      expect(flat(qc).map((m) => m.id)).toEqual(['100']);
    });

    it('a websocket echo of my own message swaps the pending bubble in place (no flash of two)', () => {
      seedMessages(qc, [[msg('1')]]);
      upsertMessage(qc, msg('tmp-1', { senderId: ME, body: 'yo', localState: 'sending' }));
      applyRealtimeEvent(
        qc,
        { type: 'message.new', data: msg('100', { senderId: ME, body: 'yo' }) },
        ctx,
      );
      expect(flat(qc).map((m) => m.id)).toEqual(['1', '100']);
      // a message sent from another device is simply inserted
      applyRealtimeEvent(
        qc,
        { type: 'message.new', data: msg('101', { senderId: ME, body: 'other device' }) },
        ctx,
      );
      expect(flat(qc).map((m) => m.id)).toEqual(['1', '100', '101']);
    });

    it('flags failures', () => {
      seedMessages(qc, [[msg('tmp-1', { senderId: ME, localState: 'sending' })]]);
      failOptimisticMessage(qc, 'c1', 'tmp-1');
      expect(flat(qc)[0]?.localState).toBe('failed');
    });
  });

  it('totalUnread sums unread counts', () => {
    expect(totalUnread([conv('a', { unreadCount: 2 }), conv('b', { unreadCount: 3 })])).toBe(5);
    expect(totalUnread(undefined)).toBe(0);
  });
});
