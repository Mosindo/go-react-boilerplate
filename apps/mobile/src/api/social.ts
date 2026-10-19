import { request } from "./client";
import type {
  BlockedUser,
  Conversation,
  ConversationsPage,
  LikedBy,
  Message,
  MessagesPage,
  NotificationsPage,
  PublicProfile,
  ReportReason,
  SwipeAction,
  SwipeResult
} from "./types";

// discovery & matching
export const discover = async (limit = 10) =>
  (await request<{ profiles: PublicProfile[] }>("/discover", { query: { limit } })).profiles;

export const swipe = (userId: string, action: SwipeAction) =>
  request<SwipeResult>("/swipes", { method: "POST", body: { userId, action } });

export const receivedLikes = (cursor?: string) =>
  request<{ profiles: LikedBy[]; nextCursor?: string }>("/likes/received", { query: { limit: 30, cursor } });

export const unmatch = (matchId: string) => request(`/matches/${matchId}`, { method: "DELETE" });

// chat
export const listConversations = (cursor?: string) =>
  request<ConversationsPage>("/conversations", { query: { limit: 30, cursor } });

export const getConversation = (id: string) => request<Conversation>(`/conversations/${id}`);

export const listMessages = (id: string, cursor?: string) =>
  request<MessagesPage>(`/conversations/${id}/messages`, { query: { limit: 30, cursor } });

export const sendMessage = (id: string, body: string) =>
  request<Message>(`/conversations/${id}/messages`, { method: "POST", body: { body } });

export const markRead = (id: string) => request(`/conversations/${id}/read`, { method: "POST" });

export const hideConversation = (id: string) => request(`/conversations/${id}`, { method: "DELETE" });

// notifications
export const listNotifications = (cursor?: string) =>
  request<NotificationsPage>("/notifications", { query: { limit: 30, cursor } });

export const markNotificationRead = (id: string) => request(`/notifications/${id}/read`, { method: "POST" });

export const markAllNotificationsRead = () => request("/notifications/read-all", { method: "POST" });

// safety
export const blockUser = (userId: string) => request("/blocks", { method: "POST", body: { userId } });
export const unblockUser = (userId: string) => request(`/blocks/${userId}`, { method: "DELETE" });
export const listBlocks = async () => (await request<{ blocks: BlockedUser[] }>("/blocks")).blocks;
export const reportUser = (userId: string, reason: ReportReason, details: string) =>
  request("/reports", { method: "POST", body: { userId, reason, details } });
