import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { chatApi, matchingApi, notificationsApi, profileApi, safetyApi } from "../api/endpoints";
import type { OwnProfile } from "../api/types";
import { useRealtimeStatus } from "./realtimeStatus";

export const keys = {
  profile: ["profile"] as const,
  interests: ["interests"] as const,
  publicProfile: (id: string) => ["publicProfile", id] as const,
  matches: ["matches"] as const,
  chats: ["chats"] as const,
  messages: (userId: string) => ["messages", userId] as const,
  unreadChats: ["unreadChats"] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  discover: ["discover"] as const
};

/** Own profile; `null` means the profile was not created yet (onboarding). */
export function useOwnProfile() {
  return useQuery<OwnProfile | null>({
    queryKey: keys.profile,
    queryFn: async () => {
      try {
        return await profileApi.get();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return null;
        }
        throw error;
      }
    }
  });
}

export function useSetProfile() {
  const qc = useQueryClient();
  return (profile: OwnProfile) => qc.setQueryData(keys.profile, profile);
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.catalogue, staleTime: 60 * 60_000 });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => profileApi.publicProfile(userId) });
}

/** Realtime pushes invalidations; polling only kicks in when the socket is down. */
function useFallbackInterval(ms: number) {
  const connected = useRealtimeStatus();
  return connected ? false : ms;
}

export function useMatches() {
  return useQuery({ queryKey: keys.matches, queryFn: matchingApi.matches, refetchInterval: useFallbackInterval(30_000) });
}

export function useChats() {
  return useQuery({ queryKey: keys.chats, queryFn: chatApi.list, refetchInterval: useFallbackInterval(20_000) });
}

export function useUnreadChats() {
  return useQuery({ queryKey: keys.unreadChats, queryFn: chatApi.unread, refetchInterval: useFallbackInterval(30_000) });
}

export function useNotifications() {
  return useQuery({ queryKey: keys.notifications, queryFn: notificationsApi.list, refetchInterval: useFallbackInterval(30_000) });
}

export function useBlocks() {
  return useQuery({ queryKey: keys.blocks, queryFn: safetyApi.blocks });
}

/** Invalidate everything that depends on the relationship graph (after block, unmatch, report). */
export function useInvalidateRelationships() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: keys.matches });
    void qc.invalidateQueries({ queryKey: keys.chats });
    void qc.invalidateQueries({ queryKey: keys.unreadChats });
    void qc.invalidateQueries({ queryKey: keys.blocks });
  };
}

export function useMarkNotificationsRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: notificationsApi.markAllRead,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.notifications })
  });
}
