import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { keys } from "../api/keys";
import * as api from "../api/platform";
import type { AppNotification } from "../api/types";
import { useNotifications } from "../hooks/data";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Button, NotificationItem, spacing } from "../shared/ui";

export default function NotificationsScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "Notifications">) {
  const client = useQueryClient();
  const query = useNotifications();
  const items = query.data?.pages.flatMap((page) => page.notifications) ?? [];
  const refresh = () => {
    void client.invalidateQueries({ queryKey: keys.notifications });
    void client.invalidateQueries({ queryKey: keys.notificationsUnread });
  };
  const markOne = useMutation({ mutationFn: api.markNotificationRead, onSuccess: refresh });
  const markAll = useMutation({ mutationFn: api.markAllNotificationsRead, onSuccess: refresh });

  const open = (notification: AppNotification) => {
    if (!notification.isRead) markOne.mutate(notification.id);
    const { conversationId, userId, matchId } = notification.data;
    if (conversationId && userId) {
      navigation.navigate("ProfileDetail", { userId, matchId, conversationId });
    }
  };

  if (query.isLoading) return <LoadingView fullScreen />;
  if (query.isError) return <ErrorView message="Impossible de charger vos notifications." onAction={() => void query.refetch()} />;
  if (items.length === 0) return <EmptyView icon="notifications-outline" message="Vos nouveaux matchs apparaîtront ici." title="Rien de nouveau" />;

  return (
    <FlatList
      contentContainerStyle={styles.list}
      data={items}
      ItemSeparatorComponent={() => <View style={styles.gap} />}
      keyExtractor={(item) => item.id}
      ListHeaderComponent={
        items.some((n) => !n.isRead) ? (
          <Button loading={markAll.isPending} onPress={() => markAll.mutate()} size="sm" style={styles.markAll} variant="secondary">
            Tout marquer comme lu
          </Button>
        ) : null
      }
      onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
      onRefresh={() => void query.refetch()}
      refreshing={query.isRefetching && !query.isFetchingNextPage}
      renderItem={({ item }) => (
        <NotificationItem body={item.body} createdAt={item.createdAt} isRead={item.isRead} onPress={() => open(item)} title={item.title} />
      )}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, width: "100%", maxWidth: 560, alignSelf: "center" },
  gap: { height: spacing.sm },
  markAll: { alignSelf: "flex-end", marginBottom: spacing.md }
});
