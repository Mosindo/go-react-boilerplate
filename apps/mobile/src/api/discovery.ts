import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { PublicProfile, SwipeAction, SwipeResult } from "./types";

export async function discover(limit = 10): Promise<PublicProfile[]> {
  const response = await apiRequest<{ profiles: PublicProfile[] }>(`${endpoints.discovery.discover}?limit=${limit}`, { silent: true });
  return response.profiles;
}

export function swipe(targetId: string, action: SwipeAction): Promise<SwipeResult> {
  return apiRequest<SwipeResult>(endpoints.discovery.swipe, {
    method: "POST",
    body: JSON.stringify({ targetId, action }),
    silent: true
  });
}
