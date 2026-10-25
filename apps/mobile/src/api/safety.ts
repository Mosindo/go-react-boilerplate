import { http } from "./client";
import type { BlockedUser, Page, ReportReason } from "./types";

export const safetyApi = {
  blocks: (cursor?: string) => http.get<Page<BlockedUser>>("/blocks", { limit: 30, cursor }),
  block: (userId: string) => http.post("/blocks", { userId }),
  unblock: (userId: string) => http.delete(`/blocks/${encodeURIComponent(userId)}`),
  report: (input: { userId: string; reason: ReportReason; details?: string }) =>
    http.post<{ id: string }>("/reports", input)
};
