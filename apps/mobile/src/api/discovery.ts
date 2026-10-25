import { http } from "./client";
import type { PublicProfile, SwipeAction, SwipeResponse } from "./types";

export const discoveryApi = {
  list: async (limit = 10, signal?: AbortSignal) =>
    (await http.get<{ items: PublicProfile[] }>("/discovery", { limit }, signal)).items,
  swipe: (userId: string, action: SwipeAction) => http.post<SwipeResponse>("/discovery/swipes", { userId, action })
};
