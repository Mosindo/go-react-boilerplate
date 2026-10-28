import { useInfiniteQuery, useQuery, type InfiniteData } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { keys } from "../api/keys";
import { fetchInterests, fetchPreferences, fetchProfile } from "../api/profile";
import { fetchConversations, fetchMessages, fetchNotifications } from "../api/social";
import type { ConversationsPage, MessagesPage, NotificationsPage, OwnProfile } from "../api/types";

export function useOwnProfile() {
  return useQuery<OwnProfile | null, Error>({
    queryKey: keys.profile,
    queryFn: async () => {
      try {
        return await fetchProfile();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) return null;
        throw error;
      }
    }
  });
}

export const usePreferences = () => useQuery({ queryKey: keys.preferences, queryFn: fetchPreferences });
export const useInterests = () => useQuery({ queryKey: keys.interests, queryFn: fetchInterests, staleTime: Infinity });

export function useConversations() {
  return useInfiniteQuery<ConversationsPage, Error, InfiniteData<ConversationsPage, string | null>, readonly string[], string | null>({
    queryKey: keys.conversations,
    initialPageParam: null,
    queryFn: ({ pageParam }) => fetchConversations(pageParam),
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useMessages(conversationId: string) {
  return useInfiniteQuery<MessagesPage, Error, InfiniteData<MessagesPage, string | null>, readonly string[], string | null>({
    queryKey: keys.messages(conversationId),
    initialPageParam: null,
    queryFn: ({ pageParam }) => fetchMessages(conversationId, pageParam),
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useNotifications() {
  return useInfiniteQuery<NotificationsPage, Error, InfiniteData<NotificationsPage, string | null>, readonly string[], string | null>({
    queryKey: keys.notifications,
    initialPageParam: null,
    queryFn: ({ pageParam }) => fetchNotifications(pageParam),
    getNextPageParam: (last) => last.nextCursor
  });
}
