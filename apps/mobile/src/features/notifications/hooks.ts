import { useMemo } from 'react';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { notificationsApi, queryKeys } from '../../api';
import type { AppNotification } from '../../api';
import { useErrorToast } from '../../lib/useErrorToast';
import { notificationsAllRead, notificationsMarkedRead } from '../../realtime/cache';
import type { NotificationsData } from '../../realtime/cache';

export function useNotifications() {
  const query = useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
    staleTime: 15_000,
  });
  const notifications = useMemo<AppNotification[]>(() => {
    const seen = new Set<string>();
    const list: AppNotification[] = [];
    for (const page of query.data?.pages ?? []) {
      for (const n of page.notifications) {
        if (!seen.has(n.id)) {
          seen.add(n.id);
          list.push(n);
        }
      }
    }
    return list;
  }, [query.data]);
  const unreadCount = query.data?.pages[0]?.unreadCount ?? 0;
  return { query, notifications, unreadCount };
}

export function useMarkNotificationRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => notificationsApi.markRead(id),
    onMutate: (id) => {
      qc.setQueryData<NotificationsData>(queryKeys.notifications, (old) =>
        old ? notificationsMarkedRead(old, id) : old,
      );
    },
    onError: () => void qc.invalidateQueries({ queryKey: queryKeys.notifications }),
  });
}

export function useMarkAllNotificationsRead() {
  const qc = useQueryClient();
  const onError = useErrorToast();
  return useMutation({
    mutationFn: () => notificationsApi.markAllRead(),
    onMutate: () => {
      qc.setQueryData<NotificationsData>(queryKeys.notifications, (old) =>
        old ? notificationsAllRead(old) : old,
      );
    },
    onError: (error) => {
      void qc.invalidateQueries({ queryKey: queryKeys.notifications });
      onError(error);
    },
  });
}
