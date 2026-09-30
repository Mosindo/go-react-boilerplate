import { useQuery } from "@tanstack/react-query";
import { notificationsApi } from "../../lib/api/endpoints";
import { queryKeys } from "../../lib/queryClient";
import { useRealtime } from "../../lib/realtime/RealtimeProvider";

export function useNotifications() {
  const { connected } = useRealtime();
  return useQuery({
    queryKey: queryKeys.notifications,
    queryFn: () => notificationsApi.list(0),
    refetchInterval: connected ? false : 30_000
  });
}
