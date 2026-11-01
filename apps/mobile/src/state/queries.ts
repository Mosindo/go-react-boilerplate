import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { chatApi, discoverApi, notificationsApi, photosApi, profileApi } from "../api";

export const keys = {
  conversations: ["conversations"] as const,
  messages: (conversationId: string) => ["messages", conversationId] as const,
  matches: ["matches"] as const,
  notifications: ["notifications"] as const,
  photos: ["photos"] as const,
  profile: ["profile", "me"] as const,
  publicProfile: (userId: string) => ["profile", userId] as const,
  settings: ["settings"] as const,
  interests: ["interests"] as const,
  blocks: ["blocks"] as const,
};

export function useConversations() {
  return useInfiniteQuery({
    queryKey: keys.conversations,
    queryFn: ({ pageParam }) => chatApi.conversations(pageParam || undefined),
    initialPageParam: "",
    getNextPageParam: (last) => last.nextCursor || undefined,
  });
}

export function useMatches() {
  return useQuery({ queryKey: keys.matches, queryFn: () => discoverApi.matches().then((r) => r.items) });
}

export function useNotifications() {
  return useQuery({ queryKey: keys.notifications, queryFn: () => notificationsApi.list() });
}

export function usePhotos() {
  return useQuery({ queryKey: keys.photos, queryFn: photosApi.list });
}

export function useMyProfile() {
  return useQuery({ queryKey: keys.profile, queryFn: profileApi.getMine });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: 60 * 60 * 1000 });
}
