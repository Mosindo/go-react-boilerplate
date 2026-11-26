import { api } from "./client";
import type { NotificationsPage } from "./types";

export function listNotifications(cursor?: string | null) {
  return api<NotificationsPage>(`/notifications${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`);
}

export async function unreadNotifications() {
  const res = await api<{ unreadCount: number }>("/notifications/unread-count");
  return res.unreadCount;
}

export function markNotificationRead(id: string) {
  return api<void>(`/notifications/${id}/read`, { method: "POST" });
}

export function markAllNotificationsRead() {
  return api<void>("/notifications/read-all", { method: "POST" });
}
