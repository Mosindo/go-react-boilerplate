import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { chatApi, notificationsApi, profileApi } from "../api/endpoints";

export const keys = {
  profile: ["profile", "me"] as const,
  interests: ["interests"] as const,
  conversations: ["conversations"] as const,
  messages: (id: string) => ["messages", id] as const,
  notifications: ["notifications"] as const,
  likes: ["likes"] as const,
  blocked: ["blocked"] as const,
  publicProfile: (id: string) => ["profile", id] as const,
};

export function useOwnProfile() {
  return useQuery({ queryKey: keys.profile, queryFn: profileApi.me });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: 60 * 60_000 });
}

export function useConversations() {
  return useInfiniteQuery({
    queryKey: keys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
  });
}

export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: keys.messages(conversationId),
    queryFn: ({ pageParam }) => chatApi.messages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore,
  });
}
