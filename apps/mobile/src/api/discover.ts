import { api } from './http';
import type { MatchItem, PublicProfile, SwipeAction, SwipeResult } from './types';

export const discoverApi = {
  discover: (limit = 10, signal?: AbortSignal) =>
    api
      .get<{ profiles: PublicProfile[] }>('/discover', { query: { limit }, signal })
      .then((r) => r.profiles),
  profile: (userId: string, signal?: AbortSignal) =>
    api.get<PublicProfile>(`/profiles/${userId}`, { signal }),
  swipe: (userId: string, action: SwipeAction) =>
    api.post<SwipeResult>('/swipes', { userId, action }),
  likesReceived: (limit: number, offset: number, signal?: AbortSignal) =>
    api
      .get<{ profiles: PublicProfile[] }>('/likes/received', { query: { limit, offset }, signal })
      .then((r) => r.profiles),
  matches: (limit = 50, offset = 0) =>
    api
      .get<{ matches: MatchItem[] }>('/matches', { query: { limit, offset } })
      .then((r) => r.matches),
  unmatch: (matchId: string) => api.delete<void>(`/matches/${matchId}`),
};
