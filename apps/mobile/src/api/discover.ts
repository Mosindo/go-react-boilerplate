import { apiRequest } from "./client";
import type { Match, Photo, PublicProfile, SwipeAction, SwipeResult } from "../features/types";

export async function fetchDiscover(limit = 10, signal?: AbortSignal): Promise<PublicProfile[]> {
  const res = await apiRequest<{ items: PublicProfile[] }>(`/discover?limit=${limit}`, { signal });
  return res.items;
}

export function fetchUserProfile(userId: string): Promise<PublicProfile> {
  return apiRequest<PublicProfile>(`/users/${encodeURIComponent(userId)}/profile`);
}

export function postSwipe(targetUserId: string, action: SwipeAction): Promise<SwipeResult> {
  return apiRequest<SwipeResult>("/swipes", {
    method: "POST",
    body: JSON.stringify({ targetUserId, action })
  });
}

export async function fetchMatches(): Promise<Match[]> {
  const res = await apiRequest<{ items: Match[] }>("/matches");
  return res.items;
}

export async function unmatch(matchId: string): Promise<void> {
  await apiRequest<void>(`/matches/${encodeURIComponent(matchId)}`, { method: "DELETE" });
}

/** Main (position 0) photo of the signed-in user, used by the match overlay. */
export async function fetchMyMainPhoto(): Promise<Photo | null> {
  const res = await apiRequest<{ items: Photo[] }>("/photos");
  return res.items.length > 0 ? res.items[0] : null;
}
