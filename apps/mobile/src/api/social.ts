import { apiRequest } from "./client";
import type {
  AppNotification,
  BlockedUser,
  Conversation,
  ConversationsPage,
  Message,
  MessagesPage,
  NotificationsPage,
  PublicProfile,
  ReportInput,
  SwipeAction,
  SwipeResult
} from "./types";

export const fetchDiscover = (limit = 10) =>
  apiRequest<{ profiles: PublicProfile[] }>("/discover", { query: { limit } }).then((r) => r.profiles);

export const swipe = (targetUserId: string, action: SwipeAction) =>
  apiRequest<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } });

export const resetPasses = () => apiRequest("/swipes/reset-passes", { method: "POST" });

export const fetchConversations = (cursor?: string | null, limit = 30) =>
  apiRequest<ConversationsPage>("/conversations", { query: { limit, cursor } });

export const fetchMessages = (conversationId: string, cursor?: string | null, limit = 30) =>
  apiRequest<MessagesPage>(`/conversations/${conversationId}/messages`, { query: { limit, cursor } });

export const sendMessage = (conversationId: string, body: string) =>
  apiRequest<Message>(`/conversations/${conversationId}/messages`, { method: "POST", body: { body } });

export const markConversationRead = (conversationId: string) =>
  apiRequest(`/conversations/${conversationId}/read`, { method: "POST" });

export const deleteConversation = (conversationId: string) =>
  apiRequest(`/conversations/${conversationId}`, { method: "DELETE" });

export const unmatch = (conversationId: string) =>
  apiRequest(`/conversations/${conversationId}/unmatch`, { method: "POST" });

export const fetchNotifications = (cursor?: string | null) =>
  apiRequest<NotificationsPage>("/notifications", { query: { limit: 30, cursor } });

export const markNotificationRead = (id: string) =>
  apiRequest(`/notifications/${id}/read`, { method: "POST" });

export const markAllNotificationsRead = () => apiRequest("/notifications/read-all", { method: "POST" });

export const blockUser = (userId: string) => apiRequest("/blocks", { method: "POST", body: { userId } });
export const unblockUser = (userId: string) => apiRequest(`/blocks/${userId}`, { method: "DELETE" });
export const fetchBlocks = () => apiRequest<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks);

export const reportUser = (input: ReportInput) =>
  apiRequest<{ id: string }>("/reports", { method: "POST", body: input });

export type { AppNotification, Conversation };
