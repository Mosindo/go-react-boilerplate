import { api } from "./client";
import type { AppNotification, Page } from "./types";

export function fetchNotifications(
  cursor?: string | null,
  limit = 30
): Promise<Page<"notifications", AppNotification>> {
  return api.get("/notifications", { query: { limit, cursor } });
}

export async function fetchNotificationUnreadCount(): Promise<number> {
  const data = await api.get<{ unreadCount: number }>("/notifications/unread-count");
  return data.unreadCount;
}

export function markNotificationRead(id: string): Promise<void> {
  return api.post<void>(`/notifications/${encodeURIComponent(id)}/read`);
}

export function markAllNotificationsRead(): Promise<void> {
  return api.post<void>("/notifications/read-all");
}
