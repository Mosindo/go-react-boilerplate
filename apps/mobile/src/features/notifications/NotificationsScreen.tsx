import React, { useEffect, useMemo } from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { notificationsApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { AppNotification } from "../../api/types";
import { formatListDate } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { Avatar, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "Notifications">;

function describe(n: AppNotification): string {
  const name = n.actor?.firstName ?? "Quelqu'un";
  return n.type === "match" ? `Nouveau match avec ${name} !` : `${name} vous a envoyé un message.`;
}

export function NotificationsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const client = useQueryClient();
  const query = useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);

  const markAll = useMutation({
    mutationFn: notificationsApi.markAllRead,
    onSuccess: () => client.setQueryData(queryKeys.unreadNotifications, 0)
  });

  // Opening the screen marks everything as read (the list keeps showing what was new).
  const { mutate } = markAll;
  const hasUnread = items.some((n) => !n.readAt);
  useEffect(() => {
    if (hasUnread) mutate();
  }, [hasUnread, mutate]);

  return (
    <Screen>
      <Header title="Notifications" onBack={navigation.goBack} />
      {query.isLoading ? (
        <LoadingState />
      ) : query.error ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          contentContainerStyle={styles.list}
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
          refreshing={query.isRefetching}
          onRefresh={() => void query.refetch()}
          ListEmptyComponent={
            <EmptyState icon="notifications-outline" title="Rien de nouveau" message="Vos matchs et messages apparaîtront ici." />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => item.conversationId && navigation.navigate("Chat", { conversationId: item.conversationId })}
              style={({ pressed }) => [
                styles.row,
                { backgroundColor: !item.readAt ? colors.primarySoft : pressed ? colors.surfaceMuted : "transparent" }
              ]}
              testID={`notification-${item.type}`}
            >
              <Avatar path={item.actor?.photoUrl} name={item.actor?.firstName ?? "?"} size={48} />
              <View style={styles.body}>
                <Text weight={item.readAt ? "400" : "600"}>{describe(item)}</Text>
                <Text variant="caption" tone="subtle">
                  {formatListDate(item.createdAt)}
                </Text>
              </View>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { flexGrow: 1, gap: 4, paddingBottom: spacing.xl },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, padding: spacing.sm, borderRadius: radii.md },
  body: { flex: 1, gap: 2 }
});
