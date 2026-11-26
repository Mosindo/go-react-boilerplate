import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { listConversations, listMessages, unreadConversations } from "../api/chat";
import { likesReceived, listMatches } from "../api/matching";
import { listNotifications, unreadNotifications } from "../api/notifications";
import { getPublicProfile, listInterests } from "../api/profile";
import { listBlocked } from "../api/safety";
import { queryClient, queryKeys } from "./queryClient";
import { useRealtimeStatus } from "./useRealtime";

/** Without a live socket, fall back to light polling so chat still feels current. */
function usePollingFallback(ms: number): number | false {
  const { connected } = useRealtimeStatus();
  return connected ? false : ms;
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: listInterests, staleTime: Infinity });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: queryKeys.publicProfile(userId), queryFn: () => getPublicProfile(userId) });
}

export function useMatches() {
  return useInfiniteQuery({
    queryKey: queryKeys.matches,
    queryFn: ({ pageParam }) => listMatches(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useLikes() {
  return useQuery({
    queryKey: queryKeys.likes,
    queryFn: async () => {
      const profiles = await likesReceived();
      // The server marks the "someone liked you" notification as seen when the list is viewed.
      void queryClient.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      return profiles;
    }
  });
}

export function useConversations() {
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => listConversations(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: usePollingFallback(20_000)
  });
}

export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => listMessages(conversationId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    refetchInterval: usePollingFallback(8_000)
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => listNotifications(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useUnreadCounts() {
  const interval = usePollingFallback(30_000);
  const conversations = useQuery({ queryKey: queryKeys.unreadConversations, queryFn: unreadConversations, refetchInterval: interval });
  const notifications = useQuery({ queryKey: queryKeys.unreadNotifications, queryFn: unreadNotifications, refetchInterval: interval });
  return { conversations: conversations.data ?? 0, notifications: notifications.data ?? 0 };
}

export function useBlocked() {
  return useQuery({ queryKey: queryKeys.blocked, queryFn: listBlocked });
}
