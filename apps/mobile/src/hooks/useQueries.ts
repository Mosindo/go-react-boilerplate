import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { chatApi, isApiError, notificationsApi, profileApi, type OwnProfile } from "../api";
import { queryKeys } from "../lib/queryKeys";

/** The signed-in user's profile; `null` when onboarding has not started. */
export function useOwnProfile(enabled = true) {
  return useQuery<OwnProfile | null>({
    queryKey: queryKeys.profile,
    enabled,
    queryFn: async () => {
      try {
        return await profileApi.get();
      } catch (err) {
        if (isApiError(err, 404)) return null;
        throw err;
      }
    }
  });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: profileApi.interests, staleTime: Infinity });
}

export function useConversations() {
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    getNextPageParam: (last) => last.nextCursor,
    refetchOnMount: "always"
  });
}

export function useMessages(matchId: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.messages(matchId),
    initialPageParam: undefined as number | undefined,
    queryFn: ({ pageParam }) => chatApi.messages(matchId, pageParam),
    // Messages are newest first: the next page starts before the oldest id loaded.
    getNextPageParam: (last) => (last.hasMore ? last.messages[last.messages.length - 1]?.id : undefined),
    retry: (count, err) => !isApiError(err, 404) && count < 2
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useUnreadCount(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.unread,
    queryFn: notificationsApi.unreadCount,
    enabled,
    refetchInterval: 60_000
  });
}
