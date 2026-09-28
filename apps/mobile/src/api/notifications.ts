import { api } from "./http";
import { endpoints } from "./endpoints";
import type { NotificationPage } from "./types";

export function getNotifications(cursor?: string | null, limit = 30): Promise<NotificationPage> {
  return api.request<NotificationPage>(endpoints.notifications, { query: { cursor, limit } });
}

export function markNotificationRead(id: string): Promise<void> {
  return api.request(endpoints.notificationRead(id), { method: "POST" });
}

export function markAllNotificationsRead(): Promise<void> {
  return api.request(endpoints.notificationsReadAll, { method: "POST" });
}
