import { api } from './http';
import type { Conversation, Message, MessagesPage } from './types';

export const chatApi = {
  conversations: (limit = 50, offset = 0, signal?: AbortSignal) =>
    api
      .get<{ conversations: Conversation[] }>('/conversations', {
        query: { limit, offset },
        signal,
      })
      .then((r) => r.conversations),
  messages: (conversationId: string, before?: string, limit = 30, signal?: AbortSignal) =>
    api.get<MessagesPage>(`/conversations/${conversationId}/messages`, {
      query: { before, limit },
      signal,
    }),
  send: (conversationId: string, body: string) =>
    api.post<Message>(`/conversations/${conversationId}/messages`, { body }),
  markRead: (conversationId: string) => api.post<void>(`/conversations/${conversationId}/read`),
  remove: (conversationId: string) => api.delete<void>(`/conversations/${conversationId}`),
};
