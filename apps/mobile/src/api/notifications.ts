import { http } from './client';
import type { NotificationsPage } from './types';

export const notificationsApi = {
  list(cursor?: string | null): Promise<NotificationsPage> {
    return http.request<NotificationsPage>('GET', '/notifications', { query: { cursor } });
  },

  markRead(id: string): Promise<void> {
    return http.request<void>('POST', `/notifications/${encodeURIComponent(id)}/read`);
  },

  markAllRead(): Promise<void> {
    return http.request<void>('POST', '/notifications/read-all');
  },
};
