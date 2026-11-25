import { api } from "./client";
import type { NotificationsPage } from "./types";

export const notificationsApi = {
  list: (before?: string | null, limit = 20) => api<NotificationsPage>(`/notifications?limit=${limit}${before ? `&before=${before}` : ""}`),
  unreadCount: async () => (await api<{ unreadCount: number }>("/notifications/unread-count")).unreadCount,
  markRead: (id: string) => api<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api<void>("/notifications/read-all", { method: "POST" })
};
