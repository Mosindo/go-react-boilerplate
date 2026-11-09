import { useQuery } from "@tanstack/react-query";
import { notificationApi, profileApi } from "../api/endpoints";
import { ApiError } from "../api/client";
import { keys } from "./keys";

/** The signed-in user's profile. A 404 means onboarding has not started: exposed as `null`. */
export function useMyProfile() {
  return useQuery({
    queryKey: keys.profile,
    queryFn: async () => {
      try {
        return await profileApi.me();
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
  return useQuery({ queryKey: keys.interests, queryFn: profileApi.interests, staleTime: 3_600_000 });
}

export function useUnreadNotifications() {
  return useQuery({
    queryKey: keys.notifications,
    queryFn: () => notificationApi.list(),
    select: (data) => data.unreadCount,
    refetchInterval: 60_000
  });
}
