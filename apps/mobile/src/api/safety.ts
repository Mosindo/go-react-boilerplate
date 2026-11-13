import { request } from "./client";
import type { BlockedUser, ReportReason } from "./types";

export const safetyApi = {
  block: (userId: string) => request<void>("/blocks", { method: "POST", body: { userId } }),
  unblock: (userId: string) => request<void>(`/blocks/${userId}`, { method: "DELETE" }),
  blocked: () => request<{ blocks: BlockedUser[] }>("/blocks").then((r) => r.blocks),
  report: (userId: string, reason: ReportReason, details: string) =>
    request<void>("/reports", { method: "POST", body: { userId, reason, details } })
};
