import { apiRequest } from "./client";
import type { ReportReason, UserSummary } from "../features/types";

export async function fetchBlocks(): Promise<UserSummary[]> {
  const res = await apiRequest<{ items: UserSummary[] }>("/blocks");
  return res.items;
}

export async function blockUser(userId: string): Promise<void> {
  await apiRequest<void>("/blocks", { method: "POST", body: JSON.stringify({ userId }) });
}

export async function unblockUser(userId: string): Promise<void> {
  await apiRequest<void>(`/blocks/${encodeURIComponent(userId)}`, { method: "DELETE" });
}

export function reportUser(
  targetUserId: string,
  reason: ReportReason,
  details?: string
): Promise<{ id: string }> {
  const trimmed = details?.trim();
  return apiRequest<{ id: string }>("/reports", {
    method: "POST",
    body: JSON.stringify(trimmed ? { targetUserId, reason, details: trimmed } : { targetUserId, reason })
  });
}
