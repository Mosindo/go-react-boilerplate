import React from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../api/platform";
import type { AppNotification } from "../api/types";
import { useMatches, useNotifications } from "../hooks/queries";
import { describeError } from "../lib/errors";
import { formatRelative } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Button, NotificationItem, spacing } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Notifications">;

export default function NotificationsScreen({ navigation }: Props) {
  const queryClient = useQueryClient();
  const notifications = useNotifications();
  const matches = useMatches();

  const refreshAll = () =>
    Promise.all([queryClient.invalidateQueries({ queryKey: ["notifications"] }), queryClient.invalidateQueries({ queryKey: ["matches"] })]);

  const readOne = useMutation({ mutationFn: markNotificationRead, onSuccess: refreshAll });
  const readAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: refreshAll });

  const open = (notification: AppNotification) => {
    if (!notification.isRead) {
      readOne.mutate(notification.id);
    }
    const matchId = notification.data.matchId;
    const match = matches.data?.matches.find((candidate) => candidate.id === matchId);
    if (match) {
      navigation.navigate("Chat", { matchId: match.id, peerId: match.user.userId, peerName: match.user.firstName });
    } else {
      navigation.navigate("Tabs", { screen: "Messages" });
    }
  };

  const items = notifications.data?.notifications ?? [];
  const unread = notifications.data?.unreadCount ?? 0;

  if (notifications.isLoading) {
    return <LoadingView fullScreen label="Chargement…" />;
  }
  if (notifications.isError) {
    return <ErrorView message={describeError(notifications.error)} onAction={() => void notifications.refetch()} />;
  }
  if (items.length === 0) {
    return <EmptyView icon="notifications-outline" message="Vos nouveaux matchs et messages apparaîtront ici." title="Rien de nouveau" />;
  }

  return (
    <FlatList
      contentContainerStyle={styles.list}
      data={items}
      ItemSeparatorComponent={() => <View style={styles.gap} />}
      keyExtractor={(item) => item.id}
      ListHeaderComponent={
        unread > 0 ? (
          <Button
            label="Tout marquer comme lu"
            loading={readAll.isPending}
            onPress={() => readAll.mutate()}
            size="sm"
            variant="secondary"
          />
        ) : null
      }
      refreshControl={<RefreshControl onRefresh={() => void notifications.refetch()} refreshing={notifications.isRefetching} />}
      renderItem={({ item }) => (
        <NotificationItem
          body={item.body}
          icon={item.type === "new_match" ? "heart" : "chatbubble"}
          onPress={() => open(item)}
          time={formatRelative(item.createdAt)}
          title={item.title}
          unread={!item.isRead}
        />
      )}
      testID="notifications-screen"
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.md },
  gap: { height: spacing.sm }
});
