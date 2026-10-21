import { apiRequest } from "./client";
import type { NotificationPage } from "./types";

export const fetchNotifications = () => apiRequest<NotificationPage>("/notifications", { query: { limit: 50 } });
export const markNotificationsRead = (ids?: string[]) =>
  apiRequest("/notifications/read", { method: "POST", body: ids && ids.length ? { ids } : {} });
