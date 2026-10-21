import { apiRequest } from "./client";
import type { ConversationPage, Message, MessagePage } from "./types";

export const fetchConversations = () => apiRequest<ConversationPage>("/conversations", { query: { limit: 50 } });

export const fetchMessages = (conversationId: string, before?: number) =>
  apiRequest<MessagePage>(`/conversations/${conversationId}/messages`, { query: { limit: 30, before } });

export const sendMessage = (conversationId: string, body: string) =>
  apiRequest<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } });

export const markConversationRead = (conversationId: string) => apiRequest(`/conversations/${conversationId}/read`, { method: "POST" });

export const hideConversation = (conversationId: string) => apiRequest(`/conversations/${conversationId}`, { method: "DELETE" });
