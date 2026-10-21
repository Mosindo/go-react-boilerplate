import { apiRequest } from "./client";
import type { BlockedUser } from "./types";

export const blockUser = (userId: string) => apiRequest(`/blocks/${userId}`, { method: "PUT" });
export const unblockUser = (userId: string) => apiRequest(`/blocks/${userId}`, { method: "DELETE" });
export const fetchBlocks = () => apiRequest<{ items: BlockedUser[] }>("/blocks").then((r) => r.items);
export const reportUser = (userId: string, reason: string, details: string, alsoBlock: boolean) =>
  apiRequest("/reports", { method: "POST", body: { userId, reason, details, alsoBlock } });
