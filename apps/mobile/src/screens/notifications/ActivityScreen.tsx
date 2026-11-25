import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { notificationsApi } from "../../api/notifications";
import type { AppNotification } from "../../api/types";
import { queryKeys } from "../../hooks/queryClient";
import { useNotifications } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { formatRelative } from "../../utils/format";

const icons: Record<AppNotification["type"], keyof typeof Ionicons.glyphMap> = {
  match: "sparkles",
  message: "chatbubble-outline",
  system: "information-circle-outline"
};

export function ActivityScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const qc = useQueryClient();
  const notifications = useNotifications();
  const items = useMemo(() => notifications.data?.pages.flatMap((p) => p.notifications) ?? [], [notifications.data]);
  const unread = notifications.data?.pages[0]?.unreadCount ?? 0;

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.notifications });
    void qc.invalidateQueries({ queryKey: queryKeys.notificationsUnread });
  };
  const markRead = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refresh });
  const markAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refresh });

  const open = (n: AppNotification) => {
    if (!n.readAt) {
      markRead.mutate(n.id);
    }
    if (n.type === "match" && n.data.conversationId) {
      navigation.navigate("Chat", { conversationId: n.data.conversationId });
    }
  };

  return (
    <Screen edges={["top"]} padded={false} testID="activity-screen">
      <View style={styles.header}>
        <Header
          title="Activité"
          large
          right={
            unread > 0 ? (
              <Button
                label="Tout lire"
                variant="ghost"
                compact
                onPress={() => markAll.mutate()}
                loading={markAll.isPending}
                testID="activity-read-all"
              />
            ) : null
          }
        />
      </View>
      {notifications.isLoading ? (
        <LoadingState />
      ) : notifications.error ? (
        <ErrorState message={errorMessage(notifications.error)} onRetry={() => void notifications.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState
          icon="notifications-outline"
          title="Rien de neuf"
          body="Vos nouveaux matchs et les informations importantes apparaîtront ici."
        />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          contentContainerStyle={styles.list}
          refreshControl={
            <RefreshControl
              refreshing={notifications.isRefetching}
              onRefresh={() => void notifications.refetch()}
              tintColor={colors.primary}
            />
          }
          onEndReached={() => notifications.hasNextPage && void notifications.fetchNextPage()}
          renderItem={({ item }) => (
            <Pressable
              onPress={() => open(item)}
              accessibilityRole="button"
              accessibilityLabel={`${item.title}. ${item.body}${item.readAt ? "" : ". Non lu"}`}
              style={({ pressed }) => [styles.row, !item.readAt && styles.unread, pressed && styles.pressed]}
              testID={`notification-${item.type}`}
            >
              <View style={styles.icon}>
                <Ionicons name={icons[item.type]} size={20} color={colors.gold} />
              </View>
              <View style={styles.text}>
                <Text variant="bodyStrong">{item.title}</Text>
                <Text variant="small" color="textMuted">
                  {item.body}
                </Text>
                <Text variant="caption" color="textSubtle">
                  {formatRelative(item.createdAt)}
                </Text>
              </View>
              {!item.readAt ? <View style={styles.dot} accessibilityElementsHidden /> : null}
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.md
  },
  list: {
    padding: spacing.lg,
    gap: spacing.sm
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface
  },
  unread: {
    borderWidth: 1,
    borderColor: "rgba(238,122,95,0.35)"
  },
  pressed: {
    backgroundColor: colors.surfaceHigh
  },
  icon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: colors.goldSoft,
    alignItems: "center",
    justifyContent: "center"
  },
  text: {
    flex: 1,
    gap: 2
  },
  dot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    backgroundColor: colors.primary
  }
});
