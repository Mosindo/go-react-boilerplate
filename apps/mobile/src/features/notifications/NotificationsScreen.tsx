import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi, notificationApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { formatShortTime } from "../../lib/date";
import type { MainStackParamList } from "../../navigation/types";
import { queryKeys } from "../../realtime/cache";
import { radius, spacing, useTheme } from "../../theme/theme";
import { Button, EmptyState, ErrorState, Icon, LoadingState, Screen, Text } from "../../ui";

export function NotificationsScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const queryClient = useQueryClient();
  const { data, error, isLoading, isRefetching, refetch } = useQuery({
    queryKey: queryKeys.notifications,
    queryFn: () => notificationApi.list(),
  });

  const refreshCounters = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    void queryClient.invalidateQueries({ queryKey: queryKeys.notificationsUnread });
  };
  const markRead = useMutation({
    mutationFn: notificationApi.markRead,
    onSuccess: refreshCounters,
  });
  const markAll = useMutation({
    mutationFn: notificationApi.markAllRead,
    onSuccess: refreshCounters,
  });

  const open = async (n: AppNotification) => {
    if (!n.isRead) markRead.mutate(n.id);
    const { conversationId, userId } = n.data;
    if (n.type === "match" && conversationId && userId) {
      // The counterpart's name/photo live in the conversation list.
      const conversations = await queryClient.fetchQuery({
        queryKey: queryKeys.conversations,
        queryFn: () => chatApi.conversations(),
        staleTime: 10_000,
      });
      const conversation = conversations.find((c) => c.id === conversationId);
      if (conversation) {
        navigation.navigate("Chat", {
          conversationId,
          userId,
          name: conversation.user.firstName,
          photoId: conversation.user.photos[0]?.id,
        });
      }
    }
  };

  if (isLoading) return <LoadingState />;
  if (error) return <ErrorState error={error} onRetry={() => void refetch()} />;
  const items = data ?? [];
  const hasUnread = items.some((n) => !n.isRead);

  return (
    <Screen testID="notifications-screen" padded={false} edges={["bottom"]}>
      {items.length === 0 ? (
        <EmptyState
          testID="notifications-empty"
          icon="notifications-off-outline"
          title="Rien pour l'instant"
          message="Vos nouveaux matchs apparaîtront ici."
        />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          refreshControl={
            <RefreshControl refreshing={isRefetching} onRefresh={() => void refetch()} />
          }
          ListHeaderComponent={
            hasUnread ? (
              <View style={styles.headerAction}>
                <Button
                  testID="notifications-read-all"
                  title="Tout marquer comme lu"
                  variant="ghost"
                  onPress={() => markAll.mutate()}
                />
              </View>
            ) : null
          }
          renderItem={({ item }) => (
            <Pressable
              testID={`notification-${item.type}`}
              accessibilityRole="button"
              accessibilityState={{ selected: !item.isRead }}
              onPress={() => void open(item)}
              style={[
                styles.row,
                {
                  borderBottomColor: colors.border,
                  backgroundColor: item.isRead ? colors.bg : colors.surfaceAlt,
                },
              ]}
            >
              <View style={[styles.icon, { backgroundColor: colors.primary }]}>
                <Icon name="heart" size={20} color={colors.onPrimary} />
              </View>
              <View style={styles.body}>
                <Text variant="heading">{item.title}</Text>
                <Text muted>{item.body}</Text>
              </View>
              <Text variant="caption" muted>
                {formatShortTime(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  headerAction: { alignItems: "flex-end", paddingHorizontal: spacing.lg, paddingTop: spacing.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.lg,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  icon: {
    width: 40,
    height: 40,
    borderRadius: radius.pill,
    alignItems: "center",
    justifyContent: "center",
  },
  body: { flex: 1, gap: 2 },
});
