import { http } from './client';
import type { BlockedUser, ReportInput } from './types';

export const safetyApi = {
  block(userId: string): Promise<void> {
    return http.request<void>('POST', '/blocks', { body: { userId } });
  },

  unblock(userId: string): Promise<void> {
    return http.request<void>('DELETE', `/blocks/${encodeURIComponent(userId)}`);
  },

  async blocks(): Promise<BlockedUser[]> {
    const res = await http.request<{ blocks: BlockedUser[] }>('GET', '/blocks');
    return res.blocks;
  },

  report(input: ReportInput): Promise<{ id: string }> {
    return http.request<{ id: string }>('POST', '/reports', { body: input });
  },
};
