import { api } from "./client";
import type { PublicProfile, SwipeAction, SwipeResult } from "./types";

export const matchingApi = {
  discover: async (limit = 10) => (await api<{ profiles: PublicProfile[] }>(`/discover?limit=${limit}`)).profiles,
  swipe: (targetUserId: string, action: SwipeAction) => api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  unmatch: (matchId: string) => api<void>(`/matches/${matchId}`, { method: "DELETE" })
};
