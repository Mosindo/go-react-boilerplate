import React, { useCallback, useMemo, useState } from "react";
import { ActivityIndicator, FlatList, RefreshControl, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../api/notifications";
import type { AppNotification } from "../api/models";
import { AppButton, Banner, Skeleton, StateView, Txt } from "../components/kit";
import { NotificationRow } from "../components/NotificationRow";
import {
  flattenNotifications,
  markAllNotificationsReadInCache,
  markNotificationReadInCache,
  unreadNotificationCount
} from "../lib/dating/cache";
import {
  NOTIFICATIONS_KEY,
  useNotificationsQuery,
  type NotificationsData
} from "../realtime/queries";
import { useTheme } from "../shared/ui/theme";

type Props = { onOpenConversation: (conversationId: string) => void };

export default function NotificationsScreen({ onOpenConversation }: Props) {
  const { colors, spacing } = useTheme();
  const queryClient = useQueryClient();
  const query = useNotificationsQuery();
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const pages = query.data?.pages;
  const items = useMemo(() => (pages ? flattenNotifications(pages) : []), [pages]);
  const unread = unreadNotificationCount(pages);

  const markOne = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onMutate: (id) => {
      queryClient.setQueryData<NotificationsData>(NOTIFICATIONS_KEY, (d) =>
        d ? markNotificationReadInCache(d, id) : d
      );
    },
    onError: () => void queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY })
  });

  const markAll = useMutation({
    mutationFn: () => markAllNotificationsRead(),
    onMutate: () => {
      setError(null);
      queryClient.setQueryData<NotificationsData>(NOTIFICATIONS_KEY, (d) =>
        d ? markAllNotificationsReadInCache(d) : d
      );
    },
    onError: () => {
      setError("We couldn't mark everything as read. Please try again.");
      void queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_KEY });
    }
  });

  const onPressItem = useCallback(
    (notification: AppNotification) => {
      if (!notification.isRead) markOne.mutate(notification.id);
      const conversationId = notification.data.conversationId;
      if (conversationId) onOpenConversation(conversationId);
    },
    [markOne, onOpenConversation]
  );

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await query.refetch();
    } finally {
      setRefreshing(false);
    }
  }, [query]);

  const onEndReached = useCallback(() => {
    if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
  }, [query]);

  let body: React.ReactNode;
  if (query.isPending) {
    body = (
      <View style={{ padding: spacing.lg, gap: spacing.md }} testID="notifications-loading">
        {[0, 1, 2, 3, 4].map((i) => (
          <Skeleton key={i} style={{ height: 64, borderRadius: 16 }} />
        ))}
      </View>
    );
  } else if (query.isError && items.length === 0) {
    body = (
      <StateView
        testID="notifications-error"
        title="Couldn't load notifications"
        message="Check your connection and try again."
        actionLabel="Try again"
        onAction={() => void query.refetch()}
        refreshing={refreshing}
        onRefresh={() => void onRefresh()}
      />
    );
  } else if (items.length === 0) {
    body = (
      <StateView
        testID="notifications-empty"
        glyph="🔔"
        title="Nothing yet"
        message="New matches and messages will show up here."
        refreshing={refreshing}
        onRefresh={() => void onRefresh()}
      />
    );
  } else {
    body = (
      <FlatList
        testID="notifications-list"
        data={items}
        keyExtractor={(n) => n.id}
        renderItem={({ item }) => <NotificationRow notification={item} onPress={onPressItem} />}
        contentContainerStyle={{ padding: spacing.sm }}
        onEndReached={onEndReached}
        onEndReachedThreshold={0.5}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={() => void onRefresh()}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        }
        ListFooterComponent={
          query.isFetchingNextPage ? (
            <ActivityIndicator color={colors.primary} style={{ padding: spacing.lg }} />
          ) : null
        }
      />
    );
  }

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: colors.background }]}>
      <View style={[styles.header, { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm }]}>
        <Txt variant="title" accessibilityRole="header" style={styles.flex}>
          Notifications
        </Txt>
        {unread > 0 ? (
          <AppButton
            label="Mark all read"
            variant="ghost"
            onPress={() => markAll.mutate()}
            loading={markAll.isPending}
            testID="notifications-mark-all"
            style={{ paddingHorizontal: spacing.md }}
          />
        ) : null}
      </View>
      {error ? (
        <Banner
          tone="danger"
          message={error}
          onDismiss={() => setError(null)}
          style={{ marginHorizontal: spacing.lg }}
        />
      ) : null}
      <View style={styles.flex}>{body}</View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center" }
});
