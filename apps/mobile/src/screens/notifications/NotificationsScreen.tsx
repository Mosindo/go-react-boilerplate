import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { markAllNotificationsRead, markNotificationRead } from "../../api/notifications";
import type { AppNotification } from "../../api/types";
import { useNotifications } from "../../hooks/queries";
import { queryClient, queryKeys } from "../../hooks/queryClient";
import type { AppStackParamList } from "../../navigation/types";
import { spacing, useTheme } from "../../theme";
import { Button, Dot, EmptyState, ErrorState, Header, Icon, LoadingState, Screen, Text, type IconName } from "../../ui";
import { formatRelative } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "Notifications">;

const icons: Record<AppNotification["type"], IconName> = { match: "heart-circle", like: "heart", message: "chatbubble-ellipses" };

function refreshCounts() {
  void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
  void queryClient.invalidateQueries({ queryKey: queryKeys.unreadNotifications });
}

export function NotificationsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const notifications = useNotifications();
  const items = useMemo(() => notifications.data?.pages.flatMap((p) => p.notifications) ?? [], [notifications.data]);
  const unread = notifications.data?.pages[0]?.unreadCount ?? 0;

  const open = (n: AppNotification) => {
    if (!n.isRead)
      void markNotificationRead(n.id)
        .then(refreshCounts)
        .catch(() => undefined);
    if ((n.type === "match" || n.type === "message") && n.data.conversationId) {
      navigation.navigate("Chat", { conversationId: n.data.conversationId });
    } else if (n.type === "like") {
      navigation.navigate("Tabs", { screen: "Likes" });
    }
  };

  return (
    <Screen testID="notifications-screen">
      <Header
        onBack={navigation.goBack}
        right={
          unread > 0 ? (
            <Button label="Tout lire" onPress={() => void markAllNotificationsRead().then(refreshCounts)} size="md" variant="ghost" />
          ) : null
        }
        title="Notifications"
      />
      {notifications.isLoading ? (
        <LoadingState />
      ) : notifications.isError ? (
        <ErrorState message={errorMessage(notifications.error)} onRetry={() => void notifications.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={items}
          keyExtractor={(n) => n.id}
          ListEmptyComponent={
            <EmptyState icon="notifications-outline" message="Matchs, likes et messages apparaîtront ici." title="Rien de neuf" />
          }
          onEndReached={() => {
            if (notifications.hasNextPage && !notifications.isFetchingNextPage) void notifications.fetchNextPage();
          }}
          refreshControl={
            <RefreshControl
              onRefresh={() => void notifications.refetch()}
              refreshing={notifications.isRefetching}
              tintColor={colors.primary}
            />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => open(item)}
              style={({ pressed }) => [
                styles.row,
                { backgroundColor: pressed ? colors.surfaceMuted : item.isRead ? "transparent" : colors.primarySoft }
              ]}
              testID={`notification-${item.type}`}
            >
              <View style={[styles.icon, { backgroundColor: colors.surface }]}>
                <Icon color={item.type === "message" ? colors.accent : colors.primary} name={icons[item.type]} />
              </View>
              <View style={styles.text}>
                <Text variant="bodyStrong">{item.title}</Text>
                <Text numberOfLines={2} tone="muted" variant="caption">
                  {item.body}
                </Text>
                <Text tone="subtle" variant="caption">
                  {formatRelative(item.createdAt)}
                </Text>
              </View>
              {!item.isRead ? <Dot /> : null}
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { flexGrow: 1, gap: spacing.xs, paddingBottom: spacing.xxl },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.sm, borderRadius: 16 },
  icon: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center" },
  text: { flex: 1, gap: 2 }
});
