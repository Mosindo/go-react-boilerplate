import { api } from "./client";
import type { DeckResponse, Match, Page, PublicProfile, SwipeAction, SwipeResult } from "./types";

export function getDeck(limit = 10) {
  return api<DeckResponse>(`/discover?limit=${limit}`);
}

export function swipe(targetUserId: string, action: SwipeAction) {
  return api<SwipeResult>("/swipes", { method: "POST", body: { targetUserId, action } });
}

export async function likesReceived() {
  const res = await api<{ profiles: PublicProfile[] }>("/likes/received");
  return res.profiles;
}

export function listMatches(cursor?: string | null) {
  return api<Page<Match, "matches">>(`/matches${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`);
}

export function unmatch(matchId: string) {
  return api<void>(`/matches/${matchId}`, { method: "DELETE" });
}
