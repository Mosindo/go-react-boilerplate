import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { ConversationsPage, Message, MessagesPage } from "./types";

function withCursor(path: string, cursor?: string, limit = 30): string {
  const params = new URLSearchParams({ limit: String(limit) });
  if (cursor) {
    params.set("cursor", cursor);
  }
  return `${path}?${params.toString()}`;
}

export function fetchConversations(cursor?: string): Promise<ConversationsPage> {
  return apiRequest<ConversationsPage>(withCursor(endpoints.chat.conversations, cursor));
}

export function fetchMessages(conversationId: string, cursor?: string): Promise<MessagesPage> {
  return apiRequest<MessagesPage>(withCursor(endpoints.chat.messages(conversationId), cursor));
}

export function sendMessage(conversationId: string, content: string): Promise<Message> {
  return apiRequest<Message>(endpoints.chat.messages(conversationId), { method: "POST", body: { content } });
}

export async function markConversationRead(conversationId: string): Promise<void> {
  await apiRequest(endpoints.chat.read(conversationId), { method: "POST" });
}

export async function hideConversation(conversationId: string): Promise<void> {
  await apiRequest(endpoints.chat.conversation(conversationId), { method: "DELETE" });
}
