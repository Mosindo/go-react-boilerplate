import { http } from './client';
import type { PublicProfile, SwipeAction, SwipeResult } from './types';

export const discoverApi = {
  async batch(limit = 10): Promise<PublicProfile[]> {
    const res = await http.request<{ profiles: PublicProfile[] }>('GET', '/discover', {
      query: { limit },
    });
    return res.profiles;
  },

  swipe(userId: string, action: SwipeAction): Promise<SwipeResult> {
    return http.request<SwipeResult>('POST', '/swipes', { body: { userId, action } });
  },

  async likesReceived(): Promise<PublicProfile[]> {
    const res = await http.request<{ profiles: PublicProfile[] }>('GET', '/likes/received');
    return res.profiles;
  },

  unmatch(matchId: string): Promise<void> {
    return http.request<void>('DELETE', `/matches/${encodeURIComponent(matchId)}`);
  },
};
