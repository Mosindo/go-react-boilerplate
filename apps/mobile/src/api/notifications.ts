import { apiRequest } from "./client";
import { endpoints } from "./endpoints";
import type { NotificationsPage } from "./types";

export function fetchNotifications(cursor?: string): Promise<NotificationsPage> {
  const params = new URLSearchParams({ limit: "20" });
  if (cursor) {
    params.set("cursor", cursor);
  }
  return apiRequest<NotificationsPage>(`${endpoints.notifications.list}?${params.toString()}`);
}

export async function markNotificationRead(id: string): Promise<void> {
  await apiRequest(endpoints.notifications.read(id), { method: "POST" });
}

export async function markAllNotificationsRead(): Promise<void> {
  await apiRequest(endpoints.notifications.readAll, { method: "POST" });
}
