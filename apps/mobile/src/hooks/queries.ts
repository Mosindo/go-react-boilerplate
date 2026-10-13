import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import {
  fetchBlocked,
  fetchConversations,
  fetchInterests,
  fetchMatches,
  fetchNotifications,
  fetchOwnProfile,
  fetchPublicProfile,
  type NotificationsPage
} from "../api/platform";
import { isApiError } from "../api/client";
import type { BlockedUser, Conversation, Interest, Match, OwnProfile, PublicProfile } from "../api/types";
import { queryKeys } from "../lib/queryClient";
import { useRealtime } from "./useRealtime";

/** The caller's own profile, or null when it has not been created yet. */
export function useOwnProfile(enabled = true): UseQueryResult<OwnProfile | null, Error> {
  return useQuery<OwnProfile | null, Error>({
    queryKey: queryKeys.ownProfile,
    enabled,
    queryFn: async () => {
      try {
        return await fetchOwnProfile();
      } catch (error) {
        if (isApiError(error, 404)) {
          return null;
        }
        throw error;
      }
    }
  });
}

export function useInterests(): UseQueryResult<Interest[], Error> {
  return useQuery({ queryKey: queryKeys.interests, queryFn: fetchInterests, staleTime: Infinity });
}

export function usePublicProfile(userId: string): UseQueryResult<PublicProfile, Error> {
  return useQuery({ queryKey: queryKeys.publicProfile(userId), queryFn: () => fetchPublicProfile(userId) });
}

export function useMatches(): UseQueryResult<Match[], Error> {
  const { connected } = useRealtime();
  return useQuery({ queryKey: queryKeys.matches, queryFn: fetchMatches, refetchInterval: connected ? false : 30_000 });
}

export function useConversations(): UseQueryResult<Conversation[], Error> {
  const { connected } = useRealtime();
  return useQuery({ queryKey: queryKeys.conversations, queryFn: fetchConversations, refetchInterval: connected ? false : 15_000 });
}

export function useNotifications(): UseQueryResult<NotificationsPage, Error> {
  const { connected } = useRealtime();
  return useQuery({ queryKey: queryKeys.notifications, queryFn: fetchNotifications, refetchInterval: connected ? false : 30_000 });
}

export function useBlocked(): UseQueryResult<BlockedUser[], Error> {
  return useQuery({ queryKey: queryKeys.blocked, queryFn: fetchBlocked });
}
