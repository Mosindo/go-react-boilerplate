import { http } from "./client";
import type { NotificationsPage } from "./types";

export const notificationsApi = {
  list: (cursor?: string) => http.get<NotificationsPage>("/notifications", { limit: 30, cursor }),
  markRead: (id: string) => http.post(`/notifications/${encodeURIComponent(id)}/read`),
  markAllRead: () => http.post("/notifications/read-all")
};
