import type { AppNotification, Conversation, Message } from '../api/types';

export const ME = 'me';

export function conversation(id: string, overrides: Partial<Conversation> = {}): Conversation {
  return {
    id,
    matchId: `m-${id}`,
    user: { userId: `u-${id}`, firstName: `User ${id}`, photoUrl: null },
    lastMessage: null,
    unreadCount: 0,
    createdAt: '2026-03-01T10:00:00Z',
    lastActivityAt: '2026-03-01T10:00:00Z',
    ...overrides,
  };
}

export function message(
  id: string,
  conversationId: string,
  overrides: Partial<Message> = {},
): Message {
  return {
    id,
    conversationId,
    senderId: `u-${conversationId}`,
    body: `body ${id}`,
    createdAt: '2026-03-02T10:00:00Z',
    ...overrides,
  };
}

export function notification(
  id: string,
  overrides: Partial<AppNotification> = {},
): AppNotification {
  return {
    id,
    type: 'new_like',
    title: 'Nouveau like',
    body: 'Quelqu’un vous a liké',
    data: {},
    isRead: false,
    createdAt: '2026-03-02T10:00:00Z',
    ...overrides,
  };
}
