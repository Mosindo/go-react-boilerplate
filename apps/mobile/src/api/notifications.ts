import { api } from './http';
import type { NotificationsPage } from './types';

export const notificationsApi = {
  list: (limit = 30, offset = 0, signal?: AbortSignal) =>
    api.get<NotificationsPage>('/notifications', { query: { limit, offset }, signal }),
  markRead: (id: string) => api.post<void>(`/notifications/${id}/read`),
  markAllRead: () => api.post<void>('/notifications/read-all'),
  wsTicket: () => api.post<{ ticket: string }>('/ws/ticket').then((r) => r.ticket),
};
