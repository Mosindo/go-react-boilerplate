import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { ReportReason } from "./models";

export type BlockedUser = { userId: string; firstName: string; blockedAt: string };

export type ReportInput = {
  userId: string;
  reason: ReportReason;
  details?: string;
  block?: boolean;
};

export async function blockUser(userId: string): Promise<void> {
  await apiRequest<void>(endpoints.safety.blocks, {
    method: "POST",
    body: JSON.stringify({ userId })
  });
}

export async function unblockUser(userId: string): Promise<void> {
  await apiRequest<void>(endpoints.safety.unblock(userId), { method: "DELETE" });
}

export async function listBlocks(): Promise<BlockedUser[]> {
  const payload = await apiRequest<{ blocks: BlockedUser[] }>(endpoints.safety.blocks);
  return payload.blocks;
}

export async function reportUser(input: ReportInput): Promise<void> {
  await apiRequest<{ id: string }>(endpoints.safety.reports, {
    method: "POST",
    body: JSON.stringify(input)
  });
}
