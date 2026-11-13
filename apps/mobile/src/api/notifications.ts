import { request } from "./client";
import type { AppNotification, NotificationsPage } from "./types";

export const notificationsApi = {
  list: (offset = 0) => request<NotificationsPage>(`/notifications?limit=30&offset=${offset}`),
  markRead: (id: string) => request<AppNotification>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => request<void>("/notifications/read-all", { method: "POST" })
};
