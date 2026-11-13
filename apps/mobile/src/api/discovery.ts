import { request } from "./client";
import type { PublicProfile, SwipeAction, SwipeResult } from "./types";

export const discoveryApi = {
  next: (limit = 10) => request<{ profiles: PublicProfile[] }>(`/discovery?limit=${limit}`).then((r) => r.profiles),
  swipe: (targetUserId: string, action: SwipeAction) =>
    request<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  unmatch: (userId: string) => request<void>(`/matches/${userId}`, { method: "DELETE" })
};
