import { useQuery } from "@tanstack/react-query";
import { notificationsApi } from "../../api/endpoints";
import { keys } from "../../providers/queryClient";
import { useRealtimeConnected } from "../../providers/RealtimeProvider";

export function useNotifications() {
  const live = useRealtimeConnected();
  return useQuery({
    queryKey: keys.notifications,
    queryFn: () => notificationsApi.list(30),
    refetchInterval: live ? false : 30_000
  });
}
