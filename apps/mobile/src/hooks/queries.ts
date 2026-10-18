import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { fetchBlocks } from "../api/safety";
import { fetchConversations, fetchMessages } from "../api/chat";
import { ApiError } from "../api/client";
import { fetchNotifications } from "../api/notifications";
import { fetchInterests } from "../api/platform";
import { getMyProfile } from "../api/profile";
import type { MyProfile } from "../api/types";

export const queryKeys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  discover: ["discover"] as const,
  person: (userId: string) => ["person", userId] as const
};

/** Resolves to null when the user has not created a profile yet (onboarding). */
export function useMyProfile() {
  return useQuery<MyProfile | null>({
    queryKey: queryKeys.profile,
    queryFn: async () => {
      try {
        return await getMyProfile();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return null;
        }
        throw error;
      }
    }
  });
}

export function useInterests() {
  return useQuery({ queryKey: queryKeys.interests, queryFn: fetchInterests, staleTime: Infinity });
}

export function useConversations() {
  return useInfiniteQuery({
    queryKey: queryKeys.conversations,
    queryFn: ({ pageParam }) => fetchConversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.messages(conversationId),
    queryFn: ({ pageParam }) => fetchMessages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => fetchNotifications(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useBlocks() {
  return useQuery({ queryKey: queryKeys.blocks, queryFn: fetchBlocks });
}
