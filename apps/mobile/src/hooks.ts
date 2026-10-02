import { useQuery } from "@tanstack/react-query";

import { notificationsApi, profileApi } from "./api/endpoints";
import { keys } from "./realtime/RealtimeProvider";

export function useProfileStatus() {
  return useQuery({ queryKey: keys.profile, queryFn: profileApi.get, staleTime: 30_000 });
}

export function useInterests() {
  return useQuery({
    queryKey: keys.interests,
    queryFn: async () => (await profileApi.interests()).interests,
    staleTime: Infinity,
  });
}

/** Unread counters for the tab badges; refreshed by realtime events and every minute. */
export function useSummary(enabled = true) {
  return useQuery({
    queryKey: keys.summary,
    queryFn: notificationsApi.summary,
    enabled,
    refetchInterval: 60_000,
  });
}
