import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import {
  getCard,
  getDiscover,
  getPreferences,
  getProfile,
  listBlocks,
  listInterests,
  listMatches,
  listMessages,
  listNotifications,
  listPhotos
} from "../api";
import { notificationUnreadCount, totalUnread, type Paged } from "../realtime/cache";
import type { Card, MatchesPage, NotificationsPage } from "../api/types";
import { queryKeys } from "../session/queryKeys";

export const useInterests = () =>
  useQuery({ queryKey: queryKeys.interests, queryFn: listInterests, staleTime: 60 * 60_000 });

export const useProfile = (enabled = true) =>
  useQuery({ queryKey: queryKeys.profile, queryFn: getProfile, enabled, retry: false });

export const usePreferences = () =>
  useQuery({ queryKey: queryKeys.preferences, queryFn: getPreferences });

export const usePhotos = () => useQuery({ queryKey: queryKeys.photos, queryFn: listPhotos });

export const useBlocks = () => useQuery({ queryKey: queryKeys.blocks, queryFn: listBlocks });

export const useDiscover = () =>
  useQuery({
    queryKey: queryKeys.discover,
    queryFn: () => getDiscover(),
    staleTime: 0,
    gcTime: 0,
    refetchOnMount: "always"
  });

export const useCard = (userId: string, initial?: Card, enabled = true) =>
  useQuery({
    queryKey: queryKeys.card(userId),
    queryFn: () => getCard(userId),
    initialData: initial,
    enabled: enabled && userId !== ""
  });

export const useMatches = () =>
  useInfiniteQuery({
    queryKey: queryKeys.matches,
    queryFn: ({ pageParam }) => listMatches(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 30_000
  });

export const useMessages = (matchId: string) =>
  useInfiniteQuery({
    queryKey: queryKeys.messages(matchId),
    queryFn: ({ pageParam }) => listMessages(matchId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 30_000
  });

export const useNotifications = () =>
  useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => listNotifications(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 30_000
  });

/** Unread counters for the tab bar, derived from the (realtime-updated) cache. */
export function useUnreadBadges(): { messages: number; notifications: number } {
  const matches = useMatches();
  const notifications = useNotifications();
  return {
    messages: totalUnread(matches.data as Paged<MatchesPage> | undefined),
    notifications: notificationUnreadCount(
      notifications.data as Paged<NotificationsPage> | undefined
    )
  };
}
