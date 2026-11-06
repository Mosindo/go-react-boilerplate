import React from "react";
import { FlatList, Pressable, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../api/services";
import type { AppNotification } from "../api/types";
import { Button, Divider, EmptyState, ErrorState, Loader, Screen, Text } from "../components/ui";
import { fr } from "../i18n/fr";
import { formatListTime } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { spacing, useTheme } from "../theme";

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: ["notifications"],
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextBefore
  });
}

export function NotificationsScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const items = query.data?.pages.flatMap((p) => p.notifications) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => queryClient.invalidateQueries({ queryKey: ["notifications"] });

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      await notificationsApi.markRead(n.id).catch(() => undefined);
      void refresh();
    }
    const conversationId = n.data["conversationId"];
    if (conversationId) navigation.navigate("Chat", { conversationId });
  };

  return (
    <Screen>
      <View style={{ paddingHorizontal: spacing.lg, paddingTop: spacing.sm, flexDirection: "row", alignItems: "center", justifyContent: "space-between" }}>
        <Text variant="title">{fr.notifications.title}</Text>
        {unread > 0 ? (
          <Button
            testID="notifications-read-all"
            variant="ghost"
            title={fr.notifications.markAll}
            onPress={() => void notificationsApi.markAllRead().then(refresh)}
          />
        ) : null}
      </View>
      {query.isLoading ? <Loader /> : null}
      {query.isError ? <ErrorState onRetry={() => void query.refetch()} /> : null}
      {query.data && items.length === 0 ? (
        <EmptyState glyph="🔔" title={fr.notifications.emptyTitle} body={fr.notifications.emptyBody} />
      ) : null}
      {items.length > 0 ? (
        <FlatList
          testID="notification-list"
          data={items}
          keyExtractor={(n) => n.id}
          ItemSeparatorComponent={Divider}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          onEndReachedThreshold={0.4}
          refreshing={query.isRefetching}
          onRefresh={() => void query.refetch()}
          renderItem={({ item }) => (
            <Pressable
              testID={`notification-${item.type}`}
              accessibilityRole="button"
              accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : " (non lu)"}`}
              onPress={() => void open(item)}
              style={({ pressed }) => ({
                flexDirection: "row",
                gap: spacing.md,
                padding: spacing.lg,
                backgroundColor: pressed ? colors.surfaceAlt : item.isRead ? "transparent" : colors.primarySoft
              })}
            >
              <View style={{ width: 10, height: 10, borderRadius: 5, marginTop: 7, backgroundColor: item.isRead ? "transparent" : colors.primary }} />
              <View style={{ flex: 1, gap: 2 }}>
                <Text variant="heading">{item.title}</Text>
                <Text muted>{item.body}</Text>
              </View>
              <Text variant="caption" muted>
                {formatListTime(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      ) : null}
    </Screen>
  );
}
