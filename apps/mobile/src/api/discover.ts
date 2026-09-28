import { api } from "./http";
import { endpoints } from "./endpoints";
import type { Candidate, SwipeAction, SwipeResult } from "./types";

export async function getDiscoverBatch(limit = 10, signal?: AbortSignal): Promise<Candidate[]> {
  const response = await api.request<{ items: Candidate[] }>(endpoints.discover, { query: { limit }, signal });
  return response.items;
}

export function getCandidateProfile(userId: string): Promise<Candidate> {
  return api.request<Candidate>(endpoints.userProfile(userId));
}

export function swipe(targetUserId: string, action: SwipeAction): Promise<SwipeResult> {
  return api.request<SwipeResult>(endpoints.swipes, { method: "POST", body: { targetUserId, action } });
}
