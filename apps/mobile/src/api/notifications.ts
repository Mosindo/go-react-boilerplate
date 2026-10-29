import { request } from "./client";
import type { NotificationsPage } from "./types";

export const fetchNotifications = (before?: string) =>
  request<NotificationsPage>("/notifications", { query: { limit: 30, before } });

export const markNotificationRead = (id: string) =>
  request<void>(`/notifications/${id}/read`, { method: "POST" });

export const markAllNotificationsRead = () =>
  request<void>("/notifications/read-all", { method: "POST" });
