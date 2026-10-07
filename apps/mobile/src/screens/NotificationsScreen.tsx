import React, { useCallback, useState } from "react";
import { Alert, FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { fetchConversations } from "../api/chat";
import { fetchMatches } from "../api/discover";
import { markAllNotificationsRead, markNotificationRead } from "../api/notifications";
import { EmptyView, ErrorView } from "../shared/feedback";
import { Button, Text, radii, spacing } from "../shared/ui";
import { formatListTime } from "../features/chat/messages";
import { errorMessage } from "../features/errors";
import { useNotificationsQuery } from "../features/hooks";
import { openChat, openUserProfile, useAppNavigation } from "../features/nav";
import { queryKeys } from "../features/queryKeys";
import type { AppNotification, NotificationsPayload, UserSummary } from "../features/types";
import { ScreenHeader } from "../features/ui/ScreenHeader";
import { Skeleton } from "../features/ui/Skeleton";
import { usePalette } from "../features/ui/palette";
import { markAllReadInPayload, markNotificationReadInPayload } from "../realtime/events";

async function resolveChatUser(
  queryClient: QueryClient,
  conversationId: string
): Promise<UserSummary | null> {
  const conversations = await queryClient.ensureQueryData({
    queryKey: queryKeys.conversations,
    queryFn: fetchConversations
  });
  const conversation = conversations.find((c) => c.id === conversationId);
  if (conversation) {
    return conversation.user;
  }
  const matches = await queryClient.fetchQuery({
    queryKey: queryKeys.matches,
    queryFn: fetchMatches,
    staleTime: 10_000
  });
  return matches.find((m) => m.conversationId === conversationId)?.user ?? null;
}

export default function NotificationsScreen() {
  const p = usePalette();
  const navigation = useAppNavigation();
  const queryClient = useQueryClient();
  const query = useNotificationsQuery();
  const [refreshing, setRefreshing] = useState(false);

  const markOne = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onMutate: (id) => {
      queryClient.setQueryData<NotificationsPayload>(queryKeys.notifications, (cur) =>
        cur ? markNotificationReadInPayload(cur, id) : cur
      );
    },
    onError: () => void queryClient.invalidateQueries({ queryKey: queryKeys.notifications })
  });

  const markAll = useMutation({
    mutationFn: markAllNotificationsRead,
    onMutate: () => {
      queryClient.setQueryData<NotificationsPayload>(queryKeys.notifications, (cur) =>
        cur ? markAllReadInPayload(cur) : cur
      );
    },
    onError: (err) => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
      Alert.alert("Couldn't mark all as read", errorMessage(err, "Please try again."));
    }
  });

  const open = useCallback(
    async (n: AppNotification) => {
      if (!n.isRead) {
        markOne.mutate(n.id);
      }
      const conversationId = n.data.conversationId;
      if (conversationId) {
        try {
          const user = await resolveChatUser(queryClient, conversationId);
          if (user) {
            openChat(navigation, conversationId, user);
            return;
          }
        } catch (err) {
          Alert.alert("Couldn't open conversation", errorMessage(err, "Please try again."));
          return;
        }
        Alert.alert("Conversation unavailable", "This conversation no longer exists.");
        return;
      }
      if (n.data.userId) {
        openUserProfile(navigation, { userId: n.data.userId });
      }
    },
    [markOne, navigation, queryClient]
  );

  const refresh = useCallback(async () => {
    setRefreshing(true);
    try {
      await query.refetch();
    } finally {
      setRefreshing(false);
    }
  }, [query]);

  const unread = query.data?.unreadCount ?? 0;
  const now = new Date();

  let body: React.ReactNode;
  if (query.isLoading) {
    body = (
      <View style={styles.pad} testID="notifications-loading">
        {[0, 1, 2, 3, 4].map((i) => (
          <Skeleton key={i} style={{ height: 64, marginBottom: spacing.md, borderRadius: radii.md }} />
        ))}
      </View>
    );
  } else if (query.isError && !query.data) {
    body = (
      <View style={styles.pad}>
        <ErrorView
          message={errorMessage(query.error, "Couldn't load notifications.")}
          onAction={() => void query.refetch()}
          testID="notifications-error"
        />
      </View>
    );
  } else {
    body = (
      <FlatList
        ListEmptyComponent={
          <View style={styles.pad}>
            <EmptyView
              message="Matches and new messages will show up here."
              testID="notifications-empty"
              title="Nothing new yet"
            />
          </View>
        }
        contentContainerStyle={styles.list}
        data={query.data?.items ?? []}
        keyExtractor={(n) => n.id}
        refreshControl={<RefreshControl onRefresh={() => void refresh()} refreshing={refreshing} />}
        renderItem={({ item }) => (
          <Pressable
            accessibilityHint="Opens the related conversation"
            accessibilityLabel={`${item.isRead ? "" : "Unread. "}${item.title}. ${item.body}`}
            accessibilityRole="button"
            onPress={() => void open(item)}
            style={[
              styles.row,
              { backgroundColor: item.isRead ? p.background : p.primarySoft, borderColor: p.border }
            ]}
            testID={`notification-${item.id}`}
          >
            <View style={[styles.dot, { backgroundColor: item.isRead ? "transparent" : p.primary }]} />
            <View style={styles.flex}>
              <Text weight={item.isRead ? "medium" : "bold"}>{item.title}</Text>
              <Text numberOfLines={2} tone="muted">
                {item.body}
              </Text>
            </View>
            <Text tone="muted" variant="caption">
              {formatListTime(item.createdAt, now)}
            </Text>
          </Pressable>
        )}
        testID="notifications-list"
      />
    );
  }

  return (
    <SafeAreaView
      edges={["top"]}
      style={[styles.flex, { backgroundColor: p.background }]}
      testID="notifications-screen"
    >
      <ScreenHeader
        right={
          <Button
            disabled={unread === 0}
            label="Mark all read"
            onPress={() => markAll.mutate()}
            size="sm"
            testID="notifications-mark-all"
            variant="ghost"
          />
        }
        title="Notifications"
      />
      {body}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  pad: { padding: spacing.lg },
  list: { paddingVertical: spacing.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  dot: { width: 10, height: 10, borderRadius: 5 }
});
