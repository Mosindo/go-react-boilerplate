import React from "react";
import { useNavigation } from "@react-navigation/native";
import type { BottomTabNavigationProp } from "@react-navigation/bottom-tabs";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification } from "../../api/types";
import { keys, useNotifications } from "../../hooks/queries";
import type { MainStackParams, TabParams } from "../../navigation/types";
import type { CompositeNavigationProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { Screen } from "../../shared/layout";
import { Button, EmptyState, ErrorState, ListSkeleton, Text, spacing, useColors, useToast } from "../../shared/ui";
import { relativeTime } from "../../utils/time";

type Nav = CompositeNavigationProp<BottomTabNavigationProp<TabParams, "Notifications">, NativeStackNavigationProp<MainStackParams>>;

export function NotificationsScreen() {
  const navigation = useNavigation<Nav>();
  const qc = useQueryClient();
  const toast = useToast();
  const c = useColors();
  const query = useNotifications();
  const items = query.data?.pages.flatMap((p) => p.notifications) ?? [];
  const unread = query.data?.pages[0]?.unread ?? 0;

  const refresh = () => qc.invalidateQueries({ queryKey: keys.notifications });
  const markOne = useMutation({ mutationFn: (id: string) => notificationsApi.markRead(id), onSuccess: refresh });
  const markAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refresh, onError: (err) => toast.show(errorMessage(err), "error") });

  const open = (n: AppNotification) => {
    if (!n.isRead) markOne.mutate(n.id);
    const conversationId = n.data.conversationId;
    if ((n.type === "match" || n.type === "new_message") && conversationId) {
      const cached = qc.getQueryData<{ pages: { conversations: { id: string; with: { firstName: string; userId: string } }[] }[] }>(keys.conversations);
      const conv = cached?.pages.flatMap((p) => p.conversations).find((x) => x.id === conversationId);
      navigation.navigate("Chat", { conversationId, name: conv?.with.firstName ?? "Conversation", userId: conv?.with.userId ?? n.data.userId ?? "" });
    } else if (n.type === "like") {
      navigation.navigate("Likes");
    }
  };

  let body: React.ReactNode;
  if (query.isPending) body = <ListSkeleton />;
  else if (query.isError) body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  else if (items.length === 0) body = <EmptyState title="Rien de neuf" message="Vos matchs, likes et nouveaux messages apparaîtront ici." />;
  else {
    body = (
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl refreshing={query.isRefetching && !query.isFetchingNextPage} onRefresh={() => void query.refetch()} />}
        onEndReached={() => { if (query.hasNextPage && !query.isFetchingNextPage) void query.fetchNextPage(); }}
        renderItem={({ item }) => (
          <Pressable onPress={() => open(item)} accessibilityRole="button" accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : ". Non lue"}`} style={[styles.row, { borderBottomColor: c.border, backgroundColor: item.isRead ? "transparent" : c.surfaceMuted }]}>
            <View style={[styles.dot, { backgroundColor: item.isRead ? "transparent" : c.accent }]} />
            <View style={{ flex: 1, gap: 2 }}>
              <Text variant="label">{item.title}</Text>
              <Text tone="muted">{item.body}</Text>
            </View>
            <Text variant="caption" tone="muted">{relativeTime(item.createdAt)}</Text>
          </Pressable>
        )}
      />
    );
  }

  return (
    <Screen padded={false}>
      <View style={styles.header}>
        <Text variant="heading">Notifications</Text>
        {unread > 0 ? <Button title="Tout marquer comme lu" variant="ghost" onPress={() => markAll.mutate()} loading={markAll.isPending} testID="notifications-read-all" /> : null}
      </View>
      {body}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", padding: spacing.lg, paddingBottom: spacing.sm },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md, paddingHorizontal: spacing.lg, paddingVertical: spacing.md, minHeight: 64, borderBottomWidth: StyleSheet.hairlineWidth },
  dot: { width: 10, height: 10, borderRadius: 5 },
});
