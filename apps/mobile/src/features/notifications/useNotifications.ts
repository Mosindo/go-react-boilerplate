import { useInfiniteQuery } from "@tanstack/react-query";
import { notificationsApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";

const PAGE = 30;

export function useNotifications(enabled = true) {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    enabled,
    initialPageParam: 0,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam, PAGE),
    getNextPageParam: (last, all) => (last.notifications.length === PAGE ? all.length * PAGE : undefined),
    staleTime: 15_000
  });
}
