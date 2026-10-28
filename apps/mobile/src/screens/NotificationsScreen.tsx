import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { keys } from "../api/keys";
import { markAllNotificationsRead, markNotificationRead } from "../api/social";
import type { AppNotification } from "../api/types";
import { Button, EmptyState, ErrorState, ListSkeleton, Screen, Text, useToast } from "../components";
import { useNotifications } from "../hooks/queries";
import { errorMessage, timeLabel } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { markNotificationsRead, type NotificationsCache } from "../realtime/cache";
import { MIN_TOUCH, spacing, useTheme } from "../theme";

function describe(n: AppNotification): string {
  return n.type === "match" ? `You matched with ${n.actor.firstName}` : `${n.actor.firstName} sent you a message`;
}

export function NotificationsScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "Notifications">) {
  const { colors } = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const items = query.data?.pages.flatMap((p) => p.notifications) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const readOne = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onMutate: (id) => queryClient.setQueryData<NotificationsCache>(keys.notifications, (c) => markNotificationsRead(c, id)),
    onError: (e) => toast.show(errorMessage(e), "error")
  });
  const readAll = useMutation({
    mutationFn: markAllNotificationsRead,
    onMutate: () => queryClient.setQueryData<NotificationsCache>(keys.notifications, (c) => markNotificationsRead(c, "all")),
    onSuccess: () => toast.show("All caught up", "success"),
    onError: (e) => toast.show(errorMessage(e), "error")
  });

  if (query.isPending) return <Screen padded={false}><ListSkeleton /></Screen>;
  if (query.isError) return <Screen><ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} /></Screen>;

  return (
    <Screen padded={false} edges={["bottom"]} testID="notifications-screen">
      {unread > 0 ? <Button title="Mark all as read" variant="ghost" onPress={() => readAll.mutate()} testID="notifications-read-all" /> : null}
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        refreshing={query.isRefetching && !query.isFetchingNextPage}
        onRefresh={() => void query.refetch()}
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        contentContainerStyle={items.length === 0 ? { flexGrow: 1 } : undefined}
        ListEmptyComponent={<EmptyState title="Nothing yet" message="New matches and messages will show up here." />}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            onPress={() => {
              if (!item.readAt) readOne.mutate(item.id);
              navigation.navigate("Chat", { conversationId: item.conversationId, name: item.actor.firstName, userId: item.actor.userId });
            }}
            style={[styles.row, { borderBottomColor: colors.border, backgroundColor: item.readAt ? "transparent" : colors.primarySoft }]}
          >
            <View style={[styles.dot, { backgroundColor: item.readAt ? "transparent" : colors.accent }]} />
            <Text style={{ flex: 1, fontWeight: item.readAt ? "400" : "700" }}>{describe(item)}</Text>
            <Text variant="caption" tone="muted">
              {timeLabel(item.createdAt)}
            </Text>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, minHeight: MIN_TOUCH + 12, paddingHorizontal: spacing.lg, borderBottomWidth: StyleSheet.hairlineWidth },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
