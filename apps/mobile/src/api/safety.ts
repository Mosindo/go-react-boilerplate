import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { BlockedUser, ReportReason } from "./types";

export async function fetchBlocks(): Promise<BlockedUser[]> {
  const response = await apiRequest<{ blocks: BlockedUser[] }>(endpoints.safety.blocks);
  return response.blocks;
}

export async function blockUser(userId: string): Promise<void> {
  await apiRequest(endpoints.safety.blocks, { method: "POST", body: { userId } });
}

export async function unblockUser(userId: string): Promise<void> {
  await apiRequest(endpoints.safety.unblock(userId), { method: "DELETE" });
}

export async function reportUser(userId: string, reason: ReportReason, details: string, block: boolean): Promise<void> {
  await apiRequest(endpoints.safety.reports, { method: "POST", body: { userId, reason, details, block } });
}
