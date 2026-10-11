import { http } from './client';
import type { ConversationsPage, Message, MessagesPage } from './types';

export const chatApi = {
  conversations(cursor?: string | null): Promise<ConversationsPage> {
    return http.request<ConversationsPage>('GET', '/conversations', { query: { cursor } });
  },

  hide(conversationId: string): Promise<void> {
    return http.request<void>('DELETE', `/conversations/${encodeURIComponent(conversationId)}`);
  },

  messages(conversationId: string, before?: string | null, limit = 30): Promise<MessagesPage> {
    return http.request<MessagesPage>(
      'GET',
      `/conversations/${encodeURIComponent(conversationId)}/messages`,
      { query: { before, limit } },
    );
  },

  send(conversationId: string, body: string): Promise<Message> {
    return http.request<Message>(
      'POST',
      `/conversations/${encodeURIComponent(conversationId)}/messages`,
      { body: { body } },
    );
  },

  markRead(conversationId: string): Promise<void> {
    return http.request<void>('POST', `/conversations/${encodeURIComponent(conversationId)}/read`);
  },
};
