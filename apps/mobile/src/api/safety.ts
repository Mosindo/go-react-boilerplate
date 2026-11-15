import { api } from "./client";
import type { BlockedUser, ReportReason } from "./types";

export const safetyApi = {
  block: (userId: string) => api<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => api<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocks: () => api<{ blocks: BlockedUser[] }>("/blocks"),
  report: (userId: string, reason: ReportReason, details: string) =>
    api<{ id: string }>("/reports", { method: "POST", body: { userId, reason, details } })
};
