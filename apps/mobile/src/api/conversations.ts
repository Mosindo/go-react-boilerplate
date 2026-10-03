import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { ChatMessage, Conversation } from "./types";

export const MESSAGES_PAGE_SIZE = 30;

export async function listConversations(): Promise<Conversation[]> {
  const response = await apiRequest<{ conversations: Conversation[] }>(`${endpoints.conversations.list}?limit=50`, { silent: true });
  return response.conversations;
}

/** Newest-first page of messages, optionally older than `before`. */
export async function listMessages(matchId: string, before?: string): Promise<ChatMessage[]> {
  const query = `?limit=${MESSAGES_PAGE_SIZE}${before ? `&before=${encodeURIComponent(before)}` : ""}`;
  const response = await apiRequest<{ messages: ChatMessage[] }>(`${endpoints.conversations.messages(matchId)}${query}`, { silent: true });
  return response.messages;
}

export function sendMessage(matchId: string, body: string): Promise<ChatMessage> {
  return apiRequest<ChatMessage>(endpoints.conversations.messages(matchId), {
    method: "POST",
    body: JSON.stringify({ body }),
    silent: true
  });
}

export async function markConversationRead(matchId: string): Promise<void> {
  await apiRequest(endpoints.conversations.read(matchId), { method: "POST", silent: true });
}

export async function hideConversation(matchId: string): Promise<void> {
  await apiRequest(endpoints.conversations.hide(matchId), { method: "DELETE", silent: true });
}

export async function unmatch(matchId: string): Promise<void> {
  await apiRequest(endpoints.conversations.unmatch(matchId), { method: "DELETE", silent: true });
}
