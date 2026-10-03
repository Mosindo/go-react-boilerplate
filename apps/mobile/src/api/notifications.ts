import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { AppNotification } from "./types";

export type NotificationsPage = { notifications: AppNotification[]; unreadCount: number };

export function listNotifications(): Promise<NotificationsPage> {
  return apiRequest<NotificationsPage>(`${endpoints.notifications.list}?limit=50`, { silent: true });
}

export async function markNotificationRead(id: string): Promise<void> {
  await apiRequest(endpoints.notifications.read(id), { method: "POST", silent: true });
}

export async function markAllNotificationsRead(): Promise<void> {
  await apiRequest(endpoints.notifications.readAll, { method: "POST", silent: true });
}
