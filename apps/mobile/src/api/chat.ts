import { api } from "./client";
import type { Conversation, Message, MessagesPage, Page, ReadReceipt } from "./types";

export function listConversations(cursor?: string | null) {
  return api<Page<Conversation, "conversations">>(`/conversations${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`);
}

export function getConversation(conversationId: string) {
  return api<Conversation>(`/conversations/${conversationId}`);
}

export function listMessages(conversationId: string, cursor?: string | null) {
  const query = cursor ? `?limit=30&cursor=${encodeURIComponent(cursor)}` : "?limit=30";
  return api<MessagesPage>(`/conversations/${conversationId}/messages${query}`);
}

export function sendMessage(conversationId: string, body: string) {
  return api<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } });
}

export function markConversationRead(conversationId: string) {
  return api<ReadReceipt>(`/conversations/${conversationId}/read`, { method: "POST" });
}

export async function unreadConversations() {
  const res = await api<{ unreadCount: number }>("/conversations/unread-count");
  return res.unreadCount;
}
