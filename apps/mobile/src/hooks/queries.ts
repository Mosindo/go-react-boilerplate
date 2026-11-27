import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { discoveryApi, matchesApi, notificationsApi, profileApi, safetyApi } from "../api/endpoints";
import type { MatchSummary, MatchesPage } from "../api/types";
import { keys, queryClient } from "./queryClient";

export function useMyProfile() {
  return useQuery({ queryKey: keys.myProfile, queryFn: profileApi.mine });
}

export function useInterests() {
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: Infinity });
}

export function useDiscovery(enabled = true) {
  return useQuery({ queryKey: keys.discovery, queryFn: () => discoveryApi.feed(10), enabled, staleTime: 0 });
}

export function useMatches(pollWhenOffline = false) {
  return useInfiniteQuery({
    queryKey: keys.matches,
    queryFn: ({ pageParam }) => matchesApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor || undefined,
    refetchInterval: pollWhenOffline ? 30_000 : false
  });
}

/** Finds a match in the inbox cache, falling back to the API. */
export function useMatch(matchId: string) {
  return useQuery({
    queryKey: keys.match(matchId),
    queryFn: () => matchesApi.get(matchId),
    initialData: () => {
      const pages = queryClient.getQueryData<{ pages: MatchesPage[] }>(keys.matches);
      return pages?.pages.flatMap((p) => p.matches).find((m: MatchSummary) => m.id === matchId);
    }
  });
}

export function useMessages(matchId: string, pollWhenOffline = false) {
  return useInfiniteQuery({
    queryKey: keys.messages(matchId),
    queryFn: ({ pageParam }) => matchesApi.messages(matchId, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor || undefined,
    refetchInterval: pollWhenOffline ? 10_000 : false
  });
}

export function usePublicProfile(userId: string) {
  return useQuery({ queryKey: keys.profile(userId), queryFn: () => profileApi.publicProfile(userId) });
}

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor || undefined
  });
}

export function useUnreadNotifications() {
  return useQuery({ queryKey: keys.unread, queryFn: notificationsApi.unreadCount, refetchInterval: 120_000 });
}

export function useBlocks() {
  return useQuery({ queryKey: keys.blocks, queryFn: safetyApi.blocks });
}
