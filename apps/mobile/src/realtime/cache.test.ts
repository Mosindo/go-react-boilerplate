import { QueryClient } from '@tanstack/react-query';
import { queryKeys } from '../api/queryKeys';
import type { RealtimeEvent } from '../api/types';
import { applyRealtimeEvent, refetchAfterReconnect, totalUnread } from './cache';
import type { ConversationsData, MessagesData, NotificationsData } from './cache';
import { ME, conversation, message, notification } from './testFixtures';

function setup() {
  const qc = new QueryClient();
  const conversations: ConversationsData = {
    pageParams: [null],
    pages: [
      {
        conversations: [
          conversation('c1', {
            lastActivityAt: '2026-03-01T12:00:00Z',
            lastMessage: { body: 'old', senderId: 'u-c1', createdAt: '2026-03-01T12:00:00Z' },
          }),
          conversation('c2', { lastActivityAt: '2026-03-01T11:00:00Z', unreadCount: 2 }),
        ],
        nextCursor: null,
      },
    ],
  };
  qc.setQueryData(queryKeys.conversations, conversations);
  const messages: MessagesData = {
    pageParams: [null],
    pages: [{ messages: [message('m1', 'c1')], nextCursor: null, otherLastReadAt: null }],
  };
  qc.setQueryData(queryKeys.messages('c1'), messages);
  const notifications: NotificationsData = {
    pageParams: [null],
    pages: [{ notifications: [notification('n1')], unreadCount: 1, nextCursor: null }],
  };
  qc.setQueryData(queryKeys.notifications, notifications);
  return qc;
}

const ctx = (activeConversationId: string | null = null) => ({
  myUserId: ME,
  activeConversationId,
});
const convs = (qc: QueryClient) =>
  qc.getQueryData<ConversationsData>(queryKeys.conversations)!.pages[0].conversations;

describe('message.new', () => {
  it('updates last message, moves the chat to the top and counts unread', () => {
    const qc = setup();
    const m = message('m2', 'c2', { createdAt: '2026-03-02T09:00:00Z' });
    const effects = applyRealtimeEvent(qc, { type: 'message.new', data: m }, ctx());
    const list = convs(qc);
    expect(list.map((c) => c.id)).toEqual(['c2', 'c1']);
    expect(list[0].unreadCount).toBe(3);
    expect(list[0].lastMessage?.body).toBe('body m2');
    expect(totalUnread(qc.getQueryData(queryKeys.conversations))).toBe(3);
    expect(effects.incomingMessage?.id).toBe('m2');
  });

  it('does not add unread (nor a toast) for the conversation currently open', () => {
    const qc = setup();
    const effects = applyRealtimeEvent(
      qc,
      { type: 'message.new', data: message('m2', 'c2') },
      ctx('c2'),
    );
    expect(convs(qc).find((c) => c.id === 'c2')?.unreadCount).toBe(2);
    expect(effects.incomingMessage).toBeUndefined();
  });

  it('does not count my own messages as unread', () => {
    const qc = setup();
    applyRealtimeEvent(
      qc,
      { type: 'message.new', data: message('m2', 'c1', { senderId: ME }) },
      ctx(),
    );
    expect(convs(qc).find((c) => c.id === 'c1')?.unreadCount).toBe(0);
  });

  it('inserts into an open message cache once (dedupes by id)', () => {
    const qc = setup();
    const event: RealtimeEvent = { type: 'message.new', data: message('m2', 'c1') };
    applyRealtimeEvent(qc, event, ctx('c1'));
    applyRealtimeEvent(qc, event, ctx('c1'));
    const data = qc.getQueryData<MessagesData>(queryKeys.messages('c1'))!;
    expect(data.pages[0].messages.map((m) => m.id)).toEqual(['m2', 'm1']);
  });

  it('invalidates the list when the conversation is unknown', () => {
    const qc = setup();
    const spy = jest.spyOn(qc, 'invalidateQueries');
    applyRealtimeEvent(qc, { type: 'message.new', data: message('m9', 'zzz') }, ctx());
    expect(spy).toHaveBeenCalledWith({ queryKey: queryKeys.conversations });
  });
});

describe('message.read', () => {
  it('stores the other side read time (never moves backwards)', () => {
    const qc = setup();
    const read = (readAt: string) =>
      applyRealtimeEvent(
        qc,
        { type: 'message.read', data: { conversationId: 'c1', readerId: 'u-c1', readAt } },
        ctx(),
      );
    read('2026-03-02T12:00:00Z');
    read('2026-03-02T11:00:00Z');
    const data = qc.getQueryData<MessagesData>(queryKeys.messages('c1'))!;
    expect(data.pages[0].otherLastReadAt).toBe('2026-03-02T12:00:00Z');
  });

  it('clears my unread counter when I read on another device', () => {
    const qc = setup();
    applyRealtimeEvent(
      qc,
      {
        type: 'message.read',
        data: { conversationId: 'c2', readerId: ME, readAt: '2026-03-02T12:00:00Z' },
      },
      ctx(),
    );
    expect(convs(qc).find((c) => c.id === 'c2')?.unreadCount).toBe(0);
  });
});

describe('match.new / conversation.removed', () => {
  it('adds a new match once and surfaces it for the UI', () => {
    const qc = setup();
    const created = conversation('c3', { lastActivityAt: '2026-03-03T00:00:00Z' });
    const effects = applyRealtimeEvent(qc, { type: 'match.new', data: created }, ctx());
    applyRealtimeEvent(qc, { type: 'match.new', data: created }, ctx());
    expect(convs(qc).map((c) => c.id)).toEqual(['c3', 'c1', 'c2']);
    expect(effects.newMatch?.id).toBe('c3');
  });

  it('removes the conversation and its messages', () => {
    const qc = setup();
    const effects = applyRealtimeEvent(
      qc,
      { type: 'conversation.removed', data: { conversationId: 'c1' } },
      ctx(),
    );
    expect(convs(qc).map((c) => c.id)).toEqual(['c2']);
    expect(qc.getQueryData(queryKeys.messages('c1'))).toBeUndefined();
    expect(effects.removedConversationId).toBe('c1');
  });
});

describe('notification.new', () => {
  it('prepends, dedupes and bumps the unread count', () => {
    const qc = setup();
    const event: RealtimeEvent = { type: 'notification.new', data: notification('n2') };
    applyRealtimeEvent(qc, event, ctx());
    applyRealtimeEvent(qc, event, ctx());
    const data = qc.getQueryData<NotificationsData>(queryKeys.notifications)!;
    expect(data.pages[0].notifications.map((n) => n.id)).toEqual(['n2', 'n1']);
    expect(data.pages[0].unreadCount).toBe(2);
  });

  it('refreshes likes on a new_like', () => {
    const qc = setup();
    const spy = jest.spyOn(qc, 'invalidateQueries');
    applyRealtimeEvent(qc, { type: 'notification.new', data: notification('n3') }, ctx());
    expect(spy).toHaveBeenCalledWith({ queryKey: queryKeys.likes });
  });
});

describe('refetchAfterReconnect', () => {
  it('invalidates conversations, notifications, likes and open messages', () => {
    const qc = setup();
    const spy = jest.spyOn(qc, 'invalidateQueries');
    refetchAfterReconnect(qc);
    const keys = spy.mock.calls.map((c) =>
      JSON.stringify((c[0] as { queryKey: unknown }).queryKey),
    );
    expect(keys).toEqual(
      expect.arrayContaining([
        JSON.stringify(queryKeys.conversations),
        JSON.stringify(queryKeys.notifications),
        JSON.stringify(queryKeys.likes),
        JSON.stringify(['messages']),
      ]),
    );
  });
});
