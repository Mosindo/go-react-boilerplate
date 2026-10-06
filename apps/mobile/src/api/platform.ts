import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AppNotification, BlockedUser, NotificationsPage, ReportReason } from "./types";

export const listNotifications = (limit = 30, offset = 0) =>
  apiRequest<NotificationsPage>(endpoints.notifications, { query: { limit, offset } });

export const markNotificationRead = (id: string) => apiRequest<AppNotification>(endpoints.notificationRead(id), { method: "POST" });

export const markAllNotificationsRead = () => apiRequest(endpoints.notificationsReadAll, { method: "POST" });

export const listBlocks = async (): Promise<BlockedUser[]> => (await apiRequest<{ blocks: BlockedUser[] }>(endpoints.blocks)).blocks;

export const blockUser = (userId: string) => apiRequest(endpoints.blocks, { method: "POST", body: { userId } });

export const unblockUser = (userId: string) => apiRequest(endpoints.block(userId), { method: "DELETE" });

export const reportUser = (input: { userId: string; reason: ReportReason; details: string; alsoBlock: boolean }) =>
  apiRequest(endpoints.reports, { method: "POST", body: input });

export const deleteAccount = (password: string) => apiRequest(endpoints.account, { method: "DELETE", body: { password } });
