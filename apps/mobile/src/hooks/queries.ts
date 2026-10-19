import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { getOwnProfile, listInterests } from "../api/profile";
import { listConversations, listMessages, listNotifications } from "../api/social";
import type { OwnProfile } from "../api/types";

/** The signed-in user's profile, or null when onboarding has not created it yet. */
export function useOwnProfile() {
  return useQuery<OwnProfile | null>({
    queryKey: ["profile"],
    queryFn: async () => {
      try {
        return await getOwnProfile();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404 && error.code === "profile_not_found")
          return null;
        throw error;
      }
    },
    retry: (count, error) => !(error instanceof ApiError) && count < 2
  });
}

export function useInterests() {
  return useQuery({ queryKey: ["interests"], queryFn: listInterests, staleTime: Infinity });
}

export function useConversations() {
  return useInfiniteQuery({
    queryKey: ["conversations"],
    queryFn: ({ pageParam }) => listConversations(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useMessages(conversationId: string) {
  return useInfiniteQuery({
    queryKey: ["messages", conversationId],
    queryFn: ({ pageParam }) => listMessages(conversationId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: ["notifications"],
    queryFn: ({ pageParam }) => listNotifications(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
}

/** True when the profile is complete enough to enter the main app. */
export function isOnboarded(profile: OwnProfile | null | undefined): boolean {
  return !!profile && profile.photos.length > 0 && profile.preferences !== null;
}
