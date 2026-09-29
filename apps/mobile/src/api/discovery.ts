import type { PublicProfile } from "./models";
import { apiRequest } from "./client";

export const DISCOVER_BATCH_SIZE = 10;

export async function fetchDiscover(
  limit: number = DISCOVER_BATCH_SIZE,
  signal?: AbortSignal
): Promise<PublicProfile[]> {
  const res = await apiRequest<{ profiles: PublicProfile[] | null }>(`/discover?limit=${limit}`, {
    signal
  });
  return res?.profiles ?? [];
}

export function fetchPublicProfile(userId: string, signal?: AbortSignal): Promise<PublicProfile> {
  return apiRequest<PublicProfile>(`/profiles/${encodeURIComponent(userId)}`, {
    signal
  });
}
