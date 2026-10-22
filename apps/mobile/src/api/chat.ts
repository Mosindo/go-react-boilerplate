import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { Conversation, Message } from "./types";

export function fetchConversations(cursor?: string): Promise<{ conversations: Conversation[]; nextCursor?: string }> {
  const query = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
  return apiRequest(`${endpoints.conversations}${query}`);
}

export function fetchMessages(conversationId: string, cursor?: string): Promise<{ messages: Message[]; nextCursor?: string }> {
  const query = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
  return apiRequest(`${endpoints.messages(conversationId)}${query}`);
}

export function sendMessage(conversationId: string, content: string): Promise<Message> {
  return apiRequest<Message>(endpoints.messages(conversationId), { method: "POST", body: { content }, quiet: true });
}

export function markConversationRead(conversationId: string): Promise<{ read: number }> {
  return apiRequest(endpoints.markRead(conversationId), { method: "POST", quiet: true });
}

export function hideConversation(conversationId: string): Promise<void> {
  return apiRequest(endpoints.conversation(conversationId), { method: "DELETE", quiet: true });
}
