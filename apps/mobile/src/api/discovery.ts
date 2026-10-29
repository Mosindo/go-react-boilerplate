import { request } from "./client";
import type { PublicProfile, SwipeAction, SwipeResult } from "./types";

export const fetchDiscover = async (limit = 10) =>
  (await request<{ profiles: PublicProfile[] }>("/discover", { query: { limit } })).profiles;

export const swipe = (userId: string, action: SwipeAction) =>
  request<SwipeResult>("/swipes", { method: "POST", body: { userId, action } });

export const unmatch = (matchId: string) =>
  request<void>(`/matches/${matchId}`, { method: "DELETE" });

export const unmatchConversation = (conversationId: string) =>
  request<void>(`/conversations/${conversationId}/match`, { method: "DELETE" });
