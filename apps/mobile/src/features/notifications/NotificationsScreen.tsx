import React from "react";
import { FlatList, Pressable, RefreshControl, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { notificationsApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { AppNotification } from "../../api/types";
import type { RootStackParamList, TabParamList } from "../../navigation/types";
import { makeStyles, useTheme } from "../../theme";
import { Button, EmptyState, ErrorState, Icon, LoadingState, Screen, showToast, Text } from "../../ui";
import { formatListTime } from "../../utils/format";
import { useNotifications } from "./useNotifications";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Notifications">, NativeStackScreenProps<RootStackParamList>>;

export function NotificationsScreen({ navigation }: Props) {
  const styles = useStyles();
  const theme = useTheme();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const items = query.data?.pages.flatMap((p) => p.notifications) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const open = async (n: AppNotification) => {
    if (!n.isRead) {
      try {
        await notificationsApi.markRead(n.id);
        void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      } catch {
        // Not blocking.
      }
    }
    const otherId = n.data.userId;
    if (otherId && (n.type === "match.new" || n.type === "message.new")) {
      navigation.navigate("Conversation", { userId: otherId });
    }
  };

  const markAll = async () => {
    try {
      await notificationsApi.markAllRead();
      await queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  if (query.isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (query.isError) {
    return (
      <Screen>
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen padded={false}>
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} tintColor={theme.colors.primary} />
        }
        onEndReached={() => {
          if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
        }}
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="title" accessibilityRole="header">
              Notifications
            </Text>
            {unread > 0 ? (
              <Button
                testID="notifications-read-all"
                label="Tout marquer comme lu"
                variant="ghost"
                size="sm"
                onPress={() => void markAll()}
              />
            ) : null}
          </View>
        }
        ListEmptyComponent={
          <EmptyState icon="notifications-outline" title="Rien de neuf" message="Vos matchs et nouveaux messages apparaîtront ici." />
        }
        renderItem={({ item }) => (
          <Pressable
            testID={`notification-${item.id}`}
            accessibilityRole="button"
            accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : ". Non lue"}`}
            onPress={() => void open(item)}
            style={({ pressed }) => [styles.row, !item.isRead ? styles.unread : null, pressed ? styles.pressed : null]}
          >
            <View style={styles.icon}>
              <Icon
                name={item.type === "match.new" ? "heart" : item.type === "message.new" ? "chatbubble" : "notifications"}
                size={18}
                color="primary"
              />
            </View>
            <View style={styles.texts}>
              <Text variant="bodyStrong">{item.title}</Text>
              <Text variant="caption" tone="muted">
                {item.body}
              </Text>
            </View>
            <View style={styles.meta}>
              <Text variant="caption" tone="subtle">
                {formatListTime(item.createdAt)}
              </Text>
              {!item.isRead ? <View style={styles.dot} /> : null}
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  list: { paddingBottom: t.spacing.xxxl, flexGrow: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: t.spacing.xl,
    paddingTop: t.spacing.lg,
    paddingBottom: t.spacing.sm
  },
  row: { flexDirection: "row", gap: t.spacing.md, alignItems: "center", paddingHorizontal: t.spacing.xl, paddingVertical: t.spacing.md },
  unread: { backgroundColor: t.colors.primarySoft },
  pressed: { opacity: 0.8 },
  icon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: t.colors.surface,
    borderWidth: 1,
    borderColor: t.colors.border,
    alignItems: "center",
    justifyContent: "center"
  },
  texts: { flex: 1, gap: 2 },
  meta: { alignItems: "flex-end", gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 4, backgroundColor: t.colors.primary }
}));
