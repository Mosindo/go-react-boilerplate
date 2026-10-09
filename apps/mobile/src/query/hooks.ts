import { useQuery } from '@tanstack/react-query';
import { chatApi } from '../api/chat';
import { notificationsApi } from '../api/notifications';
import { profileApi } from '../api/profile';
import { discoverApi } from '../api/discover';
import { safetyApi } from '../api/safety';
import { qk } from './keys';
import { totalUnread } from '../realtime/cacheUpdaters';

export function useMyProfile(enabled = true) {
  return useQuery({
    queryKey: qk.myProfile,
    queryFn: ({ signal }) => profileApi.getProfile(signal),
    enabled,
  });
}

export function usePreferences(enabled = true) {
  return useQuery({ queryKey: qk.preferences, queryFn: profileApi.getPreferences, enabled });
}

export function useInterestCatalog() {
  return useQuery({ queryKey: qk.interests, queryFn: profileApi.interests, staleTime: Infinity });
}

export function useConversations() {
  return useQuery({
    queryKey: qk.conversations,
    queryFn: ({ signal }) => chatApi.conversations(50, 0, signal),
  });
}

export function useUnreadMessages(): number {
  return totalUnread(useConversations().data);
}

export function useNotifications() {
  return useQuery({
    queryKey: qk.notifications,
    queryFn: ({ signal }) => notificationsApi.list(50, 0, signal),
  });
}

export function useUnreadNotifications(): number {
  return useNotifications().data?.unreadCount ?? 0;
}

export function useBlocks() {
  return useQuery({ queryKey: qk.blocks, queryFn: safetyApi.blocks });
}

export function useProfileById(userId: string) {
  return useQuery({
    queryKey: qk.profile(userId),
    queryFn: ({ signal }) => discoverApi.profile(userId, signal),
    retry: false,
  });
}
