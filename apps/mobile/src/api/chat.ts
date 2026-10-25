import { http } from "./client";
import type { ConversationItem, Message, Page } from "./types";

const conversationPath = (id: string) => `/conversations/${encodeURIComponent(id)}`;

export const chatApi = {
  conversations: (cursor?: string) => http.get<Page<ConversationItem>>("/conversations", { limit: 30, cursor }),
  messages: (conversationId: string, cursor?: string) =>
    http.get<Page<Message>>(`${conversationPath(conversationId)}/messages`, { limit: 30, cursor }),
  send: (conversationId: string, body: string) =>
    http.post<Message>(`${conversationPath(conversationId)}/messages`, { body }),
  markRead: (conversationId: string) => http.post(`${conversationPath(conversationId)}/read`),
  hide: (conversationId: string) => http.delete(conversationPath(conversationId)),
  unreadCount: async () => (await http.get<{ unreadCount: number }>("/conversations/unread-count")).unreadCount
};
