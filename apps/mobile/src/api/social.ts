import { apiRequest } from "./client";
import type {
  ConversationsPage,
  MatchItem,
  MatchesPage,
  Message,
  MessagesPage,
  NotificationsPage,
  Profile,
  ReportReason,
  UserSummary
} from "./types";

function withParams(path: string, params: Record<string, string | number | null | undefined>): string {
  const query = Object.entries(params)
    .filter(([, v]) => v !== null && v !== undefined)
    .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
    .join("&");
  return query ? `${path}?${query}` : path;
}

export const fetchDiscover = (limit = 10) =>
  apiRequest<{ profiles: Profile[] }>(withParams("/discover", { limit }));

export const sendSwipe = (targetUserId: string, action: "like" | "pass") =>
  apiRequest<{ matched: boolean; match: MatchItem | null }>("/swipes", {
    method: "POST",
    body: { targetUserId, action }
  });

export const fetchMatches = (cursor: string | null) =>
  apiRequest<MatchesPage>(withParams("/matches", { cursor, limit: 30 }));
export const unmatch = (matchId: string) => apiRequest(`/matches/${matchId}`, { method: "DELETE" });

export const fetchConversations = (cursor: string | null) =>
  apiRequest<ConversationsPage>(withParams("/conversations", { cursor, limit: 30 }));
export const fetchMessages = (conversationId: string, cursor: string | null) =>
  apiRequest<MessagesPage>(withParams(`/conversations/${conversationId}/messages`, { cursor, limit: 30 }));
export const sendMessage = (conversationId: string, body: string) =>
  apiRequest<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } });
export const markConversationRead = (conversationId: string) =>
  apiRequest(`/conversations/${conversationId}/read`, { method: "POST" });
export const hideConversation = (conversationId: string) =>
  apiRequest(`/conversations/${conversationId}`, { method: "DELETE" });

export const blockUser = (userId: string) => apiRequest("/blocks", { method: "POST", body: { userId } });
export const fetchBlocks = () => apiRequest<{ blocks: UserSummary[] }>("/blocks");
export const unblockUser = (userId: string) => apiRequest(`/blocks/${userId}`, { method: "DELETE" });
export const reportUser = (userId: string, reason: ReportReason, details: string) =>
  apiRequest("/reports", { method: "POST", body: { userId, reason, details } });

export const fetchNotifications = (cursor: string | null) =>
  apiRequest<NotificationsPage>(withParams("/notifications", { cursor, limit: 30 }));
export const markNotificationRead = (id: string) =>
  apiRequest(`/notifications/${id}/read`, { method: "POST" });
export const markAllNotificationsRead = () => apiRequest("/notifications/read-all", { method: "POST" });
