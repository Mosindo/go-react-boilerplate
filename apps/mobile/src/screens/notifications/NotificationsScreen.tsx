import React, { useMemo } from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient, type InfiniteData } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { notificationsApi } from "../../api/endpoints";
import type { AppNotification, NotificationsPage } from "../../api/types";
import { keys, useConversations, useNotifications } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, EmptyState, ErrorState, LoadingState, Text } from "../../ui";
import { formatRelative } from "../../utils/format";

export function NotificationsScreen() {
  const { colors, spacing, radii } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const qc = useQueryClient();
  const query = useNotifications();
  const conversations = useConversations(false);
  const items = useMemo(() => query.data?.pages.flatMap((p) => p.notifications) ?? [], [query.data]);
  const unread = query.data?.pages[0]?.unreadCount ?? 0;

  const setRead = (predicate: (n: AppNotification) => boolean) =>
    qc.setQueryData<InfiniteData<NotificationsPage>>(keys.notifications, (data) =>
      data
        ? {
            ...data,
            pages: data.pages.map((p, i) => {
              const changed = p.notifications.filter((n) => !n.isRead && predicate(n)).length;
              return {
                ...p,
                unreadCount: i === 0 ? Math.max(0, p.unreadCount - changed) : p.unreadCount,
                notifications: p.notifications.map((n) => (predicate(n) ? { ...n, isRead: true } : n))
              };
            })
          }
        : data
    );

  const markOne = useMutation({
    mutationFn: (id: string) => notificationsApi.markRead(id),
    onMutate: (id) => setRead((n) => n.id === id)
  });
  const markAll = useMutation({
    mutationFn: notificationsApi.markAllRead,
    onMutate: () => setRead(() => true)
  });

  const open = (n: AppNotification) => {
    if (!n.isRead) {
      markOne.mutate(n.id);
    }
    const conv = conversations.data?.pages.flatMap((p) => p.conversations).find((c) => c.matchId === n.data.matchId);
    if (conv) {
      navigation.navigate("Chat", { matchId: conv.matchId, user: conv.user });
    } else {
      navigation.navigate("Tabs", { screen: "Matches" });
    }
  };

  if (query.isLoading) {
    return <LoadingState />;
  }
  if (query.isError) {
    return <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  }

  return (
    <SafeAreaView
      edges={["top"]}
      style={[styles.flex, { backgroundColor: colors.background }]}
      testID="notifications-screen"
    >
      <FlatList
        data={items}
        keyExtractor={(n) => n.id}
        contentContainerStyle={[styles.list, { padding: spacing.lg, gap: 10 }]}
        refreshControl={
          <RefreshControl
            refreshing={query.isRefetching}
            onRefresh={() => void query.refetch()}
            tintColor={colors.primary}
          />
        }
        onEndReached={() => query.hasNextPage && !query.isFetchingNextPage && void query.fetchNextPage()}
        ListHeaderComponent={
          <View style={styles.header}>
            <Text variant="title" style={styles.flex}>
              Activité
            </Text>
            {unread > 0 ? (
              <Button
                testID="notifications-read-all"
                label="Tout marquer comme lu"
                variant="ghost"
                size="md"
                onPress={() => markAll.mutate()}
              />
            ) : null}
          </View>
        }
        ListEmptyComponent={
          <EmptyState
            icon="notifications-outline"
            title="Rien de neuf"
            message="Tes nouveaux matchs et messages apparaîtront ici."
          />
        }
        renderItem={({ item }) => (
          <Pressable
            testID={`notification-${item.type}`}
            accessibilityRole="button"
            accessibilityLabel={`${item.title}. ${item.body}${item.isRead ? "" : ". Non lu"}`}
            onPress={() => open(item)}
            style={({ pressed }) => [
              styles.item,
              {
                borderRadius: radii.lg,
                backgroundColor: item.isRead ? colors.surface : colors.primarySoft,
                borderColor: item.isRead ? colors.border : colors.primarySoft,
                opacity: pressed ? 0.8 : 1
              }
            ]}
          >
            <View style={[styles.icon, { backgroundColor: item.isRead ? colors.surfaceSunken : colors.primary }]}>
              <Ionicons
                name={item.type === "match" ? "heart" : "chatbubble"}
                size={18}
                color={item.isRead ? colors.textMuted : colors.onPrimary}
              />
            </View>
            <View style={styles.flex}>
              <Text variant="label">{item.title}</Text>
              <Text tone="muted" numberOfLines={2}>
                {item.body}
              </Text>
            </View>
            <Text variant="caption" tone="subtle">
              {formatRelative(item.createdAt)}
            </Text>
          </Pressable>
        )}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  list: { flexGrow: 1, width: "100%", maxWidth: 640, alignSelf: "center" },
  header: { flexDirection: "row", alignItems: "center", marginBottom: 6 },
  item: { flexDirection: "row", alignItems: "center", gap: 12, padding: 14, borderWidth: 1 },
  icon: { width: 38, height: 38, borderRadius: 19, alignItems: "center", justifyContent: "center" }
});
