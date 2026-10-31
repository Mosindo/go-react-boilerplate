import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getNotifications, markAllNotificationsRead, markNotificationRead } from "../api/dating";
import type { NotificationItem } from "../api/types";
import { formatTime } from "../lib/format";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParamList } from "../navigation/types";
import { Screen } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Button, Text, colors, radii, spacing } from "../shared/ui";

export default function NotificationsScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const queryClient = useQueryClient();
  const query = useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => getNotifications(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore
  });

  const refresh = () => void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  const read = useMutation({ mutationFn: markNotificationRead, onSuccess: refresh });
  const readAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: refresh });

  const items = query.data?.pages.flatMap((p) => p.notifications) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const open = (item: NotificationItem) => {
    if (!item.isRead) read.mutate(item.id);
    const conversationId = item.data.conversationId;
    if (conversationId && (item.type === "match" || item.type === "message")) {
      navigation.navigate("Chat", { conversationId });
    } else if (item.type === "like") {
      navigation.navigate("Tabs", { screen: "Messages" });
    }
  };

  if (query.isLoading)
    return (
      <Screen>
        <LoadingView label="Chargement…" />
      </Screen>
    );
  if (query.isError)
    return (
      <Screen>
        <ErrorView actionLabel="Réessayer" message="Impossible de charger les notifications." onAction={() => void query.refetch()} />
      </Screen>
    );

  return (
    <Screen padded={false}>
      <FlatList
        ListEmptyComponent={<EmptyView message="Matchs, messages et likes apparaîtront ici." title="Rien de neuf" />}
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="heading" weight="bold">
              Notifications
            </Text>
            {unread > 0 ? (
              <Button
                label="Tout marquer comme lu"
                loading={readAll.isPending}
                onPress={() => readAll.mutate()}
                size="sm"
                variant="ghost"
              />
            ) : null}
          </View>
        }
        contentContainerStyle={styles.list}
        data={items}
        keyExtractor={(n) => n.id}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        refreshControl={
          <RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching} tintColor={colors.primary} />
        }
        renderItem={({ item }) => (
          <Pressable accessibilityRole="button" onPress={() => open(item)} style={[styles.row, !item.isRead ? styles.unread : null]}>
            <View style={styles.rowBody}>
              <Text weight={item.isRead ? "regular" : "bold"}>{item.title}</Text>
              {item.body ? (
                <Text numberOfLines={2} tone="muted">
                  {item.body}
                </Text>
              ) : null}
            </View>
            <Text tone="muted" variant="caption">
              {formatTime(item.createdAt)}
            </Text>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { paddingHorizontal: spacing.lg, paddingTop: spacing.md, paddingBottom: spacing.xxxl, gap: spacing.sm },
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingBottom: spacing.sm },
  row: {
    flexDirection: "row",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  unread: { backgroundColor: colors.primarySoft, borderColor: colors.primaryBorder },
  rowBody: { flex: 1, gap: 2 }
});
