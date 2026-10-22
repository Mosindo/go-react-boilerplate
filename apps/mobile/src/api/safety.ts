import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { BlockedUser, ReportReason } from "./types";

export function blockUser(userId: string): Promise<void> {
  return apiRequest(endpoints.blocks, { method: "POST", body: { userId }, quiet: true });
}

export function unblockUser(userId: string): Promise<void> {
  return apiRequest(endpoints.block(userId), { method: "DELETE", quiet: true });
}

export async function fetchBlocked(): Promise<BlockedUser[]> {
  return (await apiRequest<{ blocks: BlockedUser[] }>(endpoints.blocks)).blocks;
}

export function reportUser(userId: string, reason: ReportReason, details: string, alsoBlock: boolean): Promise<void> {
  return apiRequest(endpoints.reports, { method: "POST", body: { userId, reason, details, alsoBlock }, quiet: true });
}
