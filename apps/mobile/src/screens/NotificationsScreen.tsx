import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { chatApi, notificationsApi } from "../api/endpoints";
import type { AppNotification, Conversation } from "../api/types";
import { errorMessage } from "../lib/errors";
import { formatTimestamp } from "../lib/dates";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { useTheme } from "../theme";
import { Button, EmptyState, ErrorState, LoadingState, Screen, Text } from "../ui";

export function useUnreadCount() {
  return useQuery({
    queryKey: queryKeys.unread,
    queryFn: notificationsApi.unread,
    staleTime: 15_000
  });
}

export function NotificationsScreen() {
  const t = useTheme();
  const qc = useQueryClient();
  const stack = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const q = useInfiniteQuery({
    queryKey: queryKeys.notifications,
    queryFn: ({ pageParam }) => notificationsApi.list(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });
  const items = q.data?.pages.flatMap((p) => p.notifications) ?? [];
  const refreshAll = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.notifications });
    void qc.invalidateQueries({ queryKey: queryKeys.unread });
  };
  const readOne = useMutation({ mutationFn: notificationsApi.markRead, onSuccess: refreshAll });
  const readAll = useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: refreshAll });

  const open = async (n: AppNotification) => {
    if (!n.readAt) readOne.mutate(n.id);
    if (!n.conversationId) return;
    const cached = qc.getQueryData<Conversation[]>(queryKeys.conversations);
    const convos = cached?.some((c) => c.id === n.conversationId)
      ? cached
      : await qc.fetchQuery({ queryKey: queryKeys.conversations, queryFn: chatApi.conversations });
    const c = convos.find((x) => x.id === n.conversationId);
    if (c) {
      stack.navigate("Chat", {
        conversationId: c.id,
        peerId: c.peer.id,
        peerName: c.peer.firstName,
        peerPhotoId: c.peer.mainPhotoId
      });
    }
  };

  if (q.isLoading)
    return (
      <Screen edges={["top"]}>
        <LoadingState />
      </Screen>
    );
  if (q.isError)
    return (
      <Screen edges={["top"]}>
        <ErrorState message={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      </Screen>
    );

  return (
    <Screen edges={["top"]} padded={false}>
      <View style={styles.header}>
        <Text variant="title">Notifications</Text>
        {items.some((n) => !n.readAt) ? (
          <Button
            label="Tout lire"
            variant="ghost"
            onPress={() => readAll.mutate()}
            style={styles.readAll}
          />
        ) : null}
      </View>
      {items.length === 0 ? (
        <EmptyState
          icon="notifications-outline"
          title="Rien pour l'instant"
          message="Vos nouveaux matchs et messages s'afficheront ici."
        />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(n) => n.id}
          refreshControl={
            <RefreshControl
              refreshing={q.isRefetching && !q.isFetchingNextPage}
              onRefresh={() => void q.refetch()}
              tintColor={t.primary}
            />
          }
          onEndReached={() => {
            if (q.hasNextPage && !q.isFetchingNextPage) void q.fetchNextPage();
          }}
          onEndReachedThreshold={0.5}
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              onPress={() => void open(item)}
              style={[
                styles.row,
                {
                  borderColor: t.border,
                  backgroundColor: item.readAt ? "transparent" : t.primarySoft
                }
              ]}
            >
              <View style={[styles.icon, { backgroundColor: t.surface }]}>
                <Ionicons
                  name={item.type === "match" ? "heart" : "chatbubble"}
                  size={20}
                  color={t.primary}
                />
              </View>
              <View style={styles.body}>
                <Text variant="label">{item.title}</Text>
                <Text tone="muted" numberOfLines={2}>
                  {item.body}
                </Text>
              </View>
              <Text variant="caption" tone="muted">
                {formatTimestamp(item.createdAt)}
              </Text>
            </Pressable>
          )}
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
    paddingHorizontal: 20,
    paddingVertical: 8
  },
  readAll: { minHeight: 36, paddingHorizontal: 12 },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    paddingHorizontal: 20,
    paddingVertical: 14,
    borderBottomWidth: StyleSheet.hairlineWidth
  },
  icon: { width: 40, height: 40, borderRadius: 20, alignItems: "center", justifyContent: "center" },
  body: { flex: 1, gap: 2 }
});
