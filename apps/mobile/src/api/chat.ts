import type { ConversationSummary, Message } from "./models";
import { apiRequest } from "./client";

export const CONVERSATIONS_PAGE_SIZE = 30;
export const MESSAGES_PAGE_SIZE = 30;

export async function listConversations(
  before?: string,
  limit: number = CONVERSATIONS_PAGE_SIZE
): Promise<{ conversations: ConversationSummary[] }> {
  const query = new URLSearchParams({ limit: String(limit) });
  if (before) query.set("before", before);
  const res = await apiRequest<{ conversations: ConversationSummary[] | null }>(
    `/conversations?${query.toString()}`
  );
  return { conversations: res?.conversations ?? [] };
}

export async function listMessages(
  conversationId: string,
  before?: string,
  limit: number = MESSAGES_PAGE_SIZE
): Promise<{ messages: Message[]; nextCursor: string | null }> {
  const query = new URLSearchParams({ limit: String(limit) });
  if (before) query.set("before", before);
  const res = await apiRequest<{
    messages: Message[] | null;
    nextCursor: string | null;
  }>(`/conversations/${encodeURIComponent(conversationId)}/messages?${query.toString()}`);
  return { messages: res?.messages ?? [], nextCursor: res?.nextCursor ?? null };
}

export function sendMessage(conversationId: string, body: string): Promise<Message> {
  return apiRequest<Message>(`/conversations/${encodeURIComponent(conversationId)}/messages`, {
    method: "POST",
    body: JSON.stringify({ body })
  });
}

export async function markConversationRead(conversationId: string): Promise<void> {
  await apiRequest<void>(`/conversations/${encodeURIComponent(conversationId)}/read`, {
    method: "POST"
  });
}

/** Local deletion only: hides the conversation for me. */
export async function hideConversation(conversationId: string): Promise<void> {
  await apiRequest<void>(`/conversations/${encodeURIComponent(conversationId)}`, {
    method: "DELETE"
  });
}
