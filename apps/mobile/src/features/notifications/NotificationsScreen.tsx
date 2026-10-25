import React, { useCallback, useMemo } from "react";
import { FlatList, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { AppNotification } from "../../api";
import { formatListTimestamp } from "../../shared/format";
import { flattenPages } from "../../shared/pagination";
import { Avatar, Button, EmptyState, ErrorState, ListRow, Screen, Skeleton, Text } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";
import type { MainStackParamList } from "../../navigation/types";
import { useMarkAllNotificationsRead, useMarkNotificationRead, useNotifications } from "./hooks";

type Navigation = NativeStackNavigationProp<MainStackParamList>;

function describe(notification: AppNotification): string {
  const name = notification.actor?.firstName ?? "Quelqu'un";
  return notification.type === "match" ? `Nouveau match avec ${name} !` : `${name} vous a écrit un message`;
}

const NotificationRow = React.memo(function NotificationRow({
  item,
  onOpen
}: {
  item: AppNotification;
  onOpen: (item: AppNotification) => void;
}) {
  const { colors } = useTheme();
  const unread = item.readAt === null;
  return (
    <ListRow
      accessibilityLabel={`${describe(item)}${unread ? ", non lue" : ""}`}
      left={<Avatar name={item.actor?.firstName} size={52} thumbUrl={item.actor?.thumbUrl} />}
      onPress={() => onOpen(item)}
      right={
        <View style={styles.meta}>
          <Text color="textMuted" variant="caption">
            {formatListTimestamp(item.createdAt)}
          </Text>
          {unread ? <View style={[styles.dot, { backgroundColor: colors.primary }]} /> : null}
        </View>
      }
      subtitle={item.type === "match" ? "Dites-lui bonjour" : "Appuyez pour répondre"}
      testID={`notification-${item.id}`}
      title={describe(item)}
    />
  );
});

export function NotificationsScreen() {
  const navigation = useNavigation<Navigation>();
  const query = useNotifications();
  const markRead = useMarkNotificationRead();
  const markAll = useMarkAllNotificationsRead();
  const { mutate: markOneRead } = markRead;

  const items = useMemo(() => flattenPages(query.data), [query.data]);
  const unreadCount = query.data?.pages[0]?.unreadCount ?? 0;

  const open = useCallback(
    (item: AppNotification) => {
      if (item.readAt === null) {
        markOneRead(item.id);
      }
      if (item.conversationId && item.actor) {
        navigation.navigate("Chat", {
          conversationId: item.conversationId,
          userId: item.actor.userId,
          firstName: item.actor.firstName,
          thumbUrl: item.actor.thumbUrl
        });
      }
    },
    [markOneRead, navigation]
  );

  const renderItem = useCallback(
    ({ item }: { item: AppNotification }) => <NotificationRow item={item} onOpen={open} />,
    [open]
  );

  return (
    <Screen edges={["top"]} padded={false} testID="notifications-screen">
      <View style={styles.header}>
        <Text variant="title">Notifications</Text>
        {unreadCount > 0 ? (
          <Button
            label="Tout lire"
            loading={markAll.isPending}
            onPress={() => markAll.mutate()}
            testID="notifications-read-all"
            variant="ghost"
          />
        ) : null}
      </View>
      {query.isLoading ? (
        <View style={styles.skeletons}>
          {[0, 1, 2, 3].map((key) => (
            <Skeleton height={52} key={key} />
          ))}
        </View>
      ) : query.isError ? (
        <ErrorState onRetry={() => void query.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState
          icon="notifications-outline"
          message="Vos matchs et messages apparaîtront ici."
          title="Rien de neuf"
        />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={items}
          keyExtractor={(item) => item.id}
          onEndReached={() => {
            if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage();
          }}
          onEndReachedThreshold={0.5}
          refreshControl={<RefreshControl onRefresh={() => void query.refetch()} refreshing={query.isRefetching} />}
          renderItem={renderItem}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.xl,
    paddingBottom: spacing.md
  },
  list: { paddingHorizontal: spacing.xl, paddingBottom: spacing.xxl },
  skeletons: { paddingHorizontal: spacing.xl, gap: spacing.lg },
  meta: { alignItems: "flex-end", gap: spacing.xs },
  dot: { width: 10, height: 10, borderRadius: radius.pill }
});
