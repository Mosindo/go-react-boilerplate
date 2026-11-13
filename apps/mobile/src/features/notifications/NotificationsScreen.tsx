import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { notificationsApi } from "../../api/notifications";
import type { AppNotification, NotificationsPage } from "../../api/types";
import type { TabScreenProps } from "../../navigation/types";
import { useNotifications } from "../../state/queries";
import { queryKeys } from "../../state/queryClient";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, EmptyState, ErrorState, LoadingState, Screen, Text } from "../../ui";
import { describeError } from "../../utils/errors";
import { formatRelative } from "../../utils/format";

const icons: Record<string, keyof typeof Ionicons.glyphMap> = {
  match: "heart",
  message: "chatbubble-ellipses"
};

export default function NotificationsScreen({ navigation }: TabScreenProps<"Activity">) {
  const { colors, spacing, radii } = useTheme();
  const client = useQueryClient();
  const notifications = useNotifications();

  const markLocal = (predicate: (n: AppNotification) => boolean) =>
    client.setQueryData<NotificationsPage>(queryKeys.notifications, (page) => {
      if (!page) return page;
      const items = page.notifications.map((n) => (predicate(n) ? { ...n, isRead: true } : n));
      return { notifications: items, unreadCount: items.filter((n) => !n.isRead).length };
    });

  const open = (n: AppNotification) => {
    if (!n.isRead) {
      markLocal((x) => x.id === n.id);
      void notificationsApi.markRead(n.id).catch(() => undefined);
    }
    const peer = n.data.peerUserId;
    if (peer) {
      navigation.navigate("Chat", { userId: peer });
    }
  };

  const markAll = () => {
    markLocal(() => true);
    void notificationsApi.markAllRead().catch(() => void notifications.refetch());
  };

  if (notifications.isLoading) {
    return (
      <Screen edges={["top"]}>
        <LoadingState />
      </Screen>
    );
  }
  if (notifications.isError) {
    return (
      <Screen edges={["top"]}>
        <ErrorState message={describeError(notifications.error)} onAction={() => void notifications.refetch()} />
      </Screen>
    );
  }
  const page = notifications.data;

  return (
    <Screen edges={["top"]} testID="notifications-screen">
      <View style={[styles.header, { paddingVertical: spacing.sm }]}>
        <Text variant="title" accessibilityRole="header">
          Activité
        </Text>
        {page && page.unreadCount > 0 ? (
          <Button label="Tout lire" variant="ghost" size="md" onPress={markAll} testID="notifications-read-all" />
        ) : null}
      </View>
      <FlatList
        data={page?.notifications ?? []}
        keyExtractor={(n) => n.id}
        contentContainerStyle={{ gap: spacing.xs, paddingBottom: spacing.xl, flexGrow: 1 }}
        refreshControl={
          <RefreshControl
            refreshing={notifications.isRefetching}
            onRefresh={() => void notifications.refetch()}
            tintColor={colors.primary}
          />
        }
        ListEmptyComponent={
          <EmptyState
            icon="notifications-outline"
            title="Rien de neuf"
            message="Vos matchs et nouveaux messages s'afficheront ici."
          />
        }
        renderItem={({ item }) => (
          <Pressable
            onPress={() => open(item)}
            accessibilityRole="button"
            accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : ". Non lue"}`}
            style={({ pressed }) => [
              styles.item,
              {
                backgroundColor: item.isRead ? "transparent" : colors.primarySoft,
                borderRadius: radii.md,
                padding: spacing.sm,
                opacity: pressed ? 0.7 : 1
              }
            ]}
            testID={`notification-${item.type}`}
          >
            <View style={[styles.icon, { backgroundColor: colors.surface }]}>
              <Ionicons name={icons[item.type] ?? "notifications"} size={20} color={colors.primary} />
            </View>
            <View style={styles.flex}>
              <Text weight={item.isRead ? "regular" : "semibold"}>{item.title}</Text>
              <Text variant="label" tone="muted">
                {item.body}
              </Text>
              <Text variant="caption" tone="subtle">
                {formatRelative(item.createdAt)}
              </Text>
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, gap: 2 },
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  item: { flexDirection: "row", gap: 12, alignItems: "flex-start" },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" }
});
