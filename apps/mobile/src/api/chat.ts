import { request } from "./client";
import type { ConversationsPage, Message, MessagesPage } from "./types";

export const fetchConversations = (before?: string) =>
  request<ConversationsPage>("/conversations", { query: { limit: 30, before } });

export const fetchMessages = (conversationId: string, before?: number) =>
  request<MessagesPage>(`/conversations/${conversationId}/messages`, {
    query: { limit: 40, before }
  });

export const sendMessage = (conversationId: string, body: string) =>
  request<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } });

export const markConversationRead = (conversationId: string) =>
  request<{ readCount: number }>(`/conversations/${conversationId}/read`, { method: "POST" });

export const hideConversation = (conversationId: string) =>
  request<void>(`/conversations/${conversationId}`, { method: "DELETE" });
