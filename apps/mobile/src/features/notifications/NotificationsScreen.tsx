import React from "react";
import { FlatList, Pressable, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useNavigation } from "@react-navigation/native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { errorMessage, formatRelative } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { Button, EmptyState, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { useNotifications } from "./queries";

export function NotificationsScreen() {
  const navigation = useNavigation();
  const { colors, radii, spacing } = useTheme();
  const queryClient = useQueryClient();
  const notifications = useNotifications();
  const refresh = () => void queryClient.invalidateQueries({ queryKey: keys.notifications });
  const markRead = useMutation({ mutationFn: notificationsApi.markRead, onSettled: refresh });
  const markAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSettled: refresh });

  const open = (n: AppNotification) => {
    if (!n.isRead) {
      markRead.mutate(n.id);
    }
    if (n.data.conversationId) {
      navigation.navigate("Chat", { conversationId: n.data.conversationId });
    }
  };

  return (
    <Screen>
      <Header
        back
        right={notifications.data?.unreadCount ? <Button label="Tout lire" onPress={() => markAll.mutate()} size="sm" variant="ghost" /> : undefined}
        title="Notifications"
      />
      {notifications.isLoading ? (
        <LoadingState />
      ) : notifications.isError ? (
        <ErrorState message={errorMessage(notifications.error)} onRetry={() => void notifications.refetch()} />
      ) : (
        <FlatList
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm, flexGrow: 1 }}
          data={notifications.data?.notifications ?? []}
          keyExtractor={(n) => n.id}
          ListEmptyComponent={<EmptyState icon="notifications-off-outline" message="Tes nouveaux matchs et messages apparaîtront ici." title="Rien de neuf" />}
          onRefresh={() => void notifications.refetch()}
          refreshing={notifications.isRefetching}
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => open(item)}
              style={({ pressed }) => ({
                flexDirection: "row",
                gap: spacing.sm,
                padding: spacing.md,
                borderRadius: radii.md,
                backgroundColor: pressed ? colors.surfaceAlt : item.isRead ? colors.surface : colors.primarySoft,
                borderWidth: 1,
                borderColor: colors.border
              })}
              testID={`notification-${item.type}`}
            >
              <Ionicons color={colors.primary} name={item.type === "match" ? "heart" : "chatbubble"} size={22} />
              <View style={{ flex: 1, gap: 2 }}>
                <Text variant="bodyStrong">{item.title}</Text>
                <Text numberOfLines={2} tone="muted">
                  {item.body}
                </Text>
                <Text tone="subtle" variant="caption">
                  {formatRelative(item.createdAt)}
                </Text>
              </View>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}
