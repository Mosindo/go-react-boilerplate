import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../../api/notifications";
import { queryKeys } from "../../api/queryKeys";
import type { AppNotification } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { relativeTime } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Button, EmptyState, ErrorState, Header, Icon, Loading, Screen, Text, type IconName } from "../../ui";

type Props = NativeStackScreenProps<AppStackParamList, "Notifications">;

const icons: Record<AppNotification["type"], IconName> = {
  match: "heart",
  message: "chatbubble-ellipses",
  like: "sparkles"
};

export function NotificationsScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const list = useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    void queryClient.invalidateQueries({ queryKey: queryKeys.badges });
  };
  const markRead = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refresh });
  const markAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refresh });
  const items = list.data?.pages.flatMap((p) => p.notifications) ?? [];
  const hasUnread = items.some((n) => !n.readAt);

  const open = (n: AppNotification) => {
    if (!n.readAt) markRead.mutate(n.id);
    if ((n.type === "match" || n.type === "message") && n.data.conversationId) {
      navigation.navigate("Chat", { conversationId: n.data.conversationId });
    } else if (n.type === "like") {
      navigation.navigate("Tabs", { screen: "Likes" });
    }
  };

  return (
    <Screen testID="notifications-screen">
      <Header
        title="Notifications"
        onBack={() => navigation.goBack()}
        right={
          hasUnread ? (
            <Button label="Tout lire" variant="ghost" compact onPress={() => markAll.mutate()} testID="notifications-read-all" />
          ) : null
        }
      />
      {list.isPending ? (
        <Loading />
      ) : list.isError ? (
        <ErrorState message={errorMessage(list.error)} onRetry={() => void list.refetch()} />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          contentContainerStyle={[styles.list, items.length === 0 && styles.grow]}
          refreshing={list.isRefetching}
          onRefresh={() => void list.refetch()}
          onEndReached={() => list.hasNextPage && !list.isFetchingNextPage && void list.fetchNextPage()}
          ListEmptyComponent={
            <EmptyState icon="notifications-outline" title="Rien de neuf" message="Vos matchs, messages et likes apparaîtront ici." />
          }
          renderItem={({ item }) => (
            <Pressable
              onPress={() => open(item)}
              accessibilityRole="button"
              accessibilityLabel={`${item.title}. ${item.body}${item.readAt ? "" : ". Non lue"}`}
              testID="notification-row"
              style={({ pressed }) => [
                styles.row,
                { backgroundColor: item.readAt ? colors.surface : colors.primarySoft, borderColor: colors.border },
                pressed && { opacity: 0.75 }
              ]}
            >
              <View style={[styles.icon, { backgroundColor: item.readAt ? colors.surfaceAlt : colors.primary }]}>
                <Icon name={icons[item.type]} size={18} color={item.readAt ? "textMuted" : "onPrimary"} />
              </View>
              <View style={styles.texts}>
                <Text variant="label">{item.title}</Text>
                <Text variant="caption" color="textMuted" numberOfLines={2}>
                  {item.body}
                </Text>
              </View>
              <Text variant="caption" color="textSubtle">
                {relativeTime(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.xs, paddingBottom: spacing.xl },
  grow: { flexGrow: 1 },
  row: { flexDirection: "row", gap: spacing.sm, alignItems: "center", padding: spacing.md, borderRadius: radii.lg, borderWidth: 1 },
  icon: { width: 38, height: 38, borderRadius: 19, alignItems: "center", justifyContent: "center" },
  texts: { flex: 1, gap: 2 }
});
