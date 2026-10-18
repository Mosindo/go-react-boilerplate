import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markAllNotificationsRead, markNotificationRead } from "../api/notifications";
import type { AppNotification } from "../api/types";
import { Screen } from "../components/Screen";
import { queryKeys, useNotifications } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView } from "../shared/feedback";
import { Button, Loader, Text, colors, radii, spacing } from "../shared/ui";
import { formatMessageTime } from "../utils/format";

type Nav = NativeStackNavigationProp<MainStackParamList>;

export default function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const client = useQueryClient();
  const query = useNotifications();

  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;
  const refreshAll = () => client.invalidateQueries({ queryKey: queryKeys.notifications });

  const readOne = useMutation({ mutationFn: (id: string) => markNotificationRead(id), onSuccess: refreshAll });
  const readAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: refreshAll });

  const open = (n: AppNotification) => {
    if (!n.isRead) readOne.mutate(n.id);
    const { conversationId, userId, matchId } = n.data;
    if ((n.type === "match" || n.type === "message") && conversationId && userId) {
      navigation.navigate("Conversation", { conversationId, userId, matchId });
    }
  };

  return (
    <Screen
      contentStyle={styles.content}
      edges={["top", "left", "right"]}
      header={
        <View style={styles.header}>
          <Text variant="title" weight="bold">
            Notifications
          </Text>
          {unread > 0 ? (
            <Button
              label="Tout lire"
              loading={readAll.isPending}
              onPress={() => readAll.mutate()}
              size="sm"
              testID="notifications-read-all"
              variant="ghost"
            />
          ) : null}
        </View>
      }
      testID="notifications-screen"
    >
      {query.isLoading ? <Loader fullScreen label="Chargement…" /> : null}
      {query.isError && !query.data ? (
        <ErrorView
          actionLabel="Réessayer"
          message="Impossible de charger vos notifications."
          onAction={() => void query.refetch()}
          title="Oups"
        />
      ) : null}
      {query.data ? (
        <FlatList
          ListEmptyComponent={
            <EmptyView
              message="Vos matchs et nouveaux messages apparaîtront ici."
              testID="notifications-empty"
              title="Rien de nouveau"
            />
          }
          contentContainerStyle={styles.list}
          data={items}
          keyExtractor={(n) => n.id}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          onEndReachedThreshold={0.4}
          refreshControl={
            <RefreshControl
              onRefresh={() => void query.refetch()}
              refreshing={query.isRefetching && !query.isFetchingNextPage}
              tintColor={colors.primary}
            />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : " Non lu."}`}
              accessibilityRole="button"
              onPress={() => open(item)}
              style={[styles.item, item.isRead ? null : styles.unread]}
              testID={`notification-${item.type}`}
            >
              {!item.isRead ? <View style={styles.dot} /> : <View style={styles.dotSpacer} />}
              <View style={styles.copy}>
                <View style={styles.row}>
                  <Text style={styles.title} weight={item.isRead ? "semibold" : "bold"}>
                    {item.title}
                  </Text>
                  <Text tone="muted" variant="caption">
                    {formatMessageTime(item.createdAt)}
                  </Text>
                </View>
                <Text tone="muted">{item.body}</Text>
              </View>
            </Pressable>
          )}
        />
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { padding: 0, gap: 0 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.lg,
    paddingBottom: spacing.sm
  },
  list: { padding: spacing.lg, gap: spacing.sm, paddingBottom: 100, flexGrow: 1 },
  item: {
    flexDirection: "row",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.backgroundElevated,
    borderWidth: 1,
    borderColor: colors.border
  },
  unread: { backgroundColor: colors.primarySoft, borderColor: colors.primaryBorder },
  dot: { width: 10, height: 10, borderRadius: 5, backgroundColor: colors.primary, marginTop: 7 },
  dotSpacer: { width: 10 },
  copy: { flex: 1, gap: 2 },
  row: { flexDirection: "row", justifyContent: "space-between", gap: spacing.sm },
  title: { flex: 1 }
});
