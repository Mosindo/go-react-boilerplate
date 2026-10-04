import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { getErrorMessage } from "../../api/client";
import { queryKeys } from "../../api/queryKeys";
import type { AppNotification, MatchSummary } from "../../api/types";
import { useMarkAllNotificationsRead, useMarkNotificationRead, useNotifications } from "../../hooks/useNotifications";
import { flattenPages } from "../../lib/pagination";
import { formatRelative } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { MIN_TOUCH, radii, spacing, useColors } from "../../theme";
import { Button, EmptyState, ErrorState, Icon, ListSkeleton, Text, toast } from "../../ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const colors = useColors();
  const client = useQueryClient();
  const list = useNotifications();
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();
  const hasUnread = list.items.some((n) => !n.readAt);

  function open(n: AppNotification) {
    if (!n.readAt) markRead.mutate(n.id, { onError: (e) => toast.error(getErrorMessage(e)) });
    const { conversationId, userId } = n.data;
    if (n.type === "match" && conversationId) {
      const known = flattenPages(
        client.getQueryData<{
          pages: { matches: MatchSummary[]; nextCursor: string | null }[];
          pageParams: (string | null)[];
        }>(queryKeys.matches),
        "matches"
      ).find((m) => m.conversationId === conversationId);
      if (known) {
        navigation.navigate("Chat", { match: known });
        return;
      }
    }
    if (userId) navigation.navigate("ProfileDetail", { userId, source: "match" });
    else if (n.type === "match") navigation.navigate("Tabs", { screen: "Matches" });
  }

  let body: React.ReactNode;
  if (list.isPending) body = <ListSkeleton />;
  else if (list.isError && list.items.length === 0)
    body = <ErrorState error={list.error} onRetry={() => void list.refetch()} />;
  else if (list.items.length === 0) {
    body = (
      <EmptyState
        icon="notifications-outline"
        message="New matches and important updates will show up here."
        title="You're all caught up"
      />
    );
  } else {
    body = (
      <FlatList
        ListFooterComponent={list.isFetchingNextPage ? <ListSkeleton rows={2} /> : null}
        data={list.items}
        keyExtractor={(item) => item.id}
        onEndReached={() => {
          if (list.hasNextPage && !list.isFetchingNextPage) void list.fetchNextPage();
        }}
        onEndReachedThreshold={0.4}
        refreshControl={
          <RefreshControl
            onRefresh={() => void list.refetch()}
            refreshing={list.isRefetching && !list.isFetchingNextPage}
            tintColor={colors.primary}
          />
        }
        renderItem={({ item }) => {
          const unread = !item.readAt;
          return (
            <Pressable
              accessibilityLabel={`${item.title}. ${item.body}. ${unread ? "Unread." : ""}`}
              accessibilityRole="button"
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, pressed && { backgroundColor: colors.surfaceMuted }]}
            >
              <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
                <Icon color={colors.primary} name={item.type === "match" ? "heart" : "megaphone-outline"} size={20} />
              </View>
              <View style={styles.text}>
                <Text weight={unread ? "bold" : "medium"}>{item.title}</Text>
                <Text numberOfLines={2} tone="muted" variant="small">
                  {item.body}
                </Text>
              </View>
              <View style={styles.meta}>
                <Text tone="muted" variant="caption">
                  {formatRelative(item.createdAt)}
                </Text>
                {unread ? <View style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
              </View>
            </Pressable>
          );
        }}
      />
    );
  }

  return (
    <SafeAreaView edges={["top", "left", "right"]} style={[styles.screen, { backgroundColor: colors.background }]}>
      <View style={styles.header}>
        <Text variant="heading">Notifications</Text>
        {hasUnread ? (
          <Button
            fullWidth={false}
            label="Mark all read"
            loading={markAll.isPending}
            onPress={() => markAll.mutate(undefined, { onError: (e) => toast.error(getErrorMessage(e)) })}
            variant="ghost"
          />
        ) : null}
      </View>
      <View style={styles.screen}>{body}</View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.sm,
    minHeight: MIN_TOUCH + 20
  },
  icon: { width: 40, height: 40, borderRadius: radii.pill, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 },
  meta: { alignItems: "flex-end", gap: 6 },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
