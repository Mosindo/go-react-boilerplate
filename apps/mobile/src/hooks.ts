import { useQuery } from "@tanstack/react-query";
import { chatApi, notificationApi, profileApi } from "./api/endpoints";

export const PROFILE_KEY = ["profile"] as const;

export function useProfileBundle() {
  return useQuery({ queryKey: PROFILE_KEY, queryFn: profileApi.get, staleTime: 60_000 });
}

export function useUnreadMessages(): number {
  const { data } = useQuery({ queryKey: ["conversations"], queryFn: () => chatApi.conversations(), staleTime: 15_000 });
  return (data?.items ?? []).reduce((sum, c) => sum + c.unreadCount, 0);
}

export function useUnreadNotifications(): number {
  const { data } = useQuery({ queryKey: ["notifications"], queryFn: () => notificationApi.list(), staleTime: 15_000 });
  return data?.unreadCount ?? 0;
}
