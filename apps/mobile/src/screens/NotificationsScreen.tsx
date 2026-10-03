import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage, listNotifications, markAllNotificationsRead, markNotificationRead, queryKeys, type AppNotification } from "../api/platform";
import { formatListTime } from "../lib/dates";
import type { RootStackParamList } from "../navigation/types";
import { SafeAreaLayout } from "../shared/layout";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Button, Text, colors, radii, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const client = useQueryClient();
  const query = useQuery({ queryKey: queryKeys.notifications, queryFn: listNotifications });
  const refresh = () => client.invalidateQueries({ queryKey: queryKeys.notifications });
  const markRead = useMutation({ mutationFn: markNotificationRead, onSuccess: refresh });
  const markAll = useMutation({ mutationFn: markAllNotificationsRead, onSuccess: refresh });
  const items = query.data?.notifications ?? [];
  const unread = query.data?.unreadCount ?? 0;

  const open = (n: AppNotification) => {
    if (!n.isRead) {
      markRead.mutate(n.id);
    }
    const { matchId, userId } = n.data;
    if ((n.type === "match" || n.type === "message") && matchId && userId) {
      navigation.navigate("Chat", { matchId, userId, name: n.data.name || "Chat" });
    }
  };

  if (query.isLoading) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <LoadingView label="Loading activity…" />
      </SafeAreaLayout>
    );
  }

  return (
    <SafeAreaLayout edges={["top", "left", "right"]}>
      <FlatList
        ListEmptyComponent={
          query.isError ? (
            <ErrorView message={errorMessage(query.error)} onAction={() => void query.refetch()} />
          ) : (
            <EmptyView message="Matches and messages you miss will show up here." testID="notifications-empty" title="Nothing new" />
          )
        }
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="title" weight="bold">
              Activity
            </Text>
            {unread > 0 ? <Button label="Mark all read" loading={markAll.isPending} onPress={() => markAll.mutate()} size="sm" testID="notifications-read-all" variant="ghost" /> : null}
          </View>
        }
        contentContainerStyle={styles.list}
        data={items}
        keyExtractor={(n) => n.id}
        onRefresh={() => void query.refetch()}
        refreshing={query.isRefetching}
        renderItem={({ item }) => (
          <Pressable accessibilityRole="button" onPress={() => open(item)} style={[styles.row, item.isRead ? null : styles.unread]} testID={`notification-${item.type}`}>
            <View style={styles.copy}>
              <Text weight={item.isRead ? "semibold" : "bold"}>{item.title}</Text>
              <Text tone="muted">{item.body}</Text>
            </View>
            <Text tone="muted" variant="caption">
              {formatListTime(item.createdAt)}
            </Text>
          </Pressable>
        )}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.sm, flexGrow: 1 },
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.sm },
  row: { flexDirection: "row", gap: spacing.md, padding: spacing.md, borderRadius: radii.lg, backgroundColor: colors.backgroundElevated, borderWidth: 1, borderColor: colors.border },
  unread: { backgroundColor: colors.surfaceAccent, borderColor: colors.primaryBorder },
  copy: { flex: 1, gap: 2 }
});
