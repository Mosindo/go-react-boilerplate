import React, { useCallback, useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { notificationsApi } from "../../api/dating";
import type { AppNotification } from "../../api/types";
import { keys, useMatches, useNotifications } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Text, colors, spacing } from "../../shared/ui";
import { formatRelative } from "../../utils/format";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export default function ActivityScreen() {
  const navigation = useNavigation<Nav>();
  const client = useQueryClient();
  const query = useNotifications();
  const matches = useMatches();
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;
  const refetch = query.refetch;

  useFocusEffect(
    useCallback(() => {
      void refetch();
    }, [refetch])
  );

  const refreshCounts = () => {
    void client.invalidateQueries({ queryKey: keys.notifications });
    void client.invalidateQueries({ queryKey: keys.unread });
  };

  const open = (n: AppNotification) => {
    if (!n.isRead) {
      notificationsApi
        .read(n.id)
        .then(refreshCounts)
        .catch(() => undefined);
    }
    const match = matches.data?.pages.flatMap((p) => p.matches).find((m) => m.conversationId === n.data.conversationId);
    if (match) {
      navigation.navigate("Chat", {
        conversationId: match.conversationId,
        matchId: match.id,
        userId: match.user.userId,
        firstName: match.user.firstName
      });
    } else {
      navigation.navigate("Tabs", { screen: "Matches" });
    }
  };

  if (query.isLoading) {
    return <LoadingView />;
  }
  if (query.isError) {
    return <ErrorView onRetry={() => void refetch()} />;
  }

  return (
    <Screen contentStyle={styles.content} scroll={false} testID="activity-screen">
      <View style={styles.header}>
        <ScreenHeader
          right={
            unread > 0 ? (
              <Button
                label="Tout lire"
                onPress={() => notificationsApi.readAll().then(refreshCounts)}
                size="sm"
                testID="activity-read-all"
                variant="ghost"
              />
            ) : null
          }
          title="Activité"
        />
      </View>
      {items.length === 0 ? (
        <EmptyView glyph="◌" message="Tes nouveaux matchs et messages s'afficheront ici." title="Rien de neuf" />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={items}
          keyExtractor={(n) => n.id}
          onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
          refreshControl={<RefreshControl onRefresh={() => void refetch()} refreshing={query.isRefetching} tintColor={colors.primary} />}
          renderItem={({ item }) => (
            <Pressable
              onPress={() => open(item)}
              style={({ pressed }) => [styles.row, pressed ? styles.pressed : null]}
              testID={`notification-${item.type}`}
            >
              <View style={[styles.dot, item.isRead ? styles.dotRead : null]} />
              <View style={styles.rowText}>
                <Text weight={item.isRead ? "regular" : "semibold"}>{item.title}</Text>
                <Text tone="muted" variant="label">
                  {item.body}
                </Text>
              </View>
              <Text tone="subtle" variant="caption">
                {formatRelative(item.createdAt)}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: {
    paddingHorizontal: 0,
    paddingBottom: 0
  },
  header: {
    paddingHorizontal: spacing.xl
  },
  list: {
    paddingHorizontal: spacing.xl,
    paddingBottom: spacing.xxxl
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: colors.border
  },
  pressed: {
    opacity: 0.7
  },
  dot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    backgroundColor: colors.primary
  },
  dotRead: {
    backgroundColor: "transparent"
  },
  rowText: {
    flex: 1,
    gap: 2
  }
});
