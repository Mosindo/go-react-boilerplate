import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { Match, PublicProfile, SwipeAction, SwipeResult } from "./types";

export async function fetchDiscover(limit = 10): Promise<PublicProfile[]> {
  return (await apiRequest<{ profiles: PublicProfile[] }>(`${endpoints.discover}?limit=${limit}`)).profiles;
}

export function swipe(userId: string, action: SwipeAction): Promise<SwipeResult> {
  return apiRequest<SwipeResult>(endpoints.swipes, { method: "POST", body: { userId, action }, quiet: true });
}

export function fetchMatches(cursor?: string): Promise<{ matches: Match[]; nextCursor?: string }> {
  const query = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
  return apiRequest(`${endpoints.matches}${query}`);
}

export function unmatch(matchId: string): Promise<void> {
  return apiRequest(endpoints.match(matchId), { method: "DELETE", quiet: true });
}
