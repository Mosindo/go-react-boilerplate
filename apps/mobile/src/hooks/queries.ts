import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { datingApi } from "../api/dating";
import type { MatchSummary, ProfileInput, Preferences } from "../api/types";

export const keys = {
  me: ["me", "profile"] as const,
  interests: ["interests"] as const,
  discover: ["discover"] as const,
  matches: ["matches"] as const,
  messages: (matchId: string) => ["messages", matchId] as const,
  notifications: ["notifications"] as const,
  blocks: ["blocks"] as const,
  profile: (userId: string) => ["profile", userId] as const
};

export function useMyProfile() {
  return useQuery({ queryKey: keys.me, queryFn: datingApi.myProfile, staleTime: 10_000 });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: datingApi.interests, staleTime: Infinity });
}

export function useMatches() {
  return useQuery({ queryKey: keys.matches, queryFn: datingApi.matches });
}

export function useNotifications() {
  return useQuery({ queryKey: keys.notifications, queryFn: datingApi.notifications });
}

export function useBlocks() {
  return useQuery({ queryKey: keys.blocks, queryFn: datingApi.blocks });
}

export function useUserProfile(userId: string) {
  return useQuery({ queryKey: keys.profile(userId), queryFn: () => datingApi.profile(userId) });
}

export function useUnreadMessages(): number {
  const { data } = useMatches();
  return (data ?? []).reduce((sum: number, m: MatchSummary) => sum + m.unreadCount, 0);
}

export function useSaveProfile() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: ProfileInput) => datingApi.saveProfile(input),
    onSuccess: (data) => client.setQueryData(keys.me, data)
  });
}

export function useSavePreferences() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: Preferences) => datingApi.savePreferences(input),
    onSuccess: (data) => {
      client.setQueryData(keys.me, data);
      // Preferences change who can be shown.
      void client.invalidateQueries({ queryKey: keys.discover });
    }
  });
}

export function useSavePrivacy() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: { isVisible?: boolean; showDistance?: boolean }) => datingApi.savePrivacy(input),
    onSuccess: (data) => client.setQueryData(keys.me, data)
  });
}
