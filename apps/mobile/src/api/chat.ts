import { api } from "./client";
import type { MatchSummary, Message, Page } from "./types";

export function fetchConversations(cursor?: string | null, limit = 30): Promise<Page<"conversations", MatchSummary>> {
  return api.get("/conversations", { query: { limit, cursor } });
}

export function fetchMessages(
  conversationId: string,
  cursor?: string | null,
  limit = 30
): Promise<Page<"messages", Message>> {
  return api.get(`/conversations/${encodeURIComponent(conversationId)}/messages`, { query: { limit, cursor } });
}

export function sendMessage(conversationId: string, body: string): Promise<Message> {
  return api.post<Message>(`/conversations/${encodeURIComponent(conversationId)}/messages`, { body });
}

export function markConversationRead(conversationId: string): Promise<void> {
  return api.post<void>(`/conversations/${encodeURIComponent(conversationId)}/read`);
}

export function hideConversation(conversationId: string): Promise<void> {
  return api.delete<void>(`/conversations/${encodeURIComponent(conversationId)}`);
}

export async function fetchConversationUnreadCount(): Promise<number> {
  const data = await api.get<{ unreadCount: number }>("/conversations/unread-count");
  return data.unreadCount;
}
