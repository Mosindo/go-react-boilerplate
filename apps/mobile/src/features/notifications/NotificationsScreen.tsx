import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../../api/social";
import { queryKeys } from "../../api/keys";
import type { NotificationItem } from "../../api/types";
import { formatTime } from "../../lib/format";
import { flattenPages } from "../../lib/pagination";
import type { MainStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Avatar, Button, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { useNotificationsQuery } from "./hooks";

type Props = NativeStackScreenProps<MainStackParamList, "Notifications">;

export function describeNotification(n: NotificationItem): string {
  const who = n.actor?.firstName ?? "Someone";
  return n.type === "match" ? `You matched with ${who}` : `${who} sent you a message`;
}

export function NotificationsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const qc = useQueryClient();
  const query = useNotificationsQuery();
  const items = flattenPages(
    query.data?.pages.map((p) => ({ items: p.notifications, nextCursor: p.nextCursor }))
  );
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => qc.invalidateQueries({ queryKey: queryKeys.notifications });
  const readOne = useMutation({ mutationFn: markNotificationRead, onSuccess: refresh });
  const readAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: refresh });

  const open = (n: NotificationItem) => {
    if (!n.readAt) readOne.mutate(n.id);
    const conversationId = typeof n.data.conversationId === "string" ? n.data.conversationId : null;
    if (conversationId && n.actor) {
      navigation.navigate("Thread", { conversationId, userId: n.actor.userId, name: n.actor.firstName });
    }
  };

  return (
    <Screen edges={["bottom"]}>
      {unread > 0 ? (
        <Button
          label="Mark all as read"
          variant="ghost"
          onPress={() => readAll.mutate()}
          loading={readAll.isPending}
        />
      ) : null}
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState title="Nothing new" message="Matches and messages will show up here." />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => open(item)}
              style={[styles.row, { borderBottomColor: colors.border }]}
            >
              <Avatar url={item.actor?.photoUrl ?? null} name={item.actor?.firstName ?? "?"} size={44} />
              <View style={styles.flex}>
                <Text style={item.readAt ? undefined : { fontWeight: "700" }}>
                  {describeNotification(item)}
                </Text>
                <Text variant="caption" tone="muted">
                  {formatTime(item.createdAt)}
                </Text>
              </View>
              {!item.readAt ? (
                <View style={[styles.dot, { backgroundColor: colors.accent }]} accessibilityLabel="Unread" />
              ) : null}
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  flex: { flex: 1, marginHorizontal: spacing.md },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
