import { api } from './http';
import type { BlockItem, ReportReason } from './types';

export const safetyApi = {
  blocks: () => api.get<{ blocks: BlockItem[] }>('/blocks').then((r) => r.blocks),
  block: (userId: string) => api.post<void>('/blocks', { userId }),
  unblock: (userId: string) => api.delete<void>(`/blocks/${userId}`),
  report: (userId: string, reason: ReportReason, details?: string) =>
    api.post<{ id: string }>('/reports', {
      userId,
      reason,
      ...(details && details.trim() ? { details: details.trim() } : {}),
    }),
};
