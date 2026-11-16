import { apiRequest } from "./client";
import type {
  BlockedUser,
  DiscoverResponse,
  Match,
  MatchesResponse,
  Message,
  MessagesPage,
  NotificationsResponse,
  ReportReason,
  SwipeAction,
  SwipeResponse
} from "./types";

export const discoveryApi = {
  deck: (limit = 10) => apiRequest<DiscoverResponse>(`/discover?limit=${limit}`),
  swipe: (targetUserId: string, action: SwipeAction) => apiRequest<SwipeResponse>("/swipes", { method: "POST", body: { targetUserId, action } })
};

export const matchesApi = {
  list: (offset = 0, limit = 30) => apiRequest<MatchesResponse>(`/matches?limit=${limit}&offset=${offset}`),
  get: (matchId: string) => apiRequest<Match>(`/matches/${matchId}`),
  unmatch: (matchId: string) => apiRequest<void>(`/matches/${matchId}`, { method: "DELETE" })
};

export const chatApi = {
  messages: (matchId: string, before?: string | null, limit = 30) =>
    apiRequest<MessagesPage>(`/matches/${matchId}/messages?limit=${limit}${before ? `&before=${before}` : ""}`),
  send: (matchId: string, body: string, clientMessageId: string) =>
    apiRequest<Message>(`/matches/${matchId}/messages`, { method: "POST", body: { body, clientMessageId } }),
  markRead: (matchId: string) => apiRequest<void>(`/matches/${matchId}/read`, { method: "POST" })
};

export const safetyApi = {
  block: (userId: string) => apiRequest<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => apiRequest<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: () => apiRequest<{ blocks: BlockedUser[] }>("/blocks"),
  report: (userId: string, reason: ReportReason, details: string) => apiRequest<void>("/reports", { method: "POST", body: { userId, reason, details } })
};

export const notificationsApi = {
  list: (offset = 0) => apiRequest<NotificationsResponse>(`/notifications?limit=30&offset=${offset}`),
  markRead: (id: string) => apiRequest<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => apiRequest<void>("/notifications/read-all", { method: "POST" })
};

export const realtimeApi = {
  ticket: () => apiRequest<{ ticket: string; expiresInSeconds: number }>("/realtime/ticket", { method: "POST" })
};
