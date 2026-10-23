import React, { useCallback, useState } from "react";
import { ActivityIndicator, FlatList, RefreshControl, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../../api";
import type { NotificationItem, NotificationsPage } from "../../api/types";
import { errorMessage } from "../../i18n/errorMessage";
import { t } from "../../i18n/fr";
import { formatListTime } from "../../lib/dates";
import { useNotifications } from "../../queries";
import {
  markAllNotificationsReadInCache,
  markNotificationReadInCache,
  notificationUnreadCount,
  type Paged
} from "../../realtime/cache";
import { queryKeys } from "../../session/queryKeys";
import { createStyles, useTheme } from "../../theme/ThemeProvider";
import {
  Button,
  EmptyState,
  ErrorState,
  ListSkeleton,
  ScalePressable,
  Screen,
  Text,
  useToast
} from "../../ui";
import type { MainStackParams, TabParams } from "../navTypes";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Notifications">,
  NativeStackScreenProps<MainStackParams>
>;

const useStyles = createStyles(({ colors, spacing }) => ({
  row: {
    minHeight: 72,
    flexDirection: "row",
    gap: spacing.md,
    alignItems: "center",
    paddingVertical: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.border
  },
  dot: { width: 10, height: 10, borderRadius: 5, backgroundColor: colors.accent },
  main: { flex: 1, gap: 2 },
  footer: { paddingVertical: spacing.lg },
  toolbar: { alignItems: "flex-end" }
}));

export default function NotificationsScreen({ navigation }: Props) {
  const styles = useStyles();
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const [refreshing, setRefreshing] = useState(false);

  const data = query.data as Paged<NotificationsPage> | undefined;
  const items = data?.pages.flatMap((page) => page.notifications) ?? [];
  const unread = notificationUnreadCount(data);

  const markOne = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onMutate: (id) =>
      queryClient.setQueryData<Paged<NotificationsPage>>(queryKeys.notifications, (current) =>
        markNotificationReadInCache(current, id, new Date().toISOString())
      ),
    onError: () => void queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
  });

  const markAll = useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: () => {
      queryClient.setQueryData<Paged<NotificationsPage>>(queryKeys.notifications, (current) =>
        markAllNotificationsReadInCache(current, new Date().toISOString())
      );
      toast.success(t.notifications.allRead);
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  const refresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await query.refetch();
    } finally {
      setRefreshing(false);
    }
  }, [query]);

  const open = (item: NotificationItem) => {
    if (!item.readAt) markOne.mutate(item.id);
    if (item.data.matchId) {
      navigation.navigate("Chat", { matchId: item.data.matchId, userId: item.data.userId });
    }
  };

  if (query.isPending) {
    return (
      <Screen testID="notifications-screen">
        <ListSkeleton />
      </Screen>
    );
  }
  if (query.isError && items.length === 0) {
    return (
      <Screen testID="notifications-screen">
        <ErrorState onRetry={() => query.refetch()} title={t.notifications.errorTitle} />
      </Screen>
    );
  }

  return (
    <Screen testID="notifications-screen">
      {unread > 0 ? (
        <View style={styles.toolbar}>
          <Button
            compact
            label={t.notifications.readAll}
            loading={markAll.isPending}
            onPress={() => markAll.mutate()}
            testID="notifications-read-all"
            variant="ghost"
          />
        </View>
      ) : null}
      <FlatList
        contentContainerStyle={items.length === 0 ? { flexGrow: 1 } : undefined}
        data={items}
        keyExtractor={(item) => item.id}
        ListEmptyComponent={
          <EmptyState
            glyph="✧"
            message={t.notifications.emptyText}
            title={t.notifications.emptyTitle}
          />
        }
        ListFooterComponent={
          query.isFetchingNextPage ? (
            <View style={styles.footer}>
              <ActivityIndicator color={colors.primary} />
            </View>
          ) : null
        }
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        onEndReachedThreshold={0.4}
        refreshControl={
          <RefreshControl onRefresh={refresh} refreshing={refreshing} tintColor={colors.primary} />
        }
        renderItem={({ item }) => (
          <ScalePressable
            accessibilityLabel={`${item.readAt ? "" : `${t.notifications.unread}. `}${item.title}. ${item.body}`}
            accessibilityRole="button"
            onPress={() => open(item)}
            testID={`notification-${item.id}`}
          >
            <View style={styles.row}>
              <View style={[styles.dot, item.readAt ? { opacity: 0 } : null]} />
              <View style={styles.main}>
                <Text style={{ fontWeight: item.readAt ? "500" : "800" }} variant="label">
                  {item.title}
                </Text>
                <Text numberOfLines={2} tone="muted">
                  {item.body}
                </Text>
              </View>
              <Text tone="muted" variant="caption">
                {formatListTime(item.createdAt)}
              </Text>
            </View>
          </ScalePressable>
        )}
      />
    </Screen>
  );
}
