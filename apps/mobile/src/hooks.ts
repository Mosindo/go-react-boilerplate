import { useQuery } from "@tanstack/react-query";
import { chatApi, notificationsApi, profileApi } from "./api/endpoints";
import { queryKeys } from "./api/queryKeys";

export function useMyProfile() {
  return useQuery({ queryKey: queryKeys.profile, queryFn: profileApi.me, staleTime: 60_000 });
}

export function useInterests() {
  return useQuery({
    queryKey: queryKeys.interests,
    queryFn: async () => (await profileApi.interests()).interests,
    staleTime: 60 * 60_000
  });
}

export function useUnreadConversations() {
  return useQuery({
    queryKey: queryKeys.unreadConversations,
    queryFn: async () => (await chatApi.unreadCount()).unreadCount,
    staleTime: 15_000
  });
}

export function useUnreadNotifications() {
  return useQuery({
    queryKey: queryKeys.unreadNotifications,
    queryFn: async () => (await notificationsApi.unreadCount()).unreadCount,
    staleTime: 15_000
  });
}
