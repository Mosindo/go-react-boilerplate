import React, { useEffect } from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { markAllNotificationsRead, markNotificationRead } from "../../api/social";
import type { NotificationItem } from "../../api/types";
import { useNotifications } from "../../hooks/queries";
import { formatListDate } from "../../lib/dates";
import type { MainStackParams } from "../../navigation/types";
import { Avatar, Button, EmptyState, ErrorState, Loading, Text } from "../../ui";
import { spacing } from "../../theme/tokens";
import { useTheme } from "../../theme/ThemeProvider";

export function notificationText(n: NotificationItem): string {
  const who = n.actor?.firstName ?? "Quelqu'un";
  return n.type === "new_match" ? `Vous avez matché avec ${who} 🎉` : `${who} vous a envoyé un message`;
}

export default function NotificationsScreen({
  navigation
}: NativeStackScreenProps<MainStackParams, "Notifications">) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const query = useNotifications();

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ["notifications"] });
  };

  useEffect(() => {
    navigation.setOptions({ title: "Notifications" });
  }, [navigation]);

  if (query.isPending) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  const items = query.data.pages.flatMap((p) => p.notifications);
  const unread = query.data.pages[0]?.unreadCount ?? 0;
  if (items.length === 0) {
    return (
      <EmptyState
        icon="🔔"
        title="Rien pour l'instant"
        message="Vos nouveaux matchs et messages apparaîtront ici."
      />
    );
  }

  const open = (n: NotificationItem) => {
    if (!n.readAt)
      markNotificationRead(n.id)
        .then(refresh)
        .catch(() => undefined);
    if (n.conversationId)
      navigation.navigate("Chat", {
        conversationId: n.conversationId,
        ...(n.actor ? { name: n.actor.firstName } : {})
      });
  };

  return (
    <FlatList
      testID="notifications-list"
      data={items}
      keyExtractor={(n) => n.id}
      refreshing={query.isRefetching}
      onRefresh={() => void query.refetch()}
      onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
      ListHeaderComponent={
        unread > 0 ? (
          <Button
            label="Tout marquer comme lu"
            variant="ghost"
            onPress={() => {
              markAllNotificationsRead()
                .then(refresh)
                .catch(() => undefined);
            }}
            testID="notifications-read-all"
          />
        ) : null
      }
      renderItem={({ item }) => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${notificationText(item)}${item.readAt ? "" : ", non lu"}`}
          onPress={() => open(item)}
          style={[
            styles.row,
            { borderColor: colors.border, backgroundColor: item.readAt ? "transparent" : colors.accentSoft }
          ]}
        >
          <Avatar photoId={item.actor?.photoId} name={item.actor?.firstName} size={48} />
          <View style={styles.flex}>
            <Text bold={!item.readAt}>{notificationText(item)}</Text>
            <Text variant="caption" tone="muted">
              {formatListDate(item.createdAt)}
            </Text>
          </View>
        </Pressable>
      )}
    />
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  flex: { flex: 1 }
});
