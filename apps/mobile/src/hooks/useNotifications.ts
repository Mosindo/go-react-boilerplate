import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getNotifications, markAllNotificationsRead, markNotificationRead } from "../api/notifications";
import { queryKeys } from "../api/queryKeys";
import {
  markAllNotificationsReadInCache,
  markNotificationReadInCache,
  type NotificationPages
} from "../domain/realtimeCache";
import type { AppNotification } from "../api/types";

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => getNotifications(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor ?? undefined
  });
}

export function flattenNotifications(data: NotificationPages | undefined): AppNotification[] {
  return data ? data.pages.flatMap((page) => page.items) : [];
}

export function useUnreadNotificationCount(): number {
  const { data } = useNotifications();
  return data?.pages[0]?.unreadCount ?? 0;
}

export function useMarkNotificationRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onMutate: (id) => {
      queryClient.setQueryData<NotificationPages>(queryKeys.notifications, (data) =>
        markNotificationReadInCache(data, id, new Date().toISOString())
      );
    },
    onError: () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
  });
}

export function useMarkAllNotificationsRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => markAllNotificationsRead(),
    onSuccess: () => {
      queryClient.setQueryData<NotificationPages>(queryKeys.notifications, (data) =>
        markAllNotificationsReadInCache(data, new Date().toISOString())
      );
    }
  });
}
