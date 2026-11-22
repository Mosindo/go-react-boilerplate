import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { markAllNotificationsRead, markNotificationRead } from "../../api/dating";
import type { Notification } from "../../api/types";
import { qk, useNotifications } from "../../hooks/queries";
import { formatListTime } from "../../lib/format";
import type { MainScreenProps } from "../../navigation/types";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text, colors, radii, showToast, spacing } from "../../ui";

const icons: Record<string, keyof typeof Ionicons.glyphMap> = {
  "match.created": "sparkles",
  "like.received": "heart"
};

export default function NotificationsScreen({ navigation }: MainScreenProps<"Notifications">) {
  const client = useQueryClient();
  const notifications = useNotifications();

  const open = async (n: Notification) => {
    if (!n.isRead) {
      markNotificationRead(n.id)
        .then(() => client.invalidateQueries({ queryKey: qk.notifications }))
        .catch(() => undefined);
    }
    if (n.type === "match.created" && n.data.userId) {
      navigation.navigate("ProfileDetail", { userId: n.data.userId, context: "match" });
    } else if (n.type === "like.received") {
      navigation.navigate("Tabs", { screen: "Likes" });
    }
  };

  const readAll = async () => {
    try {
      await markAllNotificationsRead();
      await client.invalidateQueries({ queryKey: qk.notifications });
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  const unread = notifications.data?.unreadCount ?? 0;

  return (
    <Screen testID="notifications-screen">
      <Header
        onBack={navigation.goBack}
        right={
          unread > 0 ? <Button compact label="Tout lire" onPress={readAll} testID="notifications-read-all" variant="ghost" /> : undefined
        }
        title="Notifications"
      />
      {notifications.isPending ? (
        <LoadingState />
      ) : notifications.isError ? (
        <ErrorState message={errorMessage(notifications.error)} onAction={() => void notifications.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={notifications.data.notifications.length ? styles.list : styles.empty}
          data={notifications.data.notifications}
          keyExtractor={(n) => n.id}
          ListEmptyComponent={
            <EmptyState icon="notifications-outline" message="Vos matchs et likes apparaîtront ici." title="Rien de neuf" />
          }
          refreshControl={
            <RefreshControl
              onRefresh={() => void notifications.refetch()}
              refreshing={notifications.isRefetching}
              tintColor={colors.accent}
            />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => void open(item)}
              style={({ pressed }) => [styles.row, !item.isRead ? styles.unread : null, pressed ? styles.pressed : null]}
              testID={`notification-${item.type}`}
            >
              <View style={styles.icon}>
                <Ionicons color={colors.accent} name={icons[item.type] ?? "notifications"} size={18} />
              </View>
              <View style={styles.body}>
                <Text variant="label">{item.title}</Text>
                <Text tone="muted" variant="caption">
                  {item.body}
                </Text>
              </View>
              <Text tone="subtle" variant="caption">
                {formatListTime(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: spacing.xs,
    paddingBottom: spacing.xl
  },
  empty: {
    flexGrow: 1
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    padding: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.surface
  },
  unread: {
    borderLeftWidth: 3,
    borderLeftColor: colors.accent
  },
  pressed: {
    opacity: 0.85
  },
  icon: {
    width: 36,
    height: 36,
    borderRadius: 18,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.accentSoft
  },
  body: {
    flex: 1,
    gap: 2
  }
});
