import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { AppNotification } from "../../api/types";
import type { TabScreenProps } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme";
import { Button, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { shortTime } from "../../utils/format";

export function NotificationsScreen({ navigation }: TabScreenProps<"Activity">) {
  const queryClient = useQueryClient();
  const query = useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  };
  const markAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refresh });
  const markOne = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refresh });

  function open(n: AppNotification) {
    if (!n.isRead) markOne.mutate(n.id);
    const conversationId = n.data.conversationId;
    if (conversationId) navigation.navigate("Conversation", { conversationId });
  }

  if (query.isLoading)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  if (query.isError)
    return (
      <Screen>
        <ErrorState onRetry={() => void query.refetch()} />
      </Screen>
    );

  return (
    <Screen testID="notifications-screen">
      <View style={styles.header}>
        <Text variant="title">Activité</Text>
        {unread > 0 ? (
          <Button
            label="Tout marquer comme lu"
            variant="ghost"
            compact
            testID="notifications-read-all"
            loading={markAll.isPending}
            onPress={() => markAll.mutate()}
          />
        ) : null}
      </View>
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={colors.accent} />}
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        contentContainerStyle={styles.list}
        ListEmptyComponent={
          <EmptyState
            icon="notifications-outline"
            title="Rien de neuf pour l'instant"
            message="Tes nouveaux matchs et messages apparaîtront ici."
          />
        }
        renderItem={({ item }) => (
          <Pressable
            testID={`notification-${item.type}`}
            accessibilityRole="button"
            accessibilityLabel={`${item.title}. ${item.body}`}
            onPress={() => open(item)}
            style={({ pressed }) => [styles.item, !item.isRead ? styles.unread : null, pressed ? styles.pressed : null]}
          >
            <View style={[styles.icon, item.type === "match.new" ? styles.matchIcon : null]}>
              <Ionicons
                name={item.type === "match.new" ? "heart" : "chatbubble"}
                size={18}
                color={item.type === "match.new" ? colors.accentText : colors.text}
              />
            </View>
            <View style={styles.flex}>
              <Text variant="bodyStrong">{item.title}</Text>
              <Text tone="muted">{item.body}</Text>
            </View>
            <View style={styles.meta}>
              <Text variant="caption" tone="subtle">
                {shortTime(item.createdAt)}
              </Text>
              {!item.isRead ? <View style={styles.dot} accessibilityLabel="Non lu" /> : null}
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: spacing.md
  },
  list: {
    flexGrow: 1,
    gap: spacing.sm,
    paddingBottom: spacing.xl
  },
  item: {
    flexDirection: "row",
    gap: spacing.md,
    alignItems: "center",
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  unread: {
    borderColor: colors.accent
  },
  pressed: {
    backgroundColor: colors.surfacePressed
  },
  icon: {
    width: 38,
    height: 38,
    borderRadius: 19,
    backgroundColor: colors.surfaceRaised,
    alignItems: "center",
    justifyContent: "center"
  },
  matchIcon: {
    backgroundColor: colors.accent
  },
  flex: {
    flex: 1,
    gap: 2
  },
  meta: {
    alignItems: "flex-end",
    gap: spacing.xs
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    backgroundColor: colors.accent
  }
});
