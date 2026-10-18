import { API_BASE_URL, apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { Interest } from "./types";

/** Media paths returned by the API are relative and carry a signed query string. */
export function resolveMediaUrl(path: string | null | undefined): string | null {
  if (!path) {
    return null;
  }
  return /^https?:\/\//.test(path) ? path : `${API_BASE_URL}${path}`;
}

export async function fetchInterests(): Promise<Interest[]> {
  const response = await apiRequest<{ interests: Interest[] }>(endpoints.profile.interests);
  return response.interests;
}

export async function fetchRealtimeUrl(): Promise<string> {
  const { ticket } = await apiRequest<{ ticket: string }>(endpoints.realtime.ticket, { method: "POST" });
  const wsBase = API_BASE_URL.replace(/^http/, "ws");
  return `${wsBase}${endpoints.realtime.socket}?ticket=${encodeURIComponent(ticket)}`;
}
