import { api } from "./client";
import type { BlockedUser, ReportReason } from "./types";

export const safetyApi = {
  blocks: async () => (await api<{ blocks: BlockedUser[] }>("/blocks")).blocks,
  block: (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" }),
  report: (userId: string, reason: ReportReason, details: string, block = true) =>
    api<{ id: string; blocked: boolean }>("/reports", { method: "POST", body: { userId, reason, details, block } })
};
