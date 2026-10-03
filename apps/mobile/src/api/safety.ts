import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { BlockedUser, ReportReason } from "./types";

export async function blockUser(userId: string): Promise<void> {
  await apiRequest(endpoints.safety.blocks, { method: "POST", body: JSON.stringify({ userId }), silent: true });
}

export async function unblockUser(userId: string): Promise<void> {
  await apiRequest(endpoints.safety.unblock(userId), { method: "DELETE", silent: true });
}

export async function listBlocks(): Promise<BlockedUser[]> {
  const response = await apiRequest<{ blocks: BlockedUser[] }>(endpoints.safety.blocks, { silent: true });
  return response.blocks;
}

export async function reportUser(userId: string, reason: ReportReason, details: string): Promise<void> {
  await apiRequest(endpoints.safety.reports, { method: "POST", body: JSON.stringify({ userId, reason, details }), silent: true });
}
