import { api } from "./http";
import { endpoints } from "./endpoints";
import type { Message, MatchSummary, Page } from "./types";

export function getMatches(cursor?: string | null, limit = 30): Promise<Page<MatchSummary>> {
  return api.request<Page<MatchSummary>>(endpoints.matches, { query: { cursor, limit } });
}

export function unmatch(matchId: string): Promise<void> {
  return api.request(endpoints.match(matchId), { method: "DELETE" });
}

export function getMessages(conversationId: string, before?: string | null, limit = 30): Promise<Page<Message>> {
  return api.request<Page<Message>>(endpoints.messages(conversationId), { query: { before, limit } });
}

export function sendMessage(conversationId: string, body: string): Promise<Message> {
  return api.request<Message>(endpoints.messages(conversationId), { method: "POST", body: { body } });
}

export function markConversationRead(conversationId: string): Promise<void> {
  return api.request(endpoints.markRead(conversationId), { method: "POST" });
}
