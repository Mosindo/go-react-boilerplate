import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AppNotification } from "./types";

export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number; nextCursor?: string };

export function fetchNotifications(cursor?: string): Promise<NotificationsPage> {
  const query = cursor ? `?cursor=${encodeURIComponent(cursor)}` : "";
  return apiRequest<NotificationsPage>(`${endpoints.notifications}${query}`);
}

export function markNotificationRead(id: string): Promise<void> {
  return apiRequest(endpoints.notificationRead(id), { method: "POST", quiet: true });
}

export function markAllNotificationsRead(): Promise<void> {
  return apiRequest(endpoints.notificationsReadAll, { method: "POST", quiet: true });
}
