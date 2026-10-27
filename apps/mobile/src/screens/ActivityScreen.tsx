import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { notificationsApi } from "../api/endpoints";
import type { AppNotification } from "../api/types";
import { formatTime } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { keys } from "../state/keys";
import { useNotifications } from "../state/queries";
import { usePalette } from "../theme";
import { Avatar } from "../ui/AuthImage";
import { Button } from "../ui/Button";
import { Screen } from "../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../ui/states";
import { Text } from "../ui/Text";

export function describeNotification(n: AppNotification): string {
  const name = n.actor?.firstName || "Someone";
  return n.type === "match" ? `You matched with ${name}` : `${name} sent you a message`;
}

export function ActivityScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const p = usePalette();
  const queryClient = useQueryClient();
  const query = useNotifications();
  const items = query.data?.pages.flatMap((page) => page.notifications) ?? [];
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const refreshAll = () => {
    void queryClient.invalidateQueries({ queryKey: keys.notifications });
  };
  const readOne = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refreshAll });
  const readAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refreshAll });

  const open = (n: AppNotification) => {
    if (!n.readAt) readOne.mutate(n.id);
    if (n.refId && n.actor) {
      navigation.navigate("Chat", {
        conversationId: n.refId,
        peer: { userId: n.actor.userId, firstName: n.actor.firstName, photoId: n.actor.photoId }
      });
    }
  };

  if (query.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (query.isError) {
    return (
      <Screen>
        <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen>
      <View style={styles.header}>
        <Text variant="title" accessibilityRole="header">
          Activity
        </Text>
        {unread > 0 ? (
          <Button label="Mark all read" variant="ghost" onPress={() => readAll.mutate()} />
        ) : null}
      </View>
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        ListEmptyComponent={
          <EmptyState title="Nothing yet" message="New matches and messages show up here." />
        }
        refreshControl={
          <RefreshControl
            refreshing={query.isRefetching}
            onRefresh={() => void query.refetch()}
            tintColor={p.primary}
          />
        }
        onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
        renderItem={({ item }) => (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={describeNotification(item)}
            onPress={() => open(item)}
            style={[styles.row, !item.readAt && { backgroundColor: p.primarySoft }]}
          >
            <Avatar size={48} photoId={item.actor?.photoId} label={item.actor?.firstName} />
            <View style={styles.body}>
              <Text style={item.readAt ? undefined : styles.bold}>
                {describeNotification(item)}
              </Text>
              <Text variant="caption" tone="muted">
                {formatTime(item.createdAt)}
              </Text>
            </View>
          </Pressable>
        )}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 20,
    paddingTop: 12
  },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 14,
    paddingHorizontal: 20,
    paddingVertical: 12
  },
  body: { flex: 1, gap: 2 },
  bold: { fontWeight: "700" }
});
