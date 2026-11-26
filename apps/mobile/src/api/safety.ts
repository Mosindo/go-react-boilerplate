import { api } from "./client";
import type { BlockedUser, ReportReason } from "./types";

export function blockUser(userId: string) {
  return api<void>("/blocks", { method: "POST", body: { userId } });
}

export function unblockUser(userId: string) {
  return api<void>(`/blocks/${userId}`, { method: "DELETE" });
}

export async function listBlocked() {
  const res = await api<{ blocked: BlockedUser[] }>("/blocks");
  return res.blocked;
}

export function reportUser(userId: string, reason: ReportReason, details: string) {
  return api<void>("/reports", { method: "POST", body: { userId, reason, details } });
}
