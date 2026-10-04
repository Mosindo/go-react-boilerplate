import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as api from "../api/notifications";
import { queryKeys } from "../api/queryKeys";
import type { AppNotification } from "../api/types";
import { flattenPages, mapPages, nextCursorOf, type CursorData } from "../lib/pagination";

type Data = CursorData<"notifications", AppNotification>;

export function useNotifications() {
  const query = useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => api.fetchNotifications(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: nextCursorOf
  });
  return { ...query, items: flattenPages(query.data, "notifications") };
}

export function useNotificationUnreadCount(): number {
  return (
    useQuery({
      queryKey: queryKeys.notificationsUnread,
      queryFn: api.fetchNotificationUnreadCount,
      staleTime: 15_000
    }).data ?? 0
  );
}

function patch(data: Data | undefined, fn: (n: AppNotification) => AppNotification): Data | undefined {
  return data ? mapPages(data, "notifications", (items) => items.map(fn)) : data;
}

export function useMarkNotificationRead() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.markNotificationRead(id),
    onMutate: (id) => {
      const readAt = new Date().toISOString();
      const before = client
        .getQueryData<Data>(queryKeys.notifications)
        ?.pages.flatMap((p) => p.notifications)
        .find((n) => n.id === id);
      client.setQueryData<Data>(queryKeys.notifications, (d) =>
        patch(d, (n) => (n.id === id ? { ...n, readAt: n.readAt ?? readAt } : n))
      );
      if (before && !before.readAt) {
        client.setQueryData<number>(queryKeys.notificationsUnread, (c) => Math.max(0, (c ?? 0) - 1));
      }
    },
    onSettled: () => {
      void client.invalidateQueries({ queryKey: queryKeys.notificationsUnread });
    }
  });
}

export function useMarkAllNotificationsRead() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: api.markAllNotificationsRead,
    onSuccess: () => {
      const readAt = new Date().toISOString();
      client.setQueryData<Data>(queryKeys.notifications, (d) =>
        patch(d, (n) => ({ ...n, readAt: n.readAt ?? readAt }))
      );
      client.setQueryData<number>(queryKeys.notificationsUnread, 0);
    }
  });
}
