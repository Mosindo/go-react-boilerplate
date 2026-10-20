import { useInfiniteQuery } from "@tanstack/react-query";
import { fetchNotifications } from "../../api/social";
import { queryKeys } from "../../api/keys";

export function useNotificationsQuery() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => fetchNotifications(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
}

export function useNotificationBadge(): number {
  const query = useNotificationsQuery();
  return query.data?.pages[0]?.unreadCount ?? 0;
}
