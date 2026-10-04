import { api } from "./client";
import type { BlockedUser, ReportReason } from "./types";

export async function listBlocks(): Promise<BlockedUser[]> {
  const data = await api.get<{ blocks: BlockedUser[] }>("/blocks");
  return data.blocks;
}

export function blockUser(userId: string): Promise<void> {
  return api.post<void>("/blocks", { userId });
}

export function unblockUser(userId: string): Promise<void> {
  return api.delete<void>(`/blocks/${encodeURIComponent(userId)}`);
}

export type ReportInput = { userId: string; reason: ReportReason; details?: string; block?: boolean };

export function reportUser(input: ReportInput): Promise<{ id: string }> {
  return api.post<{ id: string }>("/reports", input);
}
