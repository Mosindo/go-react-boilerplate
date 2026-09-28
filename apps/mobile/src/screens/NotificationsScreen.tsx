import React, { useCallback } from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../api/client";
import type { AppNotification } from "../api/types";
import {
  flattenNotifications,
  useMarkAllNotificationsRead,
  useMarkNotificationRead,
  useNotifications,
  useUnreadNotificationCount
} from "../hooks/useNotifications";
import type { MainStackParamList, TabParamList } from "../navigation/types";
import { EmptyView, ErrorView, ListSkeleton, showToast } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { Button, Loader, NotificationItem } from "../shared/ui";
import { spacing, useTheme } from "../theme";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParamList, "Notifications">,
  NativeStackScreenProps<MainStackParamList>
>;

export default function NotificationsScreen({ navigation }: Props) {
  const theme = useTheme();
  const query = useNotifications();
  const unread = useUnreadNotificationCount();
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();
  const items = flattenNotifications(query.data);

  const open = useCallback(
    (notification: AppNotification) => {
      if (notification.readAt === null) {
        markRead.mutate(notification.id);
      }
      const { conversationId, matchId, userId } = notification.data;
      if (conversationId) {
        navigation.navigate("Chat", { conversationId, matchId, userId });
      } else {
        navigation.navigate("Matches");
      }
    },
    [markRead, navigation]
  );

  return (
    <SafeAreaLayout edges={["top"]}>
      <View style={styles.header}>
        <Header
          right={
            <Button
              disabled={unread === 0}
              label="Mark all read"
              loading={markAll.isPending}
              onPress={() => markAll.mutate(undefined, { onError: (error) => showToast(errorMessage(error), "error") })}
              variant="ghost"
            />
          }
          title="Notifications"
        />
      </View>
      {query.isPending ? (
        <ListSkeleton />
      ) : query.isError && items.length === 0 ? (
        <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={items.length === 0 ? styles.emptyList : styles.list}
          data={items}
          keyExtractor={(item) => item.id}
          ListEmptyComponent={<EmptyView message="New matches and messages will show up here." title="Nothing yet" />}
          ListFooterComponent={query.isFetchingNextPage ? <Loader /> : null}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) {
              void query.fetchNextPage();
            }
          }}
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              tintColor={theme.primary}
            />
          }
          renderItem={({ item }) => <NotificationItem notification={item} onPress={open} />}
        />
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  list: { paddingHorizontal: spacing.sm, gap: spacing.xs },
  emptyList: { flexGrow: 1 }
});
