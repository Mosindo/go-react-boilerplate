import { request } from "./client";
import type { BlockedUser, ReportReason } from "./types";

export const blockUser = (userId: string) =>
  request<void>("/blocks", { method: "POST", body: { userId } });

export const unblockUser = (userId: string) =>
  request<void>(`/blocks/${userId}`, { method: "DELETE" });

export const fetchBlocked = async () =>
  (await request<{ blocks: BlockedUser[] }>("/blocks")).blocks;

export const reportUser = (userId: string, reason: ReportReason, details: string, block: boolean) =>
  request<void>("/reports", { method: "POST", body: { userId, reason, details, block } });
