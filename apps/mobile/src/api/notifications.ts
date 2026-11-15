import { api } from "./client";
import type { AppNotification, Badges, Page } from "./types";

export const notificationsApi = {
  list: (cursor?: string | null) =>
    api<Page<"notifications", AppNotification>>(`/notifications${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""}`),
  markRead: (id: string) => api<void>(`/notifications/${id}/read`, { method: "POST" }),
  markAllRead: () => api<void>("/notifications/read-all", { method: "POST" }),
  badges: () => api<Badges>("/me/badges")
};
