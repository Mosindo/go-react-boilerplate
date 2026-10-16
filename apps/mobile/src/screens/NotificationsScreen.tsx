import React from "react";
import { FlatList, RefreshControl, StyleSheet } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../api/platform";
import type { AppNotification } from "../api/types";
import { queryKeys, useNotifications } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import { formatRelativeTime } from "../lib/format";
import { openThread } from "../navigation/ref";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Button, NotificationItem, colors, spacing } from "../shared/ui";

export default function NotificationsScreen() {
  const queryClient = useQueryClient();
  const { connected } = useRealtime();
  const query = useNotifications(connected);

  const refresh = () => queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  const markRead = useMutation({ mutationFn: markNotificationRead, onSuccess: refresh });
  const markAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: refresh });

  const open = (notification: AppNotification) => {
    if (!notification.isRead) {
      markRead.mutate(notification.id);
    }
    const { matchId, userId } = notification.data;
    if (matchId && userId) {
      void openThread(matchId, userId);
    }
  };

  const unread = query.data?.unread ?? 0;

  return (
    <ScreenContainer testID="screen-notifications">
      <Header
        action={
          unread > 0 ? (
            <Button
              label="Tout lire"
              loading={markAll.isPending}
              onPress={() => markAll.mutate()}
              size="sm"
              testID="mark-all-read"
              variant="outline"
            />
          ) : undefined
        }
        subtitle={unread > 0 ? `${unread} non ${unread > 1 ? "lues" : "lue"}` : "Vous êtes à jour"}
        title="Alertes"
      />
      {query.isLoading ? <LoadingView label="Chargement…" /> : null}
      {query.isError && !query.data ? (
        <ErrorView message="Impossible de charger vos alertes." onAction={() => void query.refetch()} />
      ) : null}
      {query.data ? (
        <FlatList<AppNotification>
          contentContainerStyle={styles.list}
          data={query.data.notifications}
          keyExtractor={(item) => item.id}
          ListEmptyComponent={
            <EmptyView
              message="Vos matchs et nouveaux messages apparaîtront ici."
              testID="notifications-empty"
              title="Rien pour l'instant"
            />
          }
          refreshControl={
            <RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching} tintColor={colors.primary} />
          }
          renderItem={({ item }) => (
            <NotificationItem
              body={item.body}
              createdAtLabel={formatRelativeTime(item.createdAt)}
              isRead={item.isRead}
              onPress={() => open(item)}
              title={item.title}
            />
          )}
          showsVerticalScrollIndicator={false}
        />
      ) : null}
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  list: { paddingTop: spacing.lg, paddingBottom: spacing.xxxl }
});
