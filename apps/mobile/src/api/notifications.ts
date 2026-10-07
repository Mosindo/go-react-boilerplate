import { apiRequest } from "./client";
import type { AppNotification, NotificationsPayload } from "../features/types";

export function fetchNotifications(): Promise<NotificationsPayload> {
  return apiRequest<NotificationsPayload>("/notifications");
}

export function markNotificationRead(id: string): Promise<AppNotification> {
  return apiRequest<AppNotification>(`/notifications/${encodeURIComponent(id)}/read`, { method: "POST" });
}

export async function markAllNotificationsRead(): Promise<void> {
  await apiRequest<void>("/notifications/read-all", { method: "POST" });
}
