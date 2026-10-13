import React from "react";
import { FlatList, RefreshControl, StyleSheet } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../api/platform";
import type { AppNotification } from "../api/types";
import { useNotifications } from "../hooks/queries";
import { queryKeys } from "../lib/queryClient";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, NotificationItem, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const queryClient = useQueryClient();
  const query = useNotifications();

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  const markOne = useMutation({ mutationFn: markNotificationRead, onSuccess: refresh });
  const markAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: refresh });

  function open(notification: AppNotification) {
    if (!notification.isRead) {
      markOne.mutate(notification.id);
    }
    const { conversationId, userId } = notification.data;
    if (conversationId && userId) {
      navigation.navigate("Conversation", { conversationId, userId, name: notification.data.name ?? notification.title });
    }
  }

  if (query.isLoading) {
    return (
      <ScreenContainer>
        <LoadingView />
      </ScreenContainer>
    );
  }
  if (query.isError) {
    return (
      <ScreenContainer>
        <ErrorView message="Impossible de charger vos notifications." onAction={() => void query.refetch()} />
      </ScreenContainer>
    );
  }

  const items = query.data?.notifications ?? [];
  const unread = query.data?.unreadCount ?? 0;
  return (
    <ScreenContainer>
      <FlatList
        ListEmptyComponent={
          <EmptyView icon="notifications-outline" message="Vos nouveaux matchs et messages apparaîtront ici." title="Rien de neuf" />
        }
        ListHeaderComponent={
          <Header
            action={
              unread > 0 ? (
                <Button
                  label="Tout lire"
                  loading={markAll.isPending}
                  onPress={() => markAll.mutate()}
                  size="sm"
                  variant="secondary"
                  testID="notifications-read-all"
                />
              ) : undefined
            }
            subtitle={unread > 0 ? `${unread} non lue${unread > 1 ? "s" : ""}` : undefined}
            title="Notifications"
          />
        }
        contentContainerStyle={styles.list}
        data={items}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching} />}
        renderItem={({ item }) => (
          <NotificationItem
            body={item.body}
            createdAt={item.createdAt}
            isRead={item.isRead}
            onPress={() => open(item)}
            title={item.title}
            type={item.type}
          />
        )}
        showsVerticalScrollIndicator={false}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({ list: { flexGrow: 1, gap: spacing.xs, paddingBottom: spacing.xxxl } });
