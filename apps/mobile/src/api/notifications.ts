import type { AppNotification } from "./models";
import { apiRequest } from "./client";

export const NOTIFICATIONS_PAGE_SIZE = 30;

export async function listNotifications(
  before?: string,
  limit: number = NOTIFICATIONS_PAGE_SIZE
): Promise<{ notifications: AppNotification[]; unreadCount: number }> {
  const query = new URLSearchParams({ limit: String(limit) });
  if (before) query.set("before", before);
  const res = await apiRequest<{ notifications: AppNotification[] | null; unreadCount: number }>(
    `/notifications?${query.toString()}`
  );
  return { notifications: res?.notifications ?? [], unreadCount: res?.unreadCount ?? 0 };
}

export async function markNotificationRead(id: string): Promise<void> {
  await apiRequest<void>(`/notifications/${encodeURIComponent(id)}/read`, { method: "POST" });
}

export async function markAllNotificationsRead(): Promise<void> {
  await apiRequest<void>("/notifications/read-all", { method: "POST" });
}
