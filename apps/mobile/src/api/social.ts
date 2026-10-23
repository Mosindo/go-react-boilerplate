import { request } from "./client";
import type {
  BlockedUser,
  Card,
  MatchesPage,
  Message,
  MessagesPage,
  NotificationsPage,
  ReportInput,
  SwipeAction,
  SwipeResult
} from "./types";

export const PAGE_SIZE = { matches: 20, messages: 30, notifications: 20, discover: 10 } as const;

export const getDiscover = async (limit: number = PAGE_SIZE.discover) =>
  (await request<{ profiles: Card[] }>("/discover", { query: { limit } })).profiles;

export const getCard = (userId: string) => request<Card>(`/profiles/${userId}`);

export const swipe = (targetUserId: string, action: SwipeAction) =>
  request<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } });

export const listMatches = (cursor: string | null) =>
  request<MatchesPage>("/matches", { query: { limit: PAGE_SIZE.matches, cursor } });
export const unmatch = (matchId: string) => request(`/matches/${matchId}`, { method: "DELETE" });
export const hideMatch = (matchId: string) =>
  request(`/matches/${matchId}/hide`, { method: "POST" });

export const listMessages = (matchId: string, cursor: string | null) =>
  request<MessagesPage>(`/matches/${matchId}/messages`, {
    query: { limit: PAGE_SIZE.messages, cursor }
  });
export const sendMessage = (matchId: string, content: string) =>
  request<Message>(`/matches/${matchId}/messages`, { method: "POST", body: { content } });
export const markMatchRead = (matchId: string) =>
  request(`/matches/${matchId}/read`, { method: "POST" });

export const listBlocks = async () => (await request<{ blocks: BlockedUser[] }>("/blocks")).blocks;
export const blockUser = (userId: string) =>
  request("/blocks", { method: "POST", body: { userId } });
export const unblockUser = (userId: string) => request(`/blocks/${userId}`, { method: "DELETE" });
export const report = (input: ReportInput) => request("/reports", { method: "POST", body: input });

export const listNotifications = (cursor: string | null) =>
  request<NotificationsPage>("/notifications", {
    query: { limit: PAGE_SIZE.notifications, cursor }
  });
export const markNotificationRead = (id: string) =>
  request(`/notifications/${id}/read`, { method: "POST" });
export const markAllNotificationsRead = () =>
  request("/notifications/read-all", { method: "POST" });
