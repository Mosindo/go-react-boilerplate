import { infiniteQueryOptions, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationsApi, queryKeys } from "../../api";
import { markAllNotificationsRead, markNotificationRead, type NotificationPages } from "../../realtime/cache";

const notificationsQuery = infiniteQueryOptions({
  queryKey: queryKeys.notifications,
  queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
  initialPageParam: undefined as string | undefined,
  getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined
});

export function useNotifications() {
  return useInfiniteQuery(notificationsQuery);
}

export function useUnreadNotificationCount(): number {
  const { data } = useInfiniteQuery({ ...notificationsQuery, select: (pages) => pages.pages[0]?.unreadCount ?? 0 });
  return data ?? 0;
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: notificationsApi.markRead,
    meta: { silent: true },
    onSuccess: (_void, id) => {
      queryClient.setQueryData<NotificationPages>(queryKeys.notifications, (data) =>
        markNotificationRead(data, id, new Date().toISOString())
      );
    }
  });
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: notificationsApi.markAllRead,
    onSuccess: () => {
      queryClient.setQueryData<NotificationPages>(queryKeys.notifications, (data) =>
        markAllNotificationsRead(data, new Date().toISOString())
      );
    }
  });
}
