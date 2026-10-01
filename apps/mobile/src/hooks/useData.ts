import { useQuery } from "@tanstack/react-query";
import { chatApi, discoverApi, notificationApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import { useRealtime } from "./useRealtime";

/** While the socket is down, lists fall back to light polling. */
function pollingInterval(connected: boolean, ms = 15_000): number | false {
  return connected ? false : ms;
}

export function useConversations() {
  const { connected } = useRealtime();
  return useQuery({
    queryKey: queryKeys.conversations,
    queryFn: () => chatApi.conversations(),
    refetchInterval: pollingInterval(connected)
  });
}

export function useMatches() {
  const { connected } = useRealtime();
  return useQuery({
    queryKey: queryKeys.matches,
    queryFn: () => discoverApi.matches(),
    refetchInterval: pollingInterval(connected, 30_000)
  });
}

export function useNotifications() {
  const { connected } = useRealtime();
  return useQuery({
    queryKey: queryKeys.notifications,
    queryFn: () => notificationApi.list(),
    refetchInterval: pollingInterval(connected)
  });
}
