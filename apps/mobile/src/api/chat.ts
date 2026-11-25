import { api } from "./client";
import type { Conversation, Message, MessagesPage } from "./types";

export const chatApi = {
  conversations: (offset = 0, limit = 30) =>
    api<{ conversations: Conversation[]; hasMore: boolean }>(`/conversations?limit=${limit}&offset=${offset}`),
  conversation: (conversationId: string) => api<Conversation>(`/conversations/${conversationId}`),
  unreadCount: async () => (await api<{ unreadCount: number }>("/conversations/unread-count")).unreadCount,
  messages: (conversationId: string, before?: string | null, limit = 30) => {
    const cursor = before ? `&before=${before}` : "";
    return api<MessagesPage>(`/conversations/${conversationId}/messages?limit=${limit}${cursor}`);
  },
  send: (conversationId: string, body: string) =>
    api<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } }),
  markRead: (conversationId: string) => api<{ readAt: string }>(`/conversations/${conversationId}/read`, { method: "POST" }),
  clear: (conversationId: string) => api<void>(`/conversations/${conversationId}`, { method: "DELETE" })
};
