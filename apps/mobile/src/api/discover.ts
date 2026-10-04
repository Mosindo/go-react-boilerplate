import { api } from "./client";
import type { MatchSummary, Page, PublicProfile, SwipeAction, SwipeResult } from "./types";

export async function fetchDiscover(limit = 10, signal?: AbortSignal): Promise<PublicProfile[]> {
  const data = await api.get<{ profiles: PublicProfile[] }>("/discover", { query: { limit }, signal });
  return data.profiles;
}

export function swipe(targetId: string, action: SwipeAction): Promise<SwipeResult> {
  return api.post<SwipeResult>("/swipes", { targetId, action });
}

export function fetchMatches(cursor?: string | null, limit = 30): Promise<Page<"matches", MatchSummary>> {
  return api.get("/matches", { query: { limit, cursor } });
}

export function unmatch(matchId: string): Promise<void> {
  return api.delete<void>(`/matches/${encodeURIComponent(matchId)}`);
}
