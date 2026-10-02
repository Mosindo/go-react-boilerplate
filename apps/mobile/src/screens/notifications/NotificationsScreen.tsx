import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";

import { notificationsApi } from "../../api/endpoints";
import type { NotificationItem } from "../../api/types";
import { formatShortTime } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { keys } from "../../realtime/RealtimeProvider";
import { spacing, useTheme } from "../../theme";
import { Button } from "../../ui/Button";
import { PhotoImage } from "../../ui/PhotoImage";
import { Screen } from "../../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../../ui/States";
import { Text } from "../../ui/Text";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export function NotificationsScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const qc = useQueryClient();

  const query = useInfiniteQuery({
    queryKey: keys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
  });

  const refreshCounters = () => {
    void qc.invalidateQueries({ queryKey: keys.notifications });
    void qc.invalidateQueries({ queryKey: keys.summary });
  };
  const markAll = useMutation({
    mutationFn: notificationsApi.markAllRead,
    onSuccess: refreshCounters,
  });
  const markOne = useMutation({
    mutationFn: notificationsApi.markRead,
    onSuccess: refreshCounters,
  });

  const items = query.data?.pages.flatMap((p) => p.notifications) ?? [];
  const unread = query.data?.pages[0]?.unread ?? 0;

  const open = (n: NotificationItem) => {
    if (!n.readAt) markOne.mutate(n.id);
    navigation.navigate("Chat", {
      matchId: n.matchId,
      name: n.actor?.firstName ?? "",
      userId: n.actor?.id ?? "",
    });
  };

  if (query.isLoading) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={styles.header}>
        <Text variant="title">Notifications</Text>
        {unread > 0 ? (
          <Button label="Tout lire" variant="ghost" onPress={() => markAll.mutate()} />
        ) : null}
      </View>
      {items.length === 0 ? (
        <EmptyState
          icon="notifications-outline"
          title="Rien pour le moment"
          message="Vos nouveaux matchs et messages apparaîtront ici."
        />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          refreshControl={
            <RefreshControl
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              onRefresh={() => void query.refetch()}
              tintColor={colors.primary}
            />
          }
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          renderItem={({ item }) => {
            const name = item.actor?.firstName ?? "Quelqu'un";
            return (
              <Pressable
                onPress={() => open(item)}
                accessibilityRole="button"
                style={({ pressed }) => [
                  styles.row,
                  { backgroundColor: item.readAt ? "transparent" : colors.primarySoft },
                  pressed && { opacity: 0.8 },
                ]}
              >
                <PhotoImage photo={item.actor?.photos[0]} name={name} round size={52} />
                <View style={styles.text}>
                  <Text>
                    {item.type === "match"
                      ? `Vous avez un match avec ${name} !`
                      : `${name} vous a écrit.`}
                  </Text>
                  <Text variant="caption" tone="muted">
                    {formatShortTime(item.createdAt)}
                  </Text>
                </View>
              </Pressable>
            );
          }}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm,
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
  },
  text: { flex: 1, gap: 2 },
});
