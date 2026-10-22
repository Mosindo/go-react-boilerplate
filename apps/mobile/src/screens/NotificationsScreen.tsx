import React from "react";
import { FlatList, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { markAllNotificationsRead, markNotificationRead } from "../api/notifications";
import type { AppNotification } from "../api/types";
import { PersonAvatar } from "../components/PersonAvatar";
import { useNotificationsQuery } from "../hooks/useFeeds";
import { formatShortTime } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Badge, Button, ListItem, Text, spacing } from "../shared/ui";

function describe(notification: AppNotification): { title: string; body: string } {
  const name = notification.actor?.firstName ?? "Quelqu'un";
  return notification.type === "match"
    ? { title: "Nouveau match", body: `Vous et ${name} vous plaisez mutuellement.` }
    : { title: `Message de ${name}`, body: "Vous avez de nouveaux messages non lus." };
}

export default function NotificationsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const query = useNotificationsQuery();

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["notifications"] });
  const readOne = useMutation({ mutationFn: markNotificationRead, onSuccess: invalidate });
  const readAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: invalidate });

  const items = query.data?.pages.flatMap((page) => page.notifications) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const open = (notification: AppNotification) => {
    if (!notification.readAt) {
      readOne.mutate(notification.id);
    }
    if (notification.conversationId && notification.actor) {
      navigation.navigate("Chat", {
        conversationId: notification.conversationId,
        peerId: notification.actor.userId,
        name: notification.actor.firstName
      });
    }
  };

  if (query.isLoading) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <LoadingView label="Chargement…" />
      </ScreenContainer>
    );
  }
  if (query.isError) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <ErrorView actionLabel="Réessayer" message={errorMessage(query.error)} onAction={() => void query.refetch()} />
      </ScreenContainer>
    );
  }

  return (
    <ScreenContainer contentMaxWidth={640} edges={["top", "left", "right"]} testID="notifications-screen">
      <FlatList
        ItemSeparatorComponent={() => <View style={styles.separator} />}
        ListEmptyComponent={
          <EmptyView message="Matchs et nouveaux messages s'afficheront ici." testID="notifications-empty" title="Rien de nouveau" />
        }
        ListHeaderComponent={
          <View style={styles.header}>
            <Text accessibilityRole="header" variant="heading" weight="bold">
              Notifications
            </Text>
            {unread > 0 ? (
              <Button
                label="Tout marquer comme lu"
                loading={readAll.isPending}
                onPress={() => readAll.mutate()}
                size="sm"
                testID="notifications-read-all"
                variant="ghost"
              />
            ) : null}
          </View>
        }
        data={items}
        keyExtractor={(item) => item.id}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) {
            void query.fetchNextPage();
          }
        }}
        onRefresh={() => void query.refetch()}
        refreshing={query.isRefetching}
        renderItem={({ item }) => {
          const text = describe(item);
          return (
            <ListItem
              leading={item.actor ? <PersonAvatar name={item.actor.firstName} photo={item.actor.photo} size={48} /> : undefined}
              onPress={() => open(item)}
              subtitle={text.body}
              testID={`notification-${item.type}`}
              title={text.title}
              trailing={
                <View style={styles.trailing}>
                  <Text tone="muted" variant="caption">
                    {formatShortTime(item.createdAt)}
                  </Text>
                  {!item.readAt ? <Badge label="Nouveau" variant="primary" /> : null}
                </View>
              }
              variant={item.readAt ? "muted" : "accent"}
            />
          );
        }}
      />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.md },
  separator: { height: spacing.sm },
  trailing: { alignItems: "flex-end", gap: spacing.xs }
});
