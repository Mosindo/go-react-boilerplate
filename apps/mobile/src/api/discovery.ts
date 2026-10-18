import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { PublicProfile, SwipeResult } from "./types";

export async function fetchDiscover(limit = 10): Promise<PublicProfile[]> {
  const response = await apiRequest<{ profiles: PublicProfile[] }>(`${endpoints.discovery.discover}?limit=${limit}`);
  return response.profiles;
}

export function swipe(userId: string, action: "like" | "pass"): Promise<SwipeResult> {
  return apiRequest<SwipeResult>(endpoints.discovery.swipes, { method: "POST", body: { userId, action } });
}

export async function resetPasses(): Promise<number> {
  const response = await apiRequest<{ reset: number }>(endpoints.discovery.resetPasses, { method: "DELETE" });
  return response.reset;
}

export async function unmatch(matchId: string): Promise<void> {
  await apiRequest(endpoints.discovery.unmatch(matchId), { method: "DELETE" });
}
