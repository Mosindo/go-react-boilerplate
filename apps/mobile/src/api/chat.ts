import { apiRequest } from "./client";
import type { Conversation, Message, MessagesPage } from "../features/types";

export async function fetchConversations(): Promise<Conversation[]> {
  const res = await apiRequest<{ items: Conversation[] }>("/conversations");
  return res.items;
}

export function fetchMessages(
  conversationId: string,
  before?: string | null,
  limit = 30
): Promise<MessagesPage> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (before) {
    params.set("before", before);
  }
  return apiRequest<MessagesPage>(
    `/conversations/${encodeURIComponent(conversationId)}/messages?${params.toString()}`
  );
}

export function sendMessage(conversationId: string, content: string): Promise<Message> {
  return apiRequest<Message>(`/conversations/${encodeURIComponent(conversationId)}/messages`, {
    method: "POST",
    body: JSON.stringify({ content })
  });
}

export async function markConversationRead(conversationId: string): Promise<void> {
  await apiRequest<void>(`/conversations/${encodeURIComponent(conversationId)}/read`, { method: "POST" });
}

export async function hideConversation(conversationId: string): Promise<void> {
  await apiRequest<void>(`/conversations/${encodeURIComponent(conversationId)}`, { method: "DELETE" });
}

export type RealtimeTicket = { ticket: string; expiresIn: number };

export function fetchRealtimeTicket(): Promise<RealtimeTicket> {
  return apiRequest<RealtimeTicket>("/realtime/ticket", { method: "POST" });
}
