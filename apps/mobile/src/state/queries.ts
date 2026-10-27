import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { chatApi, discoveryApi, notificationsApi, profileApi } from "../api/endpoints";
import type { MyProfile } from "../api/types";
import { keys } from "./keys";

/** The caller's profile, or null when it has not been created yet (404). */
export function useMyProfile() {
  return useQuery<MyProfile | null>({
    queryKey: keys.profile,
    queryFn: async () => {
      try {
        return await profileApi.get();
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null;
        throw e;
      }
    },
    retry: (count, error) =>
      !(error instanceof ApiError && error.status < 500 && error.status !== 0) && count < 2
  });
}

export function useMatches() {
  return useInfiniteQuery({
    queryKey: keys.matches,
    queryFn: ({ pageParam }) => discoveryApi.matches(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useConversations() {
  return useInfiniteQuery({
    queryKey: keys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useBadgeCounts(): { messages: number; activity: number } {
  const conversations = useConversations();
  const notifications = useNotifications();
  const messages = (conversations.data?.pages[0]?.conversations ?? []).filter(
    (c) => c.unreadCount > 0
  ).length;
  return { messages, activity: notifications.data?.pages[0]?.unreadCount ?? 0 };
}
