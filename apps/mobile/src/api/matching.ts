import { api } from "./client";
import type { Page, ProfileCard, ReceivedLike, SwipeAction, SwipeResult } from "./types";

export const matchingApi = {
  discovery: (limit = 10) => api<{ profiles: ProfileCard[] }>(`/discovery?limit=${limit}`),
  swipe: (targetUserId: string, action: SwipeAction) => api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } }),
  likes: (cursor?: string | null) =>
    api<Page<"likes", ReceivedLike>>(`/likes/received${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`),
  unmatch: (matchId: string) => api<void>(`/matches/${matchId}`, { method: "DELETE" })
};
