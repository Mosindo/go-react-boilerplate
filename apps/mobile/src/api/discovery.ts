import { apiRequest } from "./client";
import type { Match, PublicProfile, SwipeAction, SwipeResult } from "./types";

export const fetchDiscover = (limit = 10) => apiRequest<{ items: PublicProfile[] }>("/discover", { query: { limit } }).then((r) => r.items);

export const swipe = (targetId: string, action: SwipeAction) =>
  apiRequest<SwipeResult>("/swipes", { method: "POST", body: { targetId, action } });

export const resetPasses = () => apiRequest("/swipes/passes", { method: "DELETE" });

export const fetchMatches = () => apiRequest<{ items: Match[] }>("/matches", { query: { limit: 50 } }).then((r) => r.items);
export const unmatch = (matchId: string) => apiRequest(`/matches/${matchId}`, { method: "DELETE" });
