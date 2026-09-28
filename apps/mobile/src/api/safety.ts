import { api } from "./http";
import { endpoints } from "./endpoints";
import type { BlockedUser, ReportReason } from "./types";

export function blockUser(userId: string): Promise<void> {
  return api.request(endpoints.blocks, { method: "POST", body: { userId } });
}

export function unblockUser(userId: string): Promise<void> {
  return api.request(endpoints.block(userId), { method: "DELETE" });
}

export async function getBlockedUsers(): Promise<BlockedUser[]> {
  const response = await api.request<{ items: BlockedUser[] }>(endpoints.blocks);
  return response.items;
}

export async function reportUser(userId: string, reason: ReportReason, details?: string): Promise<void> {
  await api.request<{ id: string }>(endpoints.reports, {
    method: "POST",
    body: { userId, reason, ...(details ? { details } : {}) }
  });
}
